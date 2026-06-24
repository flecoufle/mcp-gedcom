package display

import (
	"fmt"
	"sort"
	"strings"

	"github.com/flecoufle/mcp-gedcom/internal/gedcom"
	"github.com/flecoufle/mcp-gedcom/internal/mcp"
)

func value(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func Person(name, id string) string {
	return fmt.Sprintf("%s (%s)", name, id)
}

func PersonWithBirth(name, id, birth string) string {
	if birth == "" {
		return Person(name, id)
	}
	return fmt.Sprintf("%s (%s), Born: %s", name, id, birth)
}

func PersonShort(ind map[string]interface{}) string {
	var parts []string
	if id, ok := ind["id"]; ok {
		parts = append(parts, "ID: "+value(id))
	}
	if name, ok := ind["name"]; ok {
		parts = append(parts, "Name: "+cleanName(value(name)))
	}
	if sex, ok := ind["sex"]; ok && value(sex) != "" {
		parts = append(parts, "Sex: "+value(sex))
	}
	if date, ok := ind["date"]; ok && value(date) != "" {
		parts = append(parts, "Date: "+value(date))
	} else if birth, ok := ind["birth"]; ok && value(birth) != "" {
		parts = append(parts, "Birth: "+value(birth))
	}
	if death, ok := ind["death"]; ok && value(death) != "" {
		parts = append(parts, "Death: "+value(death))
	}
	if len(parts) == 0 {
		return ""
	}
	return "- " + strings.Join(parts, ", ")
}

func CheckError(result map[string]interface{}) *mcp.CallToolResult {
	if err, ok := result["error"]; ok {
		return mcp.NewCallToolError(value(err))
	}
	return nil
}

func cleanName(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, "\"", ""), "/", "")
}

func WritePerson(sb *strings.Builder, name, id string) {
	sb.WriteString(cleanName(name))
	sb.WriteString(" (")
	sb.WriteString(id)
	sb.WriteString(")")
}

func WritePersonWithBirth(sb *strings.Builder, name, id, birth string) {
	WritePerson(sb, name, id)
	if birth != "" {
		sb.WriteString(", Born: ")
		sb.WriteString(birth)
	}
}

func WritePersonWithDates(sb *strings.Builder, name, id, birth, death string) {
	WritePerson(sb, name, id)
	if birth != "" {
		sb.WriteString(", Born: ")
		sb.WriteString(birth)
	}
	if death != "" {
		sb.WriteString(", Died: ")
		sb.WriteString(death)
	}
}

func SearchResults(title string, results []map[string]interface{}) *mcp.CallToolResult {
	if len(results) == 0 {
		return mcp.NewCallToolError("No " + strings.ToLower(title) + " found")
	}
	if msg, ok := results[0]["message"].(string); ok && strings.Contains(msg, "No individuals found") {
		return mcp.NewCallToolError(msg)
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString(":\n")
	for _, ind := range results {
		sb.WriteString(PersonShort(ind))
		sb.WriteString("\n")
	}
	return mcp.NewCallToolResult(sb.String())
}

func GenResults(title, subjID, subjName string, result map[string]interface{}, genKey, genLabel string) *mcp.CallToolResult {
	if err := CheckError(result); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString(" ")
	sb.WriteString(subjID)
	sb.WriteString(" (")
	sb.WriteString(subjName)
	sb.WriteString("):\n")

	if gens, ok := result[genKey].(map[string][]map[string]interface{}); ok {
		genKeys := make([]string, 0, len(gens))
		for k := range gens {
			genKeys = append(genKeys, k)
		}
		sort.Slice(genKeys, func(i, j int) bool {
			return genKeys[i] < genKeys[j]
		})
		for _, gen := range genKeys {
			sb.WriteString("Generation ")
			sb.WriteString(gen)
			sb.WriteString(":\n")
			for _, item := range gens[gen] {
				sb.WriteString("  - ")
				WritePersonWithDates(&sb, value(item["name"]), value(item["id"]), value(item["birth"]), value(item["death"]))
				if rel := value(item["relationship"]); rel != "" {
					sb.WriteString(", ")
					sb.WriteString(rel)
				}
				sb.WriteString("\n")
			}
		}
		if len(genKeys) == 0 {
			sb.WriteString("- No ")
			sb.WriteString(strings.ToLower(genLabel))
			sb.WriteString(" found\n")
		}
	} else {
		sb.WriteString("- No ")
		sb.WriteString(strings.ToLower(genLabel))
		sb.WriteString(" found\n")
	}

	return mcp.NewCallToolResultWithStructured(sb.String(), result)
}

type RelBlock struct {
	DepthA        int
	DepthB        int
	RelationLabel string
	AncestorIDs   []string
	AncestorCounts map[string]int
}

func GroupIntoBlocks(links []gedcom.RelationshipLink) []RelBlock {
	type key struct {
		da, db int
		label  string
	}
	groups := make(map[key]*RelBlock)
	var keys []key

	for _, link := range links {
		k := key{link.DepthA, link.DepthB, link.RelationLabel}
		if _, ok := groups[k]; !ok {
			groups[k] = &RelBlock{
				DepthA:         link.DepthA,
				DepthB:         link.DepthB,
				RelationLabel:  link.RelationLabel,
				AncestorCounts: make(map[string]int),
			}
			keys = append(keys, k)
		}
		for _, ancID := range link.CommonAncestors {
			groups[k].AncestorCounts[ancID]++
			if groups[k].AncestorCounts[ancID] == 1 {
				groups[k].AncestorIDs = append(groups[k].AncestorIDs, ancID)
			}
		}
	}

	sort.SliceStable(keys, func(i, j int) bool {
		sumI := groups[keys[i]].DepthA + groups[keys[i]].DepthB
		sumJ := groups[keys[j]].DepthA + groups[keys[j]].DepthB
		if sumI != sumJ {
			return sumI < sumJ
		}
		if groups[keys[i]].DepthA != groups[keys[j]].DepthA {
			return groups[keys[i]].DepthA < groups[keys[j]].DepthA
		}
		return groups[keys[i]].DepthB < groups[keys[j]].DepthB
	})

	blocks := make([]RelBlock, 0, len(keys))
	for _, k := range keys {
		blocks = append(blocks, *groups[k])
	}
	return blocks
}

func DeName(name string) string {
	if name == "" {
		return "d'inconnu"
	}
	switch name[0] {
	case 'A', 'E', 'I', 'O', 'U', 'Y',
		'a', 'e', 'i', 'o', 'u', 'y',
		'Â', 'â', 'É', 'é', 'È', 'è', 'Ê', 'ê',
		'Î', 'î', 'Ô', 'ô', 'Û', 'û':
		return "d'" + name
	case 'H', 'h':
		return "d'" + name
	default:
		return "de " + name
	}
}
