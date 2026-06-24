package gedcom

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	gdcom "github.com/iand/gedcom"
)

const maxPathsPerAncestor = 50

type AncestorEntry struct {
	Path  []string
	Depth int
}

type RelationshipLink struct {
	CommonAncestors []string
	PathFromA       []string
	PathFromB       []string
	DepthA          int
	DepthB          int
	RelationLabel   string
	Signature       string
}

func NormalizeID(id string) string {
	id = strings.TrimPrefix(id, "@")
	id = strings.TrimSuffix(id, "@")
	return id
}

func pathContainsID(path []string, id string) bool {
	for i := 0; i < len(path)-1; i++ {
		if path[i] == id {
			return true
		}
	}
	return false
}

type bfsEntry struct {
	currentID string
	path      []string
	depth     int
}

func GetAllAncestors(loader *Loader, individualID string, maxDepth int) map[string][]AncestorEntry {
	result := make(map[string][]AncestorEntry)
	if maxDepth <= 0 {
		return result
	}

	cleanID := NormalizeID(individualID)
	ind := loader.Individual(cleanID)
	if ind == nil {
		return result
	}

	result[cleanID] = append(result[cleanID], AncestorEntry{
		Path:  []string{cleanID},
		Depth: 0,
	})

	queue := []bfsEntry{
		{currentID: cleanID, path: []string{cleanID}, depth: 0},
	}
	// Track visited path prefixes per ancestor to allow multiple paths
	visitedPaths := make(map[string]map[string]bool)
	visitedPaths[cleanID] = map[string]bool{cleanID: true}

	for len(queue) > 0 {
		entry := queue[0]
		queue = queue[1:]

		if entry.depth >= maxDepth {
			continue
		}

		ind := loader.Individual(entry.currentID)
		if ind == nil {
			continue
		}

		for _, parentLink := range ind.Parents {
			if parentLink.Family == nil {
				continue
			}

			if parentLink.Type != "" && parentLink.Type != "birth" {
				continue
			}

			fam := parentLink.Family

			parents := make([]*gdcom.IndividualRecord, 0, 2)
			if fam.Husband != nil {
				parents = append(parents, fam.Husband)
			}
			if fam.Wife != nil {
				parents = append(parents, fam.Wife)
			}

			for _, parent := range parents {
				if pathContainsID(entry.path, parent.Xref) {
					continue
				}

				newPath := make([]string, len(entry.path), len(entry.path)+1)
				copy(newPath, entry.path)
				newPath = append(newPath, parent.Xref)

				pathKey := strings.Join(newPath, ",")

				if visitedPaths[parent.Xref] != nil && visitedPaths[parent.Xref][pathKey] {
					continue
				}

				if visitedPaths[parent.Xref] == nil {
					visitedPaths[parent.Xref] = make(map[string]bool)
				}
				visitedPaths[parent.Xref][pathKey] = true

				ancestorEntry := AncestorEntry{
					Path:  newPath,
					Depth: entry.depth + 1,
				}
				result[parent.Xref] = append(result[parent.Xref], ancestorEntry)

				if len(result[parent.Xref]) <= maxPathsPerAncestor {
					queue = append(queue, bfsEntry{
						currentID: parent.Xref,
						path:      newPath,
						depth:     entry.depth + 1,
					})
				}
			}
		}
	}

	return result
}

func computeSignature(pathA, pathB []string) string {
	keyA := strings.Join(pathA, ",")
	keyB := strings.Join(pathB, ",")
	if keyA > keyB {
		keyA, keyB = keyB, keyA
	}
	h := sha256.Sum256([]byte(keyA + "|" + keyB))
	return fmt.Sprintf("%x", h)
}

func areMarried(loader *Loader, id1, id2 string) bool {
	for _, fam := range loader.Families() {
		if fam.Husband != nil && fam.Wife != nil {
			hID := NormalizeID(fam.Husband.Xref)
			wID := NormalizeID(fam.Wife.Xref)
			if (hID == id1 && wID == id2) || (hID == id2 && wID == id1) {
				return true
			}
		}
	}
	return false
}


func pathPrefix(path []string) string {
	if len(path) <= 1 {
		return ""
	}
	return strings.Join(path[:len(path)-1], ",")
}

func filterShadowedLinks(links []RelationshipLink) []RelationshipLink {
	if len(links) <= 1 {
		return links
	}

	sorted := make([]RelationshipLink, len(links))
	copy(sorted, links)
	sort.SliceStable(sorted, func(i, j int) bool {
		sumI := sorted[i].DepthA + sorted[i].DepthB
		sumJ := sorted[j].DepthA + sorted[j].DepthB
		if sumI != sumJ {
			return sumI < sumJ
		}
		if sorted[i].DepthA != sorted[j].DepthA {
			return sorted[i].DepthA < sorted[j].DepthA
		}
		return sorted[i].DepthB < sorted[j].DepthB
	})

	shadowed := make(map[int]bool)
	for i := 0; i < len(sorted); i++ {
		if shadowed[i] {
			continue
		}
		ancI := sorted[i].CommonAncestors[0]
		for j := i + 1; j < len(sorted); j++ {
			if shadowed[j] {
				continue
			}
			ancJ := sorted[j].CommonAncestors[0]
			if ancI == ancJ {
				continue
			}

			// ancI must be at least as close on both sides
			if sorted[i].DepthA > sorted[j].DepthA || sorted[i].DepthB > sorted[j].DepthB {
				continue
			}

			// Check if ancI appears as intermediate in ancJ's path from A or B
			inPath := false
			for k := 0; k < len(sorted[j].PathFromA)-1; k++ {
				if sorted[j].PathFromA[k] == ancI {
					inPath = true
					break
				}
			}
			if !inPath {
				for k := 0; k < len(sorted[j].PathFromB)-1; k++ {
					if sorted[j].PathFromB[k] == ancI {
						inPath = true
						break
					}
				}
			}

			if inPath {
				shadowed[j] = true
			}
		}
	}

	var result []RelationshipLink
	for i, link := range sorted {
		if !shadowed[i] {
			result = append(result, link)
		}
	}
	return result
}

func FindAllRelationships(loader *Loader, idA, idB string, maxDepth int, maxResults int) []RelationshipLink {
	cleanA := NormalizeID(idA)
	cleanB := NormalizeID(idB)

	if cleanA == cleanB {
		return nil
	}

	indA := loader.Individual(cleanA)
	indB := loader.Individual(cleanB)
	if indA == nil || indB == nil {
		return nil
	}
	sex := indA.Sex

	ancestorsA := GetAllAncestors(loader, cleanA, maxDepth)
	ancestorsB := GetAllAncestors(loader, cleanB, maxDepth)

	var links []RelationshipLink
	seen := make(map[string]bool)

	for ancID, pathsA := range ancestorsA {
		pathsB, ok := ancestorsB[ancID]
		if !ok {
			continue
		}

		for _, pa := range pathsA {
			for _, pb := range pathsB {
				sig := computeSignature(pa.Path, pb.Path)
				if seen[sig] {
					continue
				}
				seen[sig] = true

				labelSex := sex
				if pb.Depth == 0 {
					labelSex = indB.Sex
				}
				links = append(links, RelationshipLink{
					CommonAncestors: []string{ancID},
					PathFromA:       pa.Path,
					PathFromB:       pb.Path,
					DepthA:          pa.Depth,
					DepthB:          pb.Depth,
					RelationLabel:   ComputeRelationLabel(pa.Depth, pb.Depth, labelSex),
					Signature:       sig,
				})
			}
		}
	}

	links = filterShadowedLinks(links)

	if maxResults > 0 && len(links) > maxResults {
		sort.SliceStable(links, func(i, j int) bool {
			sumI := links[i].DepthA + links[i].DepthB
			sumJ := links[j].DepthA + links[j].DepthB
			if sumI != sumJ {
				return sumI < sumJ
			}
			if links[i].DepthA != links[j].DepthA {
				return links[i].DepthA < links[j].DepthA
			}
			return links[i].DepthB < links[j].DepthB
		})
		links = links[:maxResults]
	}

	links = groupCouples(loader, links)

	sort.SliceStable(links, func(i, j int) bool {
		sumI := links[i].DepthA + links[i].DepthB
		sumJ := links[j].DepthA + links[j].DepthB
		if sumI != sumJ {
			return sumI < sumJ
		}
		if links[i].DepthA != links[j].DepthA {
			return links[i].DepthA < links[j].DepthA
		}
		return links[i].DepthB < links[j].DepthB
	})

	return links
}

type groupKey struct {
	depthA  int
	depthB  int
	prefixA string
	prefixB string
}

func groupCouples(loader *Loader, links []RelationshipLink) []RelationshipLink {
	groups := make(map[groupKey][]int)
	for i, link := range links {
		if len(link.CommonAncestors) != 1 {
			continue
		}
		key := groupKey{
			depthA:  link.DepthA,
			depthB:  link.DepthB,
			prefixA: pathPrefix(link.PathFromA),
			prefixB: pathPrefix(link.PathFromB),
		}
		groups[key] = append(groups[key], i)
	}

	merged := make(map[int]bool)
	var result []RelationshipLink

	for _, indices := range groups {
		if len(indices) < 2 {
			continue
		}

		for i := 0; i < len(indices); i++ {
			if merged[indices[i]] {
				continue
			}
			li := links[indices[i]]
			if len(li.CommonAncestors) != 1 {
				continue
			}

			foundSpouse := false
			for j := i + 1; j < len(indices); j++ {
				if merged[indices[j]] {
					continue
				}
				lj := links[indices[j]]
				if len(lj.CommonAncestors) != 1 {
					continue
				}

				ancI := li.CommonAncestors[0]
				ancJ := lj.CommonAncestors[0]

				if areMarried(loader, ancI, ancJ) {
					merged[indices[i]] = true
					merged[indices[j]] = true

					result = append(result, RelationshipLink{
						CommonAncestors: []string{ancI, ancJ},
						PathFromA:       li.PathFromA,
						PathFromB:       li.PathFromB,
						DepthA:          li.DepthA,
						DepthB:          li.DepthB,
						RelationLabel:   li.RelationLabel,
						Signature:       li.Signature,
					})
					foundSpouse = true
					break
				}
			}

			if !foundSpouse {
				result = append(result, li)
				merged[indices[i]] = true
			}
		}
	}

	for i, link := range links {
		if merged[i] {
			continue
		}
		result = append(result, link)
	}

	return result
}
