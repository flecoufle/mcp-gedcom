package tools

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/flecoufle/mcp-gedcom/internal/gedcom"
	gdcom "github.com/iand/gedcom"
)

var surnamePattern = regexp.MustCompile(`/([^/]+)/`)

func resolveNameInMap(m map[string]interface{}) {
	usage, _ := m["usage_name"].(string)
	surname, _ := m["surname"].(string)
	if usage != "" {
		if surname != "" {
			m["name"] = usage + " " + surname
		} else {
			m["name"] = usage
		}
	}
}

func extractSurname(n *gdcom.NameRecord) string {
	// use n.NamePieceSurname
	if surname := strings.TrimSpace(n.NamePieceSurname); surname != "" {
		return surname
	}

	// use n.Name
	matches := surnamePattern.FindStringSubmatch(n.Name)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func HandleSearchPerson(pattern string, birthYear int, useGiven bool) ([]map[string]interface{}, error) {
	if strings.TrimSpace(pattern) == "" {
		return nil, fmt.Errorf("missing required parameter: pattern")
	}
	patternClean := strings.ReplaceAll(pattern, "\"", "")
	words := strings.Fields(strings.ToLower(patternClean))
	var results []map[string]interface{}

	for _, ind := range gedcom.Get().Individuals() {
		if len(ind.Name) > 0 {
			fullName := gedcom.CleanName(ind.Name[0].Name)
			fullNameLower := strings.ToLower(fullName)

			allWordsMatch := true
			for _, word := range words {
				if word != "" && !strings.Contains(fullNameLower, word) {
					allWordsMatch = false
					break
				}
			}

			if allWordsMatch {
				summary := gedcom.MakePersonSummary(ind)

				if birthYear > 0 {
					if birthStr, ok := summary["birth"].(string); ok && birthStr != "" {
						year := extractYear(birthStr)
						if year < birthYear-2 || year > birthYear+2 {
							continue
						}
					}
				}

				results = append(results, summary)
			}
		}
	}

	if !useGiven {
		for i := range results {
			resolveNameInMap(results[i])
		}
	}

	if len(results) == 0 {
		return []map[string]interface{}{{"message": "No individuals found matching '" + pattern + "'"}}, nil
	}

	return results, nil
}

func extractYear(dateStr string) int {
	dateStr = strings.ToUpper(dateStr)
	re := regexp.MustCompile(`\b(\d{4})\b`)
	matches := re.FindStringSubmatch(dateStr)
	if len(matches) > 1 {
		year, _ := strconv.Atoi(matches[1])
		return year
	}
	return 0
}

func collectSiblingGroup(fam *gdcom.FamilyRecord, cleanID string, groupType, label string, seenSiblings map[string]bool) map[string]interface{} {
	hasFather := fam.Husband != nil
	hasMother := fam.Wife != nil
	parents := map[string]interface{}{}
	if hasFather {
		parents["father_id"] = fam.Husband.Xref
	}
	if hasMother {
		parents["mother_id"] = fam.Wife.Xref
	}
	var list []interface{}
	for _, child := range fam.Child {
		if child.Xref == cleanID || seenSiblings[child.Xref] {
			continue
		}
		seenSiblings[child.Xref] = true
		list = append(list, gedcom.PersonMapWithBirth(child))
	}
	sg := map[string]interface{}{
		"id":      fam.Xref,
		"type":    groupType,
		"label":   label,
		"parents": parents,
	}
	if len(list) > 0 {
		sg["list"] = list
	}
	return sg
}

func buildEvents(ind *gdcom.IndividualRecord) []map[string]string {
	if ind.Event == nil {
		return nil
	}
	events := []map[string]string{}
	for _, e := range ind.Event {
		event := map[string]string{"type": e.Tag}
		if e.Date != "" {
			event["date"] = e.Date
		}
		if e.Place.Name != "" {
			event["place"] = e.Place.Name
		}
		if e.Cause != "" {
			event["cause"] = e.Cause
		}
		if e.Age != "" {
			event["age"] = e.Age
		}
		events = append(events, event)
	}
	return events
}

func buildNotes(ind *gdcom.IndividualRecord) []string {
	if len(ind.Note) == 0 {
		return nil
	}
	notes := []string{}
	for _, n := range ind.Note {
		if n.Note != "" {
			notes = append(notes, n.Note)
		}
	}
	if len(notes) == 0 {
		return nil
	}
	return notes
}

func buildAttributes(ind *gdcom.IndividualRecord) []map[string]string {
	if ind.Attribute == nil {
		return nil
	}
	attributes := []map[string]string{}
	for _, a := range ind.Attribute {
		if a.Tag != "OCCU" && a.Tag != "RESI" {
			continue
		}
		attr := map[string]string{"type": a.Tag}
		if a.Value != "" {
			attr["value"] = a.Value
		}
		if a.Date != "" {
			attr["date"] = a.Date
		}
		if a.Place.Name != "" {
			attr["place"] = a.Place.Name
		}
		attributes = append(attributes, attr)
	}
	if len(attributes) == 0 {
		return nil
	}
	return attributes
}

func buildAssociations(ind *gdcom.IndividualRecord, useGiven bool) []map[string]interface{} {
	if len(ind.Association) == 0 {
		return nil
	}
	assocs := []map[string]interface{}{}
	for _, a := range ind.Association {
		assoc := map[string]interface{}{"relation": a.Relation}
		if a.Xref != "" {
			assoc["ref"] = a.Xref
			cleanID := normalizeID(a.Xref)
			if strings.HasPrefix(cleanID, "F") {
				if fam := gedcom.Get().Family(cleanID); fam != nil {
					names := []string{}
					if fam.Husband != nil && len(fam.Husband.Name) > 0 {
						names = append(names, gedcom.ResolveName(fam.Husband.Name[0].Name, useGiven))
					}
					if fam.Wife != nil && len(fam.Wife.Name) > 0 {
						names = append(names, gedcom.ResolveName(fam.Wife.Name[0].Name, useGiven))
					}
					if len(names) > 0 {
						assoc["name"] = strings.Join(names, " & ")
					}
					for _, e := range fam.Event {
						if e.Date != "" {
							assoc["date"] = e.Date
							break
						}
					}
				}
			} else {
				if target := gedcom.Get().Individual(cleanID); target != nil && len(target.Name) > 0 {
					assoc["name"] = gedcom.CleanName(target.Name[0].Name)
					assoc["sex"] = target.Sex
					assoc["given_name"] = gedcom.ExtractGivenName(target.Name[0].Name)
					assoc["usage_name"] = gedcom.ExtractUsageName(target.Name[0].Name)
					assoc["surname"] = gedcom.ExtractSurname(target.Name[0].Name)
				}
			}
		}
		if a.Phrase != "" {
			assoc["phrase"] = a.Phrase
		}
		assocs = append(assocs, assoc)
	}
	sort.SliceStable(assocs, func(i, j int) bool {
		di, oki := assocs[i]["date"].(string)
		dj, okj := assocs[j]["date"].(string)
		if oki && okj {
			return extractYear(di) < extractYear(dj)
		}
		return oki && !okj
	})
	return assocs
}

func buildGodchildren(ind *gdcom.IndividualRecord) []map[string]interface{} {
	godchildren := gedcom.Get().GetReverseAssociations(ind.Xref)
	if len(godchildren) == 0 {
		return nil
	}
	gcs := []map[string]interface{}{}
	for _, gc := range godchildren {
		child := gedcom.Get().Individual(gc.SourceXref)
		if child == nil {
			continue
		}
		rel := "godchild"
		if child.Sex == "M" {
			rel = "godson"
		} else if child.Sex == "F" {
			rel = "goddaughter"
		}
		entry := map[string]interface{}{
			"id":         child.Xref,
			"name":       gedcom.CleanName(child.Name[0].Name),
			"sex":        child.Sex,
			"relation":   rel,
			"given_name": gedcom.ExtractGivenName(child.Name[0].Name),
			"usage_name": gedcom.ExtractUsageName(child.Name[0].Name),
			"surname":    gedcom.ExtractSurname(child.Name[0].Name),
		}
		if child.Event != nil {
			date := ""
			for _, e := range child.Event {
				if (e.Tag == "BAPM" || e.Tag == "BAPT") && e.Date != "" {
					date = e.Date
					break
				}
			}
			if date == "" {
				for _, e := range child.Event {
					if e.Tag == "BIRT" && e.Date != "" {
						date = e.Date
						break
					}
				}
			}
			if date != "" {
				entry["date"] = date
			}
		}
		gcs = append(gcs, entry)
	}
	sort.SliceStable(gcs, func(i, j int) bool {
		di, oki := gcs[i]["date"].(string)
		dj, okj := gcs[j]["date"].(string)
		if oki && okj {
			return extractYear(di) < extractYear(dj)
		}
		return oki && !okj
	})
	return gcs
}

func buildFamilies(ind *gdcom.IndividualRecord, cleanID string, withSpouse, withChildren bool, useGiven bool) []map[string]interface{} {
	if !withSpouse && !withChildren {
		return nil
	}
	var families []map[string]interface{}
	for _, fl := range ind.Family {
		if fl.Family == nil {
			continue
		}
		familyID := fl.Family.Xref
		familyRecord := gedcom.Get().Family(familyID)
		familyData := map[string]interface{}{
			"id": familyID,
		}
		if familyRecord != nil && familyRecord.NumberOfChildren != "" {
			familyData["number_of_children"] = familyRecord.NumberOfChildren
		}
		if familyRecord != nil {
			if withSpouse {
				if spouse := gedcom.SpouseInFamily(ind, familyRecord); spouse != nil {
					familyData["spouse"] = gedcom.PersonMapWithBirth(spouse)
				}
			}
			if withChildren {
				var children []map[string]interface{}
				for _, child := range familyRecord.Child {
					cm := gedcom.ChildMap(child, familyID)
					var spouseFams []map[string]interface{}
					for _, childFl := range child.Family {
						if childFl.Family == nil {
							continue
						}
						spouse := gedcom.SpouseInFamily(child, childFl.Family)
						sf := map[string]interface{}{
							"family_id": childFl.Family.Xref,
						}
						if spouse != nil && len(spouse.Name) > 0 {
							sf["spouse_name"] = gedcom.ResolveName(spouse.Name[0].Name, useGiven)
						}
						spouseFams = append(spouseFams, sf)
					}
					if len(spouseFams) > 0 {
						cm["spouse_families"] = spouseFams
					}
					children = append(children, cm)
				}
				if len(children) > 0 {
					familyData["children"] = children
				}
			}
		}
		if len(familyData) > 1 {
			families = append(families, familyData)
		}
	}
	return families
}

func buildAncestorFamilies(ind *gdcom.IndividualRecord) []map[string]interface{} {
	var ancestorFamilies []map[string]interface{}
	for _, parentLink := range ind.Parents {
		if parentLink.Family == nil {
			continue
		}
		fam := parentLink.Family
		fd := map[string]interface{}{
			"id":             fam.Xref,
			"children_count": len(fam.Child),
		}
		if fam.Husband != nil {
			fd["father"] = gedcom.PersonMapWithBirth(fam.Husband)
		}
		if fam.Wife != nil {
			fd["mother"] = gedcom.PersonMapWithBirth(fam.Wife)
		}
		ancestorFamilies = append(ancestorFamilies, fd)
	}
	return ancestorFamilies
}

func HandleGetPersonDetails(id string, withSpouse, withChildren bool, useGiven bool) (map[string]interface{}, error) {
	cleanID := normalizeID(id)

	ind := gedcom.Get().Individual(cleanID)
	if ind == nil {
		return map[string]interface{}{"error": "Individual not found: " + id}, nil
	}

	result := map[string]interface{}{
		"id":   ind.Xref,
		"name": gedcom.CleanName(ind.Name[0].Name),
		"sex":  ind.Sex,
	}

	if len(ind.Name) > 0 {
		result["given_name"] = gedcom.ExtractGivenName(ind.Name[0].Name)
		result["usage_name"] = gedcom.ExtractUsageName(ind.Name[0].Name)
		result["surname"] = gedcom.ExtractSurname(ind.Name[0].Name)
	}

	var siblingsGroups []map[string]interface{}
	seenSiblings := map[string]bool{}
	processedFamilies := map[string]bool{}

	for _, parentLink := range ind.Parents {
		if parentLink.Family == nil {
			continue
		}
		fam := parentLink.Family
		processedFamilies[fam.Xref] = true

		hasFather := fam.Husband != nil
		hasMother := fam.Wife != nil
		groupType := "full"
		label := "same father and same mother"
		if hasFather && !hasMother {
			groupType = "paternal_half"
			label = "same father"
		} else if !hasFather && hasMother {
			groupType = "maternal_half"
			label = "same mother"
		}
		sg := collectSiblingGroup(fam, cleanID, groupType, label, seenSiblings)
		if list, ok := sg["list"].([]interface{}); ok && len(list) > 0 {
			siblingsGroups = append(siblingsGroups, sg)
		}
	}

	fatherIDs := map[string]bool{}
	motherIDs := map[string]bool{}
	for _, parentLink := range ind.Parents {
		if parentLink.Family == nil {
			continue
		}
		if parentLink.Family.Husband != nil {
			fatherIDs[parentLink.Family.Husband.Xref] = true
		}
		if parentLink.Family.Wife != nil {
			motherIDs[parentLink.Family.Wife.Xref] = true
		}
	}

	for fID := range fatherIDs {
		father := gedcom.Get().Individual(fID)
		if father == nil {
			continue
		}
		for _, fl := range father.Family {
			if fl.Family == nil || processedFamilies[fl.Family.Xref] {
				continue
			}
			fam := fl.Family
			processedFamilies[fam.Xref] = true
			sg := collectSiblingGroup(fam, cleanID, "paternal_half", "same father", seenSiblings)
			if list, ok := sg["list"].([]interface{}); ok && len(list) > 0 {
				siblingsGroups = append(siblingsGroups, sg)
			}
		}
	}

	for mID := range motherIDs {
		mother := gedcom.Get().Individual(mID)
		if mother == nil {
			continue
		}
		for _, fl := range mother.Family {
			if fl.Family == nil || processedFamilies[fl.Family.Xref] {
				continue
			}
			fam := fl.Family
			processedFamilies[fam.Xref] = true
			sg := collectSiblingGroup(fam, cleanID, "maternal_half", "same mother", seenSiblings)
			if list, ok := sg["list"].([]interface{}); ok && len(list) > 0 {
				siblingsGroups = append(siblingsGroups, sg)
			}
		}
	}

	if len(siblingsGroups) > 0 {
		result["siblings_groups"] = siblingsGroups
	}

	if events := buildEvents(ind); events != nil {
		result["events"] = events
	}

	if notes := buildNotes(ind); notes != nil {
		result["notes"] = notes
	}

	if attributes := buildAttributes(ind); attributes != nil {
		result["attributes"] = attributes
	}

	if assocs := buildAssociations(ind, useGiven); assocs != nil {
		result["associations"] = assocs
	}

	if gcs := buildGodchildren(ind); gcs != nil {
		result["godchildren"] = gcs
	}

	if families := buildFamilies(ind, cleanID, withSpouse, withChildren, useGiven); families != nil {
		result["families"] = families
	}

	if ancestorFamilies := buildAncestorFamilies(ind); ancestorFamilies != nil {
		result["ancestor_families"] = ancestorFamilies
	}

	if !useGiven {
		resolveNameInMap(result)

		if sgs, ok := result["siblings_groups"].([]map[string]interface{}); ok {
			for _, sg := range sgs {
				if list, ok := sg["list"].([]interface{}); ok {
					for _, e := range list {
						if m, ok := e.(map[string]interface{}); ok {
							resolveNameInMap(m)
						}
					}
				}
			}
		}

		if families, ok := result["families"].([]map[string]interface{}); ok {
			for _, fam := range families {
				if spouse, ok := fam["spouse"].(map[string]interface{}); ok {
					resolveNameInMap(spouse)
				}
				if children, ok := fam["children"].([]map[string]interface{}); ok {
					for i := range children {
						resolveNameInMap(children[i])
					}
				}
			}
		}

		if assocs, ok := result["associations"].([]map[string]interface{}); ok {
			for _, a := range assocs {
				if _, hasSex := a["sex"]; hasSex {
					resolveNameInMap(a)
				}
			}
		}

		if godchildren, ok := result["godchildren"].([]map[string]interface{}); ok {
			for _, gc := range godchildren {
				resolveNameInMap(gc)
			}
		}

		if afs, ok := result["ancestor_families"].([]map[string]interface{}); ok {
			for _, af := range afs {
				if father, ok := af["father"].(map[string]interface{}); ok {
					resolveNameInMap(father)
				}
				if mother, ok := af["mother"].(map[string]interface{}); ok {
					resolveNameInMap(mother)
				}
			}
		}
	}

	return result, nil
}

func HandleGetRelatives(id string, useGiven bool) (map[string]interface{}, error) {
	cleanID := normalizeID(id)

	ind := gedcom.Get().Individual(cleanID)
	if ind == nil {
		return map[string]interface{}{"error": "Individual not found: " + id}, nil
	}

	result := map[string]interface{}{
		"id":         ind.Xref,
		"name":       gedcom.CleanName(ind.Name[0].Name),
		"given_name": gedcom.ExtractGivenName(ind.Name[0].Name),
		"usage_name": gedcom.ExtractUsageName(ind.Name[0].Name),
		"surname":    gedcom.ExtractSurname(ind.Name[0].Name),
		"relatives":  getRelatives(ind),
	}

	if !useGiven {
		resolveNameInMap(result)
	}

	return result, nil
}

func getRelatives(ind *gdcom.IndividualRecord) map[string][]string {
	relatives := make(map[string][]string)

	for _, fl := range ind.Family {
		if fl.Family != nil {
			relatives["spouse"] = append(relatives["spouse"], fl.Family.Xref)
		}
	}

	for _, fl := range ind.Parents {
		if fl.Family != nil {
			relatives["parents"] = append(relatives["parents"], fl.Family.Xref)
		}
	}

	return relatives
}

type neighborInfo struct {
	Individual *gdcom.IndividualRecord
	Link       string // "spouse", "parent", "child"
}

type pathNode struct {
	ID        string
	Name      string
	Link      string // "spouse", "parent", "child" from the calculated node in the path
	HaveA     string // only for printing
	IsMeeting bool
}

type bfsState struct {
	queue   []*gdcom.IndividualRecord
	visited map[string]string
	links   map[string]string
	depth   map[string]int
}

func HandleFindRelationshipPath(id1, id2 string, ancestorsOnly bool, useGiven bool) (map[string]interface{}, error) {
	cleanID1 := normalizeID(id1)
	cleanID2 := normalizeID(id2)

	ind1 := gedcom.Get().Individual(cleanID1)
	ind2 := gedcom.Get().Individual(cleanID2)

	if ind1 == nil {
		return map[string]interface{}{"error": "Individual not found: " + id1}, nil
	}
	if ind2 == nil {
		return map[string]interface{}{"error": "Individual not found: " + id2}, nil
	}

	noSpouse := ancestorsOnly
	noChildren := ancestorsOnly
	path, _, found := bidirectionalBFS(ind1, ind2, noSpouse, noChildren)

	name1 := ""
	name2 := ""
	if len(ind1.Name) > 0 {
		name1 = gedcom.ResolveName(ind1.Name[0].Name, useGiven)
	}
	if len(ind2.Name) > 0 {
		name2 = gedcom.ResolveName(ind2.Name[0].Name, useGiven)
	}

	result := map[string]interface{}{
		"id1":   ind1.Xref,
		"id2":   ind2.Xref,
		"name1": name1,
		"name2": name2,
	}

	if !found {
		msg := "no relationship found"
		if noSpouse && noChildren {
			msg += " (searched only ancestors)"
		} else if !noSpouse {
			msg += " (searched with spouses)"
		} else if !noChildren {
			msg += " (searched with children)"
		}

		result["relationship"] = msg
		result["path"] = []map[string]interface{}{}
		return result, nil
	}

	relation := calculateRelationship(path)
	result["relationship"] = relation

	meetingIdx := -1
	for i, node := range path {
		if node.IsMeeting {
			meetingIdx = i
			break
		}
	}

	pathMaps := []map[string]interface{}{}
	for i, node := range path {
		name := node.Name
		if !useGiven {
			if ind := gedcom.Get().Individual(node.ID); ind != nil && len(ind.Name) > 0 {
				name = gedcom.ResolveName(ind.Name[0].Name, false)
			}
		}
		m := map[string]interface{}{
			"id":        node.ID,
			"name":      name,
			"haveA":     node.HaveA,
			"isMeeting": node.IsMeeting,
			"link_is":   node.Link,
		}
		if node.Link != "" && meetingIdx >= 0 {
			if i <= meetingIdx && i > 0 {
				m["link_of_id"] = path[i-1].ID
			} else if i > meetingIdx && i < len(path)-1 {
				m["link_of_id"] = path[i+1].ID
			}
		}
		pathMaps = append(pathMaps, m)
	}
	result["path"] = pathMaps

	return result, nil
}

func bidirectionalBFS(start, target *gdcom.IndividualRecord, noSpouse bool, noChildren bool) ([]pathNode, string, bool) {
	if start.Xref == target.Xref {
		name := ""
		if len(start.Name) > 0 {
			name = gedcom.CleanName(start.Name[0].Name)
		}
		return []pathNode{{ID: start.Xref, Name: name, Link: "", HaveA: ""}}, "", true
	}

	state1 := &bfsState{
		queue:   []*gdcom.IndividualRecord{start},
		visited: map[string]string{start.Xref: ""},
		links:   map[string]string{},
		depth:   map[string]int{start.Xref: 0},
	}
	state2 := &bfsState{
		queue:   []*gdcom.IndividualRecord{target},
		visited: map[string]string{target.Xref: ""},
		links:   map[string]string{},
		depth:   map[string]int{target.Xref: 0},
	}

	for len(state1.queue) > 0 || len(state2.queue) > 0 {
		if len(state1.queue) > 0 {
			curr := state1.queue[0]
			state1.queue = state1.queue[1:]

			for _, nb := range getNeighbors(curr, noSpouse, noChildren) {
				if _, exists := state1.visited[nb.Individual.Xref]; !exists {
					state1.visited[nb.Individual.Xref] = curr.Xref
					state1.links[nb.Individual.Xref] = nb.Link
					state1.depth[nb.Individual.Xref] = state1.depth[curr.Xref] + 1
					state1.queue = append(state1.queue, nb.Individual)
				}

				if _, exists := state2.visited[nb.Individual.Xref]; exists {
					return reconstructPath(state1, state2, nb.Individual.Xref), nb.Individual.Xref, true
				}
			}
		}

		if len(state2.queue) > 0 {
			curr := state2.queue[0]
			state2.queue = state2.queue[1:]

			for _, nb := range getNeighbors(curr, noSpouse, noChildren) {
				if _, exists := state2.visited[nb.Individual.Xref]; !exists {
					state2.visited[nb.Individual.Xref] = curr.Xref
					state2.links[nb.Individual.Xref] = nb.Link
					state2.depth[nb.Individual.Xref] = state2.depth[curr.Xref] + 1
					state2.queue = append(state2.queue, nb.Individual)
				}

				if _, exists := state1.visited[nb.Individual.Xref]; exists {
					return reconstructPath(state1, state2, nb.Individual.Xref), nb.Individual.Xref, true
				}
			}
		}
	}

	return nil, "", false
}

func getNeighbors(ind *gdcom.IndividualRecord, noSpouse bool, noChildren bool) []neighborInfo {
	neighbors := []neighborInfo{}
	seen := make(map[string]bool)

	// Parents: ind.Parents → FamilyRecord → Husband + Wife
	for _, fl := range ind.Parents {
		if fl.Family == nil {
			continue
		}
		fam := gedcom.Get().Family(fl.Family.Xref)
		if fam == nil {
			continue
		}
		if fam.Husband != nil && !seen[fam.Husband.Xref] {
			neighbors = append(neighbors, neighborInfo{Individual: fam.Husband, Link: "parent"})
			seen[fam.Husband.Xref] = true
		}
		if fam.Wife != nil && !seen[fam.Wife.Xref] {
			neighbors = append(neighbors, neighborInfo{Individual: fam.Wife, Link: "parent"})
			seen[fam.Wife.Xref] = true
		}
	}

	// Spouses and children via Family
	for _, fl := range ind.Family {
		if fl.Family == nil {
			continue
		}
		fam := gedcom.Get().Family(fl.Family.Xref)
		if fam == nil {
			continue
		}

		// Spouse (skip if noSpouse is true)
		if !noSpouse {
			var spouse *gdcom.IndividualRecord
			if ind.Sex == "M" && fam.Wife != nil {
				spouse = fam.Wife
			} else if ind.Sex == "F" && fam.Husband != nil {
				spouse = fam.Husband
			} else if fam.Husband != nil && fam.Husband.Xref != ind.Xref {
				spouse = fam.Husband
			} else if fam.Wife != nil && fam.Wife.Xref != ind.Xref {
				spouse = fam.Wife
			}

			if spouse != nil && !seen[spouse.Xref] {
				neighbors = append(neighbors, neighborInfo{Individual: spouse, Link: "spouse"})
				seen[spouse.Xref] = true
			}
		}

		// Children (skip if noChildren is true)
		if !noChildren {
			for _, child := range fam.Child {
				if child.Xref != ind.Xref && !seen[child.Xref] {
					neighbors = append(neighbors, neighborInfo{Individual: child, Link: "child"})
					seen[child.Xref] = true
				}
			}
		}
	}

	return neighbors
}

func reconstructPath(state1, state2 *bfsState, meetingNode string) []pathNode {
	// Path from start → meetingNode (using state1)
	path := []pathNode{}
	for id := meetingNode; id != ""; id = state1.visited[id] {
		ind := gedcom.Get().Individual(id)
		name := ""
		if ind != nil && len(ind.Name) > 0 {
			name = gedcom.CleanName(ind.Name[0].Name)
		}
		originalLink := state1.links[id]
		link := invertLink(originalLink)
		isMeeting := false
		if id == meetingNode {
			isMeeting = true
		}
		path = append([]pathNode{{ID: id, Name: name, Link: originalLink, HaveA: link, IsMeeting: isMeeting}}, path...)
	}

	// Path from meetingNode → target (using state2, exclusive of meetingNode)
	// state2.visited[meetingNode] gives the parent of meetingNode in state2's traversal
	// We want to add: parent, grandparent, ..., target

	// Add the meeting node itself with the correct link and isMeeting=true
	ind := gedcom.Get().Individual(meetingNode)
	name := ""
	if ind != nil && len(ind.Name) > 0 {
		name = gedcom.CleanName(ind.Name[0].Name)
	}
	link := state2.links[meetingNode]
	path = append(path, pathNode{ID: meetingNode, Name: name, Link: link, HaveA: link, IsMeeting: false})

	for id := state2.visited[meetingNode]; id != ""; id = state2.visited[id] {
		ind := gedcom.Get().Individual(id)
		name := ""
		if ind != nil && len(ind.Name) > 0 {
			name = gedcom.CleanName(ind.Name[0].Name)
		}
		link := state2.links[id]
		path = append(path, pathNode{ID: id, Name: name, Link: link, HaveA: link})
	}

	return path
}

func invertLink(link string) string {
	switch link {
	case "child":
		return "parent"
	case "parent":
		return "child"
	default:
		return link // "spouse" stays "spouse"
	}
}

func calculateRelationship(path []pathNode) string {
	if len(path) < 2 {
		return "same person"
	}

	if len(path) < 3 {
		return "error in path calculation: the meeting node is counted twice in the path"
	}

	meetingIdx := -1
	for i, node := range path {
		if node.IsMeeting {
			meetingIdx = i
			break
		}
	}
	if meetingIdx < 0 {
		return "error: no meeting node found in path"
	}

	// Ancestors of id1 (walk forward from path[1] to meeting)
	ancestors1 := make(map[string]int)
	gen := 1
	for i := 1; i <= meetingIdx; i++ {
		switch path[i].HaveA {
		case "child":
			ancestors1[path[i].ID] = gen
			gen++
		case "parent":
			gen--
		}
	}

	// Ancestors of id2 (walk backward from just before id2 toward meeting)
	ancestors2 := make(map[string]int)
	gen = 1
	for i := len(path) - 2; i >= meetingIdx+1; i-- {
		switch path[i].HaveA {
		case "parent":
			ancestors2[path[i].ID] = gen
			gen++
		case "child":
			gen--
		}
	}

	// Find closest common ancestor (min gen1+gen2)
	commonID := ""
	commonGen1, commonGen2 := 0, 0
	for id, d1 := range ancestors1 {
		if d2, ok := ancestors2[id]; ok {
			if commonID == "" || d1+d2 < commonGen1+commonGen2 {
				commonID = id
				commonGen1, commonGen2 = d1, d2
			}
		}
	}

	if commonID == "" {
		// Fallback to current generic message
		pathLen := len(path) - 2
		switch pathLen {
		case 1:
			return "yes relationship found: directly related"
		case 2:
			return "yes relationship found: related through one intermediary"
		default:
			return fmt.Sprintf("yes relationship found: related (path length: %d)", pathLen)
		}
	}

	// Build descriptive string with relationship names
	ind := gedcom.Get().Individual(commonID)
	base := "father"
	if ind != nil && ind.Sex == "F" {
		base = "mother"
	}

	ancestorName := commonID
	if ind != nil && len(ind.Name) > 0 {
		ancestorName = gedcom.FormatNameWithID(ind.Name[0].Name, commonID)
	}

	rel1 := getRelationship(commonGen1, base)
	rel2 := getRelationship(commonGen2, base)

	name1 := gedcom.FormatNameWithID(path[0].Name, path[0].ID)
	name2 := gedcom.FormatNameWithID(path[len(path)-1].Name, path[len(path)-1].ID)

	return fmt.Sprintf(
		"yes relationship found: common ancestor: %s. %s → ancestor: %s (%d gen). %s → ancestor: %s (%d gen).",
		ancestorName, name1, rel1, commonGen1, name2, rel2, commonGen2,
	)
}

func normalizeFamilyID(id string) string {
	id = strings.TrimPrefix(id, "@")
	id = strings.TrimSuffix(id, "@")
	if !strings.HasPrefix(id, "F") {
		id = "F" + id
	}
	return id
}

func extractCityCountry(place string) (city, country string) {
	parts := strings.Split(place, ",")
	city = strings.TrimSpace(parts[0])
	country = strings.TrimSpace(parts[len(parts)-1])
	return
}

var monthNames = map[string]int{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
	"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
}

func gedcomDateToSortKey(dateStr string) int {
	dateStr = strings.ToUpper(strings.TrimSpace(dateStr))
	for _, prefix := range []string{"BEF ", "AFT ", "ABT ", "CAL ", "EST ", "INT "} {
		if strings.HasPrefix(dateStr, prefix) {
			dateStr = strings.TrimSpace(dateStr[len(prefix):])
			break
		}
	}
	if strings.HasPrefix(dateStr, "FROM ") {
		dateStr = strings.TrimSpace(dateStr[5:])
		if idx := strings.Index(dateStr, " TO "); idx >= 0 {
			dateStr = strings.TrimSpace(dateStr[:idx])
		}
	}
	re := regexp.MustCompile(`(\d{1,2})\s+([A-Z]+)\s+(\d{4})`)
	if m := re.FindStringSubmatch(dateStr); m != nil {
		day, _ := strconv.Atoi(m[1])
		month := monthNames[strings.ToLower(m[2])]
		year, _ := strconv.Atoi(m[3])
		return year*10000 + month*100 + day
	}
	re2 := regexp.MustCompile(`([A-Z]+)\s+(\d{4})`)
	if m := re2.FindStringSubmatch(dateStr); m != nil {
		month := monthNames[strings.ToLower(m[1])]
		year, _ := strconv.Atoi(m[2])
		return year*10000 + month*100
	}
	re3 := regexp.MustCompile(`\b(\d{4})\b`)
	if m := re3.FindStringSubmatch(dateStr); m != nil {
		year, _ := strconv.Atoi(m[1])
		return year * 10000
	}
	return 0
}

func HandleGetFamilyDetails(familyID string, useGiven bool) (map[string]interface{}, error) {
	cleanID := normalizeFamilyID(familyID)

	familyRecord := gedcom.Get().Family(cleanID)
	if familyRecord == nil {
		return map[string]interface{}{"error": "Family not found: " + familyID}, nil
	}

	result := map[string]interface{}{
		"id": familyRecord.Xref,
	}

	if familyRecord.Husband != nil {
		result["husband"] = gedcom.PersonMapWithBirth(familyRecord.Husband)
	}

	if familyRecord.Wife != nil {
		result["wife"] = gedcom.PersonMapWithBirth(familyRecord.Wife)
	}

	var marriage, divorce map[string]interface{}
	for _, event := range familyRecord.Event {
		if event.Tag == "MARR" && marriage == nil {
			marriage = map[string]interface{}{}
			if event.Date != "" {
				marriage["date"] = event.Date
			}
			if event.Place.Name != "" {
				marriage["place"] = event.Place.Name
			}
		} else if event.Tag == "DIV" && divorce == nil {
			divorce = map[string]interface{}{}
			if event.Date != "" {
				divorce["date"] = event.Date
			}
			if event.Place.Name != "" {
				divorce["place"] = event.Place.Name
			}
		}
	}
	if marriage != nil {
		result["marriage"] = marriage
	}
	if divorce != nil {
		result["divorce"] = divorce
	}

	var children []map[string]interface{}
	for _, child := range familyRecord.Child {
		children = append(children, gedcom.PersonMapWithBirth(child))
	}
	if len(children) > 0 {
		result["children"] = children
	}
	result["child_count"] = len(children)

	// Build geographic timeline
	type tlEvent struct {
		sortKey int
		date    string
		city    string
		country string
		label   string
	}
	var events []tlEvent

	addEvent := func(ind *gdcom.IndividualRecord, tag, date, place, label string) {
		if date == "" || place == "" {
			return
		}
		city, country := extractCityCountry(place)
		events = append(events, tlEvent{
			sortKey: gedcomDateToSortKey(date),
			date:    date,
			city:    city,
			country: country,
			label:   label,
		})
	}

	if familyRecord.Husband != nil {
		for _, e := range familyRecord.Husband.Event {
			if e.Tag == "BIRT" {
				addEvent(familyRecord.Husband, "BIRT", e.Date, e.Place.Name, "Birth of "+gedcom.ResolveName(familyRecord.Husband.Name[0].Name, useGiven))
			}
			if e.Tag == "DEAT" {
				addEvent(familyRecord.Husband, "DEAT", e.Date, e.Place.Name, "Death of "+gedcom.ResolveName(familyRecord.Husband.Name[0].Name, useGiven))
			}
		}
	}
	if familyRecord.Wife != nil {
		for _, e := range familyRecord.Wife.Event {
			if e.Tag == "BIRT" {
				addEvent(familyRecord.Wife, "BIRT", e.Date, e.Place.Name, "Birth of "+gedcom.ResolveName(familyRecord.Wife.Name[0].Name, useGiven))
			}
			if e.Tag == "DEAT" {
				addEvent(familyRecord.Wife, "DEAT", e.Date, e.Place.Name, "Death of "+gedcom.ResolveName(familyRecord.Wife.Name[0].Name, useGiven))
			}
		}
	}
	if marriage != nil {
		if d, ok := marriage["date"].(string); ok {
			if p, ok2 := marriage["place"].(string); ok2 {
				city, country := extractCityCountry(p)
				events = append(events, tlEvent{
					sortKey: gedcomDateToSortKey(d),
					date:    d,
					city:    city,
					country: country,
					label:   "Marriage",
				})
			}
		}
	}
	if divorce != nil {
		if d, ok := divorce["date"].(string); ok {
			if p, ok2 := divorce["place"].(string); ok2 {
				city, country := extractCityCountry(p)
				events = append(events, tlEvent{
					sortKey: gedcomDateToSortKey(d),
					date:    d,
					city:    city,
					country: country,
					label:   "Divorce",
				})
			}
		}
	}
	for _, child := range familyRecord.Child {
		if child.Event != nil {
			for _, e := range child.Event {
				if e.Tag == "BIRT" && e.Date != "" && e.Place.Name != "" {
					city, country := extractCityCountry(e.Place.Name)
					events = append(events, tlEvent{
						sortKey: gedcomDateToSortKey(e.Date),
						date:    e.Date,
						city:    city,
						country: country,
						label:   "Birth of " + gedcom.ResolveName(child.Name[0].Name, useGiven),
					})
				}
			}
		}
	}

	sort.SliceStable(events, func(i, j int) bool {
		return events[i].sortKey < events[j].sortKey
	})

	var segments []interface{}
	for _, ev := range events {
		if len(segments) > 0 {
			last := segments[len(segments)-1].(map[string]interface{})
			if last["city"] == ev.city && last["country"] == ev.country {
				last["to_date"] = ev.date
				last["events"] = append(last["events"].([]interface{}), map[string]interface{}{"date": ev.date, "label": ev.label})
				continue
			}
		}
		seg := map[string]interface{}{
			"from_date": ev.date,
			"to_date":   ev.date,
			"city":      ev.city,
			"country":   ev.country,
			"events":    []interface{}{map[string]interface{}{"date": ev.date, "label": ev.label}},
		}
		segments = append(segments, seg)
	}
	if len(segments) > 0 {
		result["timeline"] = segments
	}

	var notes []string
	for _, note := range familyRecord.Note {
		if note.Note != "" {
			notes = append(notes, note.Note)
		}
	}
	if len(notes) > 0 {
		result["notes"] = notes
	}

	if !useGiven {
		if husband, ok := result["husband"].(map[string]interface{}); ok {
			resolveNameInMap(husband)
		}
		if wife, ok := result["wife"].(map[string]interface{}); ok {
			resolveNameInMap(wife)
		}
		if children, ok := result["children"].([]map[string]interface{}); ok {
			for i := range children {
				resolveNameInMap(children[i])
			}
		}
	}

	return result, nil
}

func HandleSearchByDateRange(startYear, endYear int, event string, useGiven bool) ([]map[string]interface{}, error) {
	eventTag := "BIRT"
	if strings.ToLower(event) == "death" {
		eventTag = "DEAT"
	}

	var results []map[string]interface{}

	for _, ind := range gedcom.Get().Individuals() {
		if ind.Event == nil {
			continue
		}

		for _, e := range ind.Event {
			if e.Tag != eventTag || e.Date == "" {
				continue
			}

			var year int
			if len(e.Date) >= 4 {
				y, err := strconv.Atoi(e.Date[len(e.Date)-4:])
				if err != nil {
					continue
				}
				year = y
			}

			if year >= startYear && year <= endYear {
			entry := gedcom.PersonMapWithBirth(ind)
			entry["date"] = e.Date
			results = append(results, entry)
			}
			break
		}
	}

	if !useGiven {
		for i := range results {
			resolveNameInMap(results[i])
		}
	}

	if len(results) == 0 {
		return []map[string]interface{}{{"message": "No individuals found in date range"}}, nil
	}

	return results, nil
}

func normalizeID(id string) string {
	id = strings.TrimPrefix(id, "@")
	id = strings.TrimSuffix(id, "@")
	return id
}

func HandleGetChildren(id string, offset, limit int, useGiven bool) (map[string]interface{}, error) {
	cleanID := normalizeID(id)

	ind := gedcom.Get().Individual(cleanID)
	if ind == nil {
		return map[string]interface{}{"error": "Individual not found: " + id}, nil
	}

	var families []map[string]interface{}

	for _, fl := range ind.Family {
		if fl.Family == nil {
			continue
		}

		familyID := fl.Family.Xref
		familyRecord := gedcom.Get().Family(familyID)

		familyData := map[string]interface{}{
			"id":       familyID,
			"spouse":   (*map[string]interface{})(nil),
			"children": []map[string]interface{}{},
		}

		if familyRecord != nil {
			if spouse := gedcom.SpouseInFamily(ind, familyRecord); spouse != nil {
				familyData["spouse"] = gedcom.PersonMapWithBirth(spouse)
			}

			var children []map[string]interface{}
			for _, child := range familyRecord.Child {
				children = append(children, gedcom.ChildMap(child, familyID))
			}

			if len(children) > 0 {
				familyData["children"] = children
			}
		}

		families = append(families, familyData)
	}

	total := len(families)

	if offset >= total {
		families = []map[string]interface{}{}
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		families = families[offset:end]
	}

	result := map[string]interface{}{
		"id":         ind.Xref,
		"name":       gedcom.CleanName(ind.Name[0].Name),
		"families":   families,
		"pagination": gedcom.MakePagination(offset, limit, total),
	}

	if !useGiven {
		resolveNameInMap(result)
		for _, fam := range families {
			if spouse, ok := fam["spouse"].(map[string]interface{}); ok {
				resolveNameInMap(spouse)
			}
			if children, ok := fam["children"].([]map[string]interface{}); ok {
				for i := range children {
					resolveNameInMap(children[i])
				}
			}
		}
	}

	return result, nil
}

func HandleSearchSurnames(pattern string, offset, limit int) (map[string]interface{}, error) {
	if strings.TrimSpace(pattern) == "" {
		return nil, fmt.Errorf("missing required parameter: pattern")
	}

	surnameCounts := make(map[string]int)

	// Loop through all individuals and count surnames
	// Note: This is not the most efficient way
	for _, ind := range gedcom.Get().Individuals() {
		for _, n := range ind.Name {
			surname := extractSurname(n)
			if surname != "" {
				surnameCounts[surname]++
			}
		}
	}

	var matchedSurnames []map[string]interface{}
	patternClean := strings.ReplaceAll(pattern, "\"", "")
	patternLower := strings.ToLower(patternClean)
	for surname, count := range surnameCounts {
		if strings.Contains(strings.ToLower(surname), patternLower) {
			matchedSurnames = append(matchedSurnames, map[string]interface{}{
				"name":  surname,
				"count": count,
			})
		}
	}

	sort.Slice(matchedSurnames, func(i, j int) bool {
		return matchedSurnames[i]["name"].(string) < matchedSurnames[j]["name"].(string)
	})

	total := len(matchedSurnames)

	if offset >= total {
		matchedSurnames = []map[string]interface{}{}
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		matchedSurnames = matchedSurnames[offset:end]
	}

	return map[string]interface{}{
		"pattern":  pattern,
		"surnames": matchedSurnames,
		"pagination": map[string]interface{}{
			"offset": offset,
			"limit":  limit,
			"total":  total,
		},
	}, nil
}

func HandleGetParents(id string, useGiven bool) (map[string]interface{}, error) {
	cleanID := normalizeID(id)

	ind := gedcom.Get().Individual(cleanID)
	if ind == nil {
		return map[string]interface{}{"error": "Individual not found: " + id}, nil
	}

	var families []map[string]interface{}

	for _, fl := range ind.Parents {
		if fl.Family == nil {
			continue
		}

		familyID := fl.Family.Xref
		familyRecord := gedcom.Get().Family(familyID)

		familyData := map[string]interface{}{
			"family_id": familyID,
			"type":      nil,
			"father":    (*map[string]interface{})(nil),
			"mother":    (*map[string]interface{})(nil),
		}

		if fl.Type != "" && fl.Type != "birth" {
			familyData["type"] = fl.Type
		}

		if familyRecord != nil {
			for _, p := range []*gdcom.IndividualRecord{familyRecord.Husband, familyRecord.Wife} {
				if p != nil {
					parentData := gedcom.PersonMapWithBirth(p)

					switch p.Sex {
					case "M":
						familyData["father"] = parentData
					case "F":
						familyData["mother"] = parentData
					default:
						familyData["parent"] = parentData
					}
				}
			}
		}

		families = append(families, familyData)
	}

	result := map[string]interface{}{
		"id":       ind.Xref,
		"name":     gedcom.CleanName(ind.Name[0].Name),
		"families": families,
	}

	if !useGiven {
		resolveNameInMap(result)
		for _, fam := range families {
			for _, key := range []string{"father", "mother", "parent"} {
				if m, ok := fam[key].(map[string]interface{}); ok {
					resolveNameInMap(m)
				}
			}
		}
	}

	return result, nil
}

func HandleGetAncestors(id string, generations int, useGiven bool) (map[string]interface{}, error) {
	cleanID := normalizeID(id)

	ind := gedcom.Get().Individual(cleanID)
	if ind == nil {
		return map[string]interface{}{"error": "Individual not found: " + id}, nil
	}

	if generations <= 0 {
		generations = 3
	}

	ancestorsByGeneration := make(map[string][]map[string]interface{})
	seenIDs := make(map[string]bool)

	collectAncestors(ind, 1, generations, ancestorsByGeneration, seenIDs)

	result := map[string]interface{}{
		"id":          ind.Xref,
		"name":        gedcom.CleanName(ind.Name[0].Name),
		"generations": generations,
		"ancestors":   ancestorsByGeneration,
	}

	if !useGiven {
		resolveNameInMap(result)
		for _, list := range ancestorsByGeneration {
			for i := range list {
				resolveNameInMap(list[i])
			}
		}
	}

	return result, nil
}

func collectAncestors(ind *gdcom.IndividualRecord, currentGen, maxGen int, result map[string][]map[string]interface{}, seen map[string]bool) {
	if currentGen > maxGen {
		return
	}

	for _, fl := range ind.Parents {
		if fl.Family == nil {
			continue
		}

		familyID := fl.Family.Xref
		familyRecord := gedcom.Get().Family(familyID)
		if familyRecord == nil {
			continue
		}

		if familyRecord.Husband != nil && !seen[familyRecord.Husband.Xref] {
			seen[familyRecord.Husband.Xref] = true
			ancestor := gedcom.PersonMapWithBirth(familyRecord.Husband)
			ancestor["relationship"] = getRelationship(currentGen, "father")
			ancestor["family_id"] = familyID
			genKey := fmt.Sprintf("%d", currentGen)
			result[genKey] = append(result[genKey], ancestor)
			collectAncestors(familyRecord.Husband, currentGen+1, maxGen, result, seen)
		}

		if familyRecord.Wife != nil && !seen[familyRecord.Wife.Xref] {
			seen[familyRecord.Wife.Xref] = true
			ancestor := gedcom.PersonMapWithBirth(familyRecord.Wife)
			ancestor["relationship"] = getRelationship(currentGen, "mother")
			ancestor["family_id"] = familyID
			genKey := fmt.Sprintf("%d", currentGen)
			result[genKey] = append(result[genKey], ancestor)
			collectAncestors(familyRecord.Wife, currentGen+1, maxGen, result, seen)
		}
	}
}

func getRelationship(generation int, base string) string {
	switch generation {
	case 1:
		return base
	case 2:
		if base == "father" {
			return "grandfather"
		}
		return "grandmother"
	case 3:
		if base == "father" {
			return "great-grandfather"
		}
		return "great-grandmother"
	default:
		prefix := ""
		for i := 3; i < generation; i++ {
			prefix += "great-"
		}
		if base == "father" {
			return prefix + "grandfather"
		}
		return prefix + "grandmother"
	}
}

func HandleGetDescendants(id string, generations int, useGiven bool) (map[string]interface{}, error) {
	cleanID := normalizeID(id)

	ind := gedcom.Get().Individual(cleanID)
	if ind == nil {
		return map[string]interface{}{"error": "Individual not found: " + id}, nil
	}

	if generations <= 0 {
		generations = 3
	}

	descendantsByGeneration := make(map[string][]map[string]interface{})
	seenIDs := make(map[string]bool)

	collectDescendants(ind, 1, generations, descendantsByGeneration, seenIDs)

	result := map[string]interface{}{
		"id":          ind.Xref,
		"name":        gedcom.CleanName(ind.Name[0].Name),
		"generations": generations,
		"descendants": descendantsByGeneration,
	}

	if !useGiven {
		resolveNameInMap(result)
		for _, list := range descendantsByGeneration {
			for i := range list {
				resolveNameInMap(list[i])
			}
		}
	}

	return result, nil
}

func collectDescendants(ind *gdcom.IndividualRecord, currentGen, maxGen int, result map[string][]map[string]interface{}, seen map[string]bool) {
	if currentGen > maxGen {
		return
	}

	for _, fl := range ind.Family {
		if fl.Family == nil {
			continue
		}

		familyRecord := gedcom.Get().Family(fl.Family.Xref)
		if familyRecord == nil {
			continue
		}

		for _, child := range familyRecord.Child {
			if seen[child.Xref] {
				continue
			}
			seen[child.Xref] = true

			sex := "child"
			if child.Sex == "M" {
				sex = "son"
			} else if child.Sex == "F" {
				sex = "daughter"
			}

			descendant := gedcom.PersonMapWithBirth(child)
			descendant["relationship"] = getDescendantRelationship(currentGen, sex)
			descendant["family_id"] = fl.Family.Xref

			genKey := fmt.Sprintf("%d", currentGen)
			result[genKey] = append(result[genKey], descendant)

			collectDescendants(child, currentGen+1, maxGen, result, seen)
		}
	}
}

func getDescendantRelationship(generation int, base string) string {
	switch generation {
	case 1:
		return base
	case 2:
		if base == "son" {
			return "grandson"
		}
		if base == "daughter" {
			return "granddaughter"
		}
		return "grandchild"
	default:
		prefix := ""
		for i := 2; i < generation; i++ {
			prefix += "great-"
		}
		if base == "son" {
			return prefix + "grandson"
		}
		if base == "daughter" {
			return prefix + "granddaughter"
		}
		return prefix + "grandchild"
	}
}

func HandleGetStatistics() map[string]interface{} {
	individuals := gedcom.Get().Individuals()
	families := gedcom.Get().Families()

	return map[string]interface{}{
		"total_individuals": len(individuals),
		"total_families":    len(families),
	}
}

func HandleLoadGedcomFile(path string) (map[string]interface{}, error) {
	if err := gedcom.ReloadFile(path); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message":           "GEDCOM file loaded successfully",
		"path":              path,
		"total_individuals": len(gedcom.Get().Individuals()),
		"total_families":    len(gedcom.Get().Families()),
	}, nil
}

func HandleFindAllRelationships(id1, id2 string, maxDepth int, maxResults int) ([]gedcom.RelationshipLink, float64, error) {
	cleanID1 := gedcom.NormalizeID(id1)
	cleanID2 := gedcom.NormalizeID(id2)

	ind1 := gedcom.Get().Individual(cleanID1)
	if ind1 == nil {
		return nil, 0, fmt.Errorf("individual not found: " + id1)
	}
	ind2 := gedcom.Get().Individual(cleanID2)
	if ind2 == nil {
		return nil, 0, fmt.Errorf("individual not found: " + id2)
	}

	links := gedcom.FindAllRelationships(gedcom.Get(), cleanID1, cleanID2, maxDepth, maxResults)
	coeff := gedcom.ComputeKinshipCoefficient(links)

	return links, coeff, nil
}
