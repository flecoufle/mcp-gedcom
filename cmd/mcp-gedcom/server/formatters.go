package main

import (
	"fmt"
	"strings"

	"github.com/flecoufle/mcp-gedcom/internal/display"
	"github.com/flecoufle/mcp-gedcom/internal/gedcom"
	"github.com/flecoufle/mcp-gedcom/internal/mcp"
)

func formatSearchByNameResult(result []map[string]interface{}) *mcp.CallToolResult {
	return display.SearchResults("Search results", result)
}

func formatSearchSurnamesResult(result map[string]interface{}) *mcp.CallToolResult {
	var sb strings.Builder
	sb.WriteString("Surnames matching \"")
	sb.WriteString(getString(result["pattern"]))
	sb.WriteString("\":\n")

	structuredSurnames := []map[string]interface{}{}

	if surnames, ok := result["surnames"].([]map[string]interface{}); ok && len(surnames) > 0 {
		for _, s := range surnames {
			name := getString(s["name"])
			var count int
			switch v := s["count"].(type) {
			case int:
				count = v
			case int64:
				count = int(v)
			}
			sb.WriteString("- ")
			sb.WriteString(name)
			sb.WriteString(" (")
			fmt.Fprintf(&sb, "%d", count)
			sb.WriteString(")\n")
			structuredSurnames = append(structuredSurnames, map[string]interface{}{
				"name":  name,
				"count": count,
			})
		}
	} else {
		sb.WriteString("- No surnames found\n")
	}

	var pagination map[string]interface{}
	if p, ok := result["pagination"].(map[string]interface{}); ok {
		var offset, limit, total int
		switch v := p["offset"].(type) {
		case int:
			offset = v
		case int64:
			offset = int(v)
		}
		switch v := p["limit"].(type) {
		case int:
			limit = v
		case int64:
			limit = int(v)
		}
		switch v := p["total"].(type) {
		case int:
			total = v
		case int64:
			total = int(v)
		}
		pagination = map[string]interface{}{
			"offset": offset,
			"limit":  limit,
			"total":  total,
		}
		fmt.Fprintf(&sb, "\nPagination: %d-%d of %d total\n", offset, limit, total)
	}

	structuredResult := map[string]interface{}{
		"pattern":    result["pattern"],
		"surnames":   structuredSurnames,
		"pagination": pagination,
	}

	return mcp.NewCallToolResultWithStructured(sb.String(), structuredResult)
}

func formatGetPersonalDetailsResult(result map[string]interface{}) *mcp.CallToolResult {
	if err := display.CheckError(result); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("Individual details:\n- ID: ")
	sb.WriteString(getString(result["id"]))
	sb.WriteString("\n- Name: ")
	sb.WriteString(getString(result["name"]))
	if sex, ok := result["sex"].(string); ok && sex != "" {
		sb.WriteString("\n- Sex: ")
		sb.WriteString(sex)
	}

	if given, ok := result["given_name"].(string); ok && given != "" {
		sb.WriteString("\n- Given name: ")
		sb.WriteString(given)
	}

	if sgs, ok := result["siblings_groups"].([]map[string]interface{}); ok {
		for _, sg := range sgs {
			fmt.Fprintf(&sb, "\n- %s", getString(sg["label"]))
			if id := getString(sg["id"]); id != "" {
				fmt.Fprintf(&sb, " [%s]", id)
			}
			if parents, ok := sg["parents"].(map[string]interface{}); ok {
				parts := []string{}
				if fid := getString(parents["father_id"]); fid != "" {
					parts = append(parts, "father: "+fid)
				}
				if mid := getString(parents["mother_id"]); mid != "" {
					parts = append(parts, "mother: "+mid)
				}
				if len(parts) > 0 {
					sb.WriteString(" (" + strings.Join(parts, ", ") + ")")
				}
			}
			if list, ok := sg["list"].([]interface{}); ok {
				for _, sI := range list {
					entry, _ := sI.(map[string]interface{})
					if entry == nil {
						continue
					}
					sb.WriteString("\n  - ")
					display.WritePersonWithDates(&sb,
						getString(entry["name"]), getString(entry["id"]),
						getString(entry["birth"]), getString(entry["death"]))
				}
			}
		}
	}

	if events, ok := result["events"].([]map[string]string); ok && len(events) > 0 {
		for _, event := range events {
			eventType := getString(event["type"])
			if eventType == "" {
				continue
			}
			sb.WriteString("\n- ")
			sb.WriteString(eventType)
			if date := getString(event["date"]); date != "" {
				sb.WriteString(": ")
				sb.WriteString(date)
			}
			if age := getString(event["age"]); age != "" {
				sb.WriteString(", age: ")
				sb.WriteString(age)
			}
			if place := getString(event["place"]); place != "" {
				sb.WriteString(" (")
				sb.WriteString(place)
				sb.WriteString(")")
			}
			if cause := getString(event["cause"]); cause != "" {
				sb.WriteString(" (cause: ")
				sb.WriteString(cause)
				sb.WriteString(")")
			}
		}
	}

	if notes, ok := result["notes"].([]string); ok && len(notes) > 0 {
		sb.WriteString("\n- Notes: ")
		sb.WriteString(getString(notes[0]))
		if len(notes) > 1 {
			sb.WriteString("...")
		}
	}

	if attrs, ok := result["attributes"].([]map[string]string); ok && len(attrs) > 0 {
		for _, a := range attrs {
			sb.WriteString("\n- ")
			sb.WriteString(getString(a["type"]))
			if v := getString(a["value"]); v != "" {
				sb.WriteString(": ")
				sb.WriteString(v)
			}
			if d := getString(a["date"]); d != "" {
				sb.WriteString(" (")
				sb.WriteString(d)
				if p := getString(a["place"]); p != "" {
					sb.WriteString(", ")
					sb.WriteString(p)
				}
				sb.WriteString(")")
			} else if p := getString(a["place"]); p != "" {
				sb.WriteString(" (")
				sb.WriteString(p)
				sb.WriteString(")")
			}
		}
	}

	if assocs, ok := result["associations"].([]map[string]interface{}); ok && len(assocs) > 0 {
		for _, a := range assocs {
			sb.WriteString("\n- Association: ")
			sb.WriteString(getString(a["relation"]))
			sb.WriteString(": ")
			if n := getString(a["name"]); n != "" {
				sb.WriteString(n)
				sb.WriteString(" (")
				sb.WriteString(getString(a["ref"]))
				sb.WriteString(")")
			} else if p := getString(a["phrase"]); p != "" {
				sb.WriteString(p)
			} else if r := getString(a["ref"]); r != "" {
				sb.WriteString(r)
				sb.WriteString(" (unresolved)")
			} else {
				sb.WriteString("(details unavailable)")
			}
			if d := getString(a["date"]); d != "" {
				sb.WriteString(" - ")
				sb.WriteString(d)
			}
		}
	}

	if gcs, ok := result["godchildren"].([]map[string]interface{}); ok && len(gcs) > 0 {
		for _, gc := range gcs {
			sb.WriteString("\n- ")
			sb.WriteString(getString(gc["relation"]))
			sb.WriteString(": ")
			sb.WriteString(getString(gc["name"]))
			sb.WriteString(" (")
			sb.WriteString(getString(gc["id"]))
			sb.WriteString(")")
			if d := getString(gc["date"]); d != "" {
				sb.WriteString(" - ")
				sb.WriteString(d)
			}
		}
	}

	if families, ok := result["families"].([]map[string]interface{}); ok && len(families) > 0 {
		for _, fam := range families {
			familyID := getString(fam["id"])
			sb.WriteString("\n- Family ")
			sb.WriteString(familyID)
			sb.WriteString(":")

			if nc, ok := fam["number_of_children"].(string); ok && nc != "" {
				sb.WriteString("\n    - Number of children: ")
				sb.WriteString(nc)
			}

			if spouse, ok := fam["spouse"].(map[string]interface{}); ok && spouse != nil {
				sb.WriteString("\n  - Spouse: ")
				sb.WriteString(getString(spouse["name"]))
				sb.WriteString(" (")
				sb.WriteString(getString(spouse["id"]))
				sb.WriteString(")")
			}

			if children, ok := fam["children"].([]map[string]interface{}); ok && len(children) > 0 {
				sb.WriteString("\n  - Children:")
				for _, child := range children {
					sb.WriteString("\n    - ")
					display.WritePersonWithDates(&sb, getString(child["name"]), getString(child["id"]), getString(child["birth"]), getString(child["death"]))
					if pedigree, ok := child["pedigree"].(string); ok && pedigree != "" {
						sb.WriteString(" [")
						sb.WriteString(pedigree)
						sb.WriteString("]")
					}
					if sfs, ok := child["spouse_families"].([]map[string]interface{}); ok {
						for _, sf := range sfs {
							sb.WriteString(" -> Spouse: ")
							sb.WriteString(getString(sf["spouse_name"]))
							sb.WriteString(" (")
							sb.WriteString(getString(sf["family_id"]))
							sb.WriteString(")")
						}
					}
				}
			}
		}
	}

	if afamilies, ok := result["ancestor_families"].([]map[string]interface{}); ok {
		for _, af := range afamilies {
			fmt.Fprintf(&sb, "\n- Ancestor family %s:", getString(af["id"]))
			if father, ok := af["father"].(map[string]interface{}); ok {
				fmt.Fprintf(&sb, "\n  - Father: %s (%s)",
					getString(father["name"]), getString(father["id"]))
			}
			if mother, ok := af["mother"].(map[string]interface{}); ok {
				fmt.Fprintf(&sb, "\n  - Mother: %s (%s)",
					getString(mother["name"]), getString(mother["id"]))
			}
			fmt.Fprintf(&sb, "\n  - Children: %v", af["children_count"])
		}
	}

	return mcp.NewCallToolResultWithStructured(sb.String(), result)
}

func formatGetRelativesResult(result map[string]interface{}) *mcp.CallToolResult {
	if err := display.CheckError(result); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("Relatives of ")
	sb.WriteString(getString(result["id"]))
	sb.WriteString(" (")
	sb.WriteString(getString(result["name"]))
	sb.WriteString("):\n")

	if relatives, ok := result["relatives"].(map[string][]string); ok {
		if spouses, ok := relatives["spouse"]; ok && len(spouses) > 0 {
			sb.WriteString("- Spouses (families): ")
			for i, f := range spouses {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(f)
			}
			sb.WriteString("\n")
		}

		if parents, ok := relatives["parents"]; ok && len(parents) > 0 {
			sb.WriteString("- Parents (families): ")
			for i, f := range parents {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(f)
			}
			sb.WriteString("\n")
		}
	}

	return mcp.NewCallToolResult(sb.String())
}

func formatGetFamilyByIDResult(result map[string]interface{}) *mcp.CallToolResult {
	if err := display.CheckError(result); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("Family ")
	sb.WriteString(getString(result["id"]))
	sb.WriteString(":\n")

	if husband, ok := result["husband"].(map[string]interface{}); ok {
		sb.WriteString("- Husband: ")
		display.WritePersonWithDates(&sb, getString(husband["name"]), getString(husband["id"]), getString(husband["birth"]), getString(husband["death"]))
		sb.WriteString("\n")
	}

	if wife, ok := result["wife"].(map[string]interface{}); ok {
		sb.WriteString("- Wife: ")
		display.WritePersonWithDates(&sb, getString(wife["name"]), getString(wife["id"]), getString(wife["birth"]), getString(wife["death"]))
		sb.WriteString("\n")
	}

	if marriage, ok := result["marriage"].(map[string]interface{}); ok {
		sb.WriteString("- Marriage:")
		if date := getString(marriage["date"]); date != "" {
			sb.WriteString(" ")
			sb.WriteString(date)
		}
		if place := getString(marriage["place"]); place != "" {
			sb.WriteString(" (")
			sb.WriteString(place)
			sb.WriteString(")")
		}
		sb.WriteString("\n")
	}

	if children, ok := result["children"].([]map[string]interface{}); ok && len(children) > 0 {
		sb.WriteString("- Children:\n")
		for _, child := range children {
			sb.WriteString("  - ")
			display.WritePersonWithDates(&sb, getString(child["name"]), getString(child["id"]), getString(child["birth"]), getString(child["death"]))
			sb.WriteString("\n")
		}
	}

	sb.WriteString(fmt.Sprintf("- Number of children: %v\n", result["child_count"]))

	if timeline, ok := result["timeline"].([]interface{}); ok && len(timeline) > 0 {
		sb.WriteString("- Geographic timeline:\n")
		for _, segI := range timeline {
			seg, _ := segI.(map[string]interface{})
			if seg == nil {
				continue
			}
			fromDate := getString(seg["from_date"])
			toDate := getString(seg["to_date"])
			city := getString(seg["city"])
			country := getString(seg["country"])
			loc := city
			if country != "" && country != city {
				loc += " / " + country
			}
			sb.WriteString(fmt.Sprintf("  From %s to %s: %s\n", fromDate, toDate, loc))
			if evts, ok := seg["events"].([]interface{}); ok {
				for _, evI := range evts {
					ev, _ := evI.(map[string]interface{})
					if ev == nil {
						continue
					}
					sb.WriteString(fmt.Sprintf("    - %s: %s\n", getString(ev["date"]), getString(ev["label"])))
				}
			}
		}
	}

	if divorce, ok := result["divorce"].(map[string]interface{}); ok {
		sb.WriteString("- Divorce:")
		if date := getString(divorce["date"]); date != "" {
			sb.WriteString(" ")
			sb.WriteString(date)
		}
		if place := getString(divorce["place"]); place != "" {
			sb.WriteString(" (")
			sb.WriteString(place)
			sb.WriteString(")")
		}
		sb.WriteString("\n")
	}

	if notes, ok := result["notes"].([]string); ok && len(notes) > 0 {
		sb.WriteString("- Notes:\n")
		for _, note := range notes {
			sb.WriteString("  - ")
			sb.WriteString(note)
			sb.WriteString("\n")
		}
	}

	return mcp.NewCallToolResultWithStructured(sb.String(), result)
}

func formatSearchByDateRangeResult(result []map[string]interface{}) *mcp.CallToolResult {
	return display.SearchResults("Individuals found", result)
}

func getString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func formatGetChildrenResult(result map[string]interface{}) *mcp.CallToolResult {
	if err := display.CheckError(result); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("Children of ")
	sb.WriteString(getString(result["id"]))
	sb.WriteString(" (")
	sb.WriteString(getString(result["name"]))
	sb.WriteString("):\n")

	structuredFamilies := []map[string]interface{}{}

	if families, ok := result["families"].([]map[string]interface{}); ok && len(families) > 0 {
		for _, fam := range families {
			familyID := getString(fam["id"])
			sb.WriteString("- Family ")
			sb.WriteString(familyID)
			sb.WriteString(":\n")

			if spouse, ok := fam["spouse"].(map[string]interface{}); ok && spouse != nil {
				sb.WriteString("  - Spouse: ")
				display.WritePersonWithDates(&sb, getString(spouse["name"]), getString(spouse["id"]), getString(spouse["birth"]), getString(spouse["death"]))
				sb.WriteString("\n")
			}

			if children, ok := fam["children"].([]map[string]interface{}); ok && len(children) > 0 {
				sb.WriteString("  - Children:\n")
				for _, child := range children {
					sb.WriteString("    - ")
					display.WritePersonWithDates(&sb, getString(child["name"]), getString(child["id"]), getString(child["birth"]), getString(child["death"]))
					if pedigree, ok := child["pedigree"].(string); ok && pedigree != "" {
						sb.WriteString(" [")
						sb.WriteString(pedigree)
						sb.WriteString("]")
					}
					if sfs, ok := child["spouse_families"].([]map[string]interface{}); ok {
						for _, sf := range sfs {
							sb.WriteString(" -> Spouse: ")
							sb.WriteString(getString(sf["spouse_name"]))
							sb.WriteString(" (")
							sb.WriteString(getString(sf["family_id"]))
							sb.WriteString(")")
						}
					}
					sb.WriteString("\n")
				}
			}

			structuredFamilies = append(structuredFamilies, map[string]interface{}{
				"id":       familyID,
				"spouse":   fam["spouse"],
				"children": fam["children"],
			})
		}
	} else {
		sb.WriteString("- No children found\n")
	}

	var pagination map[string]interface{}
	if p, ok := result["pagination"].(map[string]interface{}); ok {
		pagination = map[string]interface{}{
			"offset": p["offset"],
			"limit":  p["limit"],
			"total":  p["total"],
		}
	}

	structuredResult := map[string]interface{}{
		"id":         result["id"],
		"name":       result["name"],
		"families":   structuredFamilies,
		"pagination": pagination,
	}

	return mcp.NewCallToolResultWithStructured(sb.String(), structuredResult)
}

func formatGetParentsResult(result map[string]interface{}) *mcp.CallToolResult {
	if err := display.CheckError(result); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("Parents of ")
	sb.WriteString(getString(result["id"]))
	sb.WriteString(" (")
	sb.WriteString(getString(result["name"]))
	sb.WriteString("):\n")

	structuredFamilies := []map[string]interface{}{}

	if families, ok := result["families"].([]map[string]interface{}); ok && len(families) > 0 {
		for _, fam := range families {
			familyID := getString(fam["family_id"])
			familyType := getString(fam["type"])

			sb.WriteString("- Family ")
			sb.WriteString(familyID)
			if familyType != "" {
				sb.WriteString(" (")
				sb.WriteString(familyType)
				sb.WriteString(")")
			}
			sb.WriteString(":\n")

			hasParent := false
			for _, role := range []string{"father", "mother", "parent"} {
				if p, ok := fam[role].(map[string]interface{}); ok && p != nil {
					sb.WriteString("  ")
					sb.WriteString(strings.ToUpper(role[:1]) + role[1:])
					sb.WriteString(": ")
					display.WritePersonWithDates(&sb, getString(p["name"]), getString(p["id"]), getString(p["birth"]), getString(p["death"]))
					sb.WriteString("\n")
					hasParent = true
				}
			}
			if !hasParent {
				sb.WriteString("  (No parents recorded)\n")
			}

			familyData := map[string]interface{}{
				"family_id": familyID,
				"father":    fam["father"],
				"mother":    fam["mother"],
			}
			if parent, ok := fam["parent"]; ok && parent != nil {
				familyData["parent"] = parent
			}
			if familyType != "" {
				familyData["type"] = familyType
			}
			structuredFamilies = append(structuredFamilies, familyData)
		}
	} else {
		sb.WriteString("- No parents found\n")
	}

	structuredResult := map[string]interface{}{
		"id":       result["id"],
		"name":     result["name"],
		"families": structuredFamilies,
	}

	return mcp.NewCallToolResultWithStructured(sb.String(), structuredResult)
}

func formatGetAncestorsResult(result map[string]interface{}) *mcp.CallToolResult {
	return display.GenResults("Ancestors of",
		getString(result["id"]), getString(result["name"]),
		result, "ancestors", "ancestors")
}

func formatGetDescendantsResult(result map[string]interface{}) *mcp.CallToolResult {
	return display.GenResults("Descendants of",
		getString(result["id"]), getString(result["name"]),
		result, "descendants", "descendants")
}

func formatFindRelationshipPathResult(result map[string]interface{}) *mcp.CallToolResult {
	if err := display.CheckError(result); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("Relationship between ")
	name1 := gedcom.FormatNameWithID(getString(result["name1"]), getString(result["id1"]))
	name2 := gedcom.FormatNameWithID(getString(result["name2"]), getString(result["id2"]))
	sb.WriteString(name1)
	sb.WriteString(" and ")
	sb.WriteString(name2)
	sb.WriteString(":\n")
	sb.WriteString("- Relationship: ")
	sb.WriteString(getString(result["relationship"]))
	sb.WriteString("\n- Path: ")

	if path, ok := result["path"].([]map[string]interface{}); ok && len(path) > 0 {
		isMeetingFound := false
		for _, node := range path {
			//Avant le isMeeting afficher le lien avant le nom
			if !isMeetingFound {
				if link, ok := node["haveA"].(string); ok && link != "" {
					sb.WriteString(fmt.Sprintf(" [%s of →] ", link))
				}
			}

			// Add (*) for meeting node
			if isMeeting, ok := node["isMeeting"].(bool); ok && isMeeting {
				sb.WriteString("(Junction) ")
				isMeetingFound = true
				continue
			}
			sb.WriteString(gedcom.FormatNameWithID(getString(node["name"]), getString(node["id"])))

			//Après le isMeeting afficher le lien après le nom
			if isMeetingFound {
				if link, ok := node["haveA"].(string); ok && link != "" {
					sb.WriteString(fmt.Sprintf(" [%s of →] ", link))
				}
			}
		}
	} else {
		sb.WriteString("none")
	}

	return mcp.NewCallToolResultWithStructured(sb.String(), result)
}

func formatGetStatisticsResult(result map[string]interface{}) *mcp.CallToolResult {
	var sb strings.Builder
	sb.WriteString("GEDCOM Statistics:\n")
	sb.WriteString("- Total individuals: ")
	sb.WriteString(fmt.Sprintf("%d", result["total_individuals"]))
	sb.WriteString("\n- Total families: ")
	sb.WriteString(fmt.Sprintf("%d", result["total_families"]))
	sb.WriteString("\n")

	return mcp.NewCallToolResult(sb.String())
}

func formatFindAllRelationshipsResult(links []gedcom.RelationshipLink, coefficient float64, useGiven bool) *mcp.CallToolResult {
	if len(links) == 0 {
		return mcp.NewCallToolResult("Aucun lien de parenté trouvé entre ces deux individus.")
	}

	cleanName := func(name string) string {
		return strings.ReplaceAll(strings.ReplaceAll(name, "\"", ""), "/", "")
	}

		type nameInfo struct {
			display string
			first   string
			surName string
		}

		getDisplayName := func(raw string) string {
			if !useGiven {
				return gedcom.ResolveName(raw, false)
			}
			return cleanName(raw)
		}

		getNameInfo := func(indID string) nameInfo {
			ind := gedcom.Get().Individual(indID)
			if ind == nil || len(ind.Name) == 0 {
				return nameInfo{display: indID, first: indID, surName: indID}
			}
			full := getDisplayName(ind.Name[0].Name)
			fullClean := cleanName(ind.Name[0].Name)
			parts := strings.Fields(fullClean)
			if len(parts) == 0 {
				return nameInfo{display: indID, first: indID, surName: indID}
			}
			surName := strings.ToUpper(parts[len(parts)-1])
			var givenNames string
			if !useGiven {
				partsDisplay := strings.Fields(full)
				if len(partsDisplay) > 0 {
					if len(partsDisplay) > 1 {
						givenNames = strings.Join(partsDisplay[:len(partsDisplay)-1], " ")
					} else {
						givenNames = partsDisplay[0]
					}
				} else {
					givenNames = strings.Join(parts[:len(parts)-1], " ")
					if givenNames == "" {
						givenNames = parts[0]
					}
				}
			} else {
				givenNames = strings.Join(parts[:len(parts)-1], " ")
				if givenNames == "" {
					givenNames = parts[0]
				}
			}
			return nameInfo{
				display: full + " (" + indID + ")",
				first:   givenNames,
				surName: surName + " (" + indID + ")",
			}
		}

		getAncestorDisplay := func(id string) string {
			ind := gedcom.Get().Individual(id)
			if ind != nil && len(ind.Name) > 0 {
				if !useGiven {
					return gedcom.ResolveName(ind.Name[0].Name, false) + " (" + id + ")"
				}
				return gedcom.FormatNameWithID(ind.Name[0].Name, id)
			}
			return id
		}

	getAncestorSex := func(id string) string {
		ind := gedcom.Get().Individual(id)
		if ind == nil || ind.Sex == "" {
			return "M"
		}
		return ind.Sex
	}

	// Get person names from first link
	nameInfoA := nameInfo{}
	nameInfoB := nameInfo{}
	if len(links) > 0 {
		if len(links[0].PathFromA) > 0 {
			nameInfoA = getNameInfo(links[0].PathFromA[0])
		}
		if len(links[0].PathFromB) > 0 {
			nameInfoB = getNameInfo(links[0].PathFromB[0])
		}
	}

	blocks := display.GroupIntoBlocks(links)

	isAncestorResult := len(blocks) > 0 && (blocks[0].DepthA == 0 || blocks[0].DepthB == 0)

	// Total individual ancestors
	totalAncestors := 0
	for _, block := range blocks {
		for _, count := range block.AncestorCounts {
			totalAncestors += count
		}
	}

	var sb strings.Builder
	var totalLabel string
	if isAncestorResult {
		if totalAncestors > 1 {
			totalLabel = fmt.Sprintf("Parenté (%d branches)\n\n", totalAncestors)
		} else {
			totalLabel = fmt.Sprintf("Parenté (%d branche)\n\n", totalAncestors)
		}
	} else if totalAncestors > 1 {
		totalLabel = fmt.Sprintf("Parenté (%d liens de parenté)\n\n", totalAncestors)
	} else {
		totalLabel = fmt.Sprintf("Parenté (%d lien de parenté)\n\n", totalAncestors)
	}
	sb.WriteString(totalLabel)

	for bi, block := range blocks {
		isAncestorBlock := block.DepthA == 0 || block.DepthB == 0

		firstDisplay := nameInfoA.display
		lastName := nameInfoB
		if block.DepthB == 0 {
			firstDisplay = nameInfoB.display
			lastName = nameInfoA
		}

		prefix := "est un"
		if bi > 0 {
			prefix = "est aussi un"
		}

		if isAncestorBlock {
			var ancestorID string
			if block.DepthA == 0 {
				ancestorID = links[0].PathFromA[0]
			} else {
				ancestorID = links[0].PathFromB[0]
			}
			if getAncestorSex(ancestorID) == "F" {
				prefix = strings.Replace(prefix, "un", "une", 1)
			}
		}

		relLabel := block.RelationLabel

		sb.WriteString(fmt.Sprintf(" %s %s %s %s %s.\n\n",
			firstDisplay, prefix, relLabel,
			display.DeName(lastName.first), lastName.surName))

		if !isAncestorResult {
			sb.WriteString("En effet,\n")

			blockSex := "M"
			for _, ancID := range block.AncestorIDs {
				ancDisplay := getAncestorDisplay(ancID)
				count := block.AncestorCounts[ancID]
				var linkLabel string
				if count > 1 {
					linkLabel = fmt.Sprintf("(%d liens de parenté)   Voir", count)
				} else {
					linkLabel = "(1 lien de parenté)   Voir"
				}
				as := getAncestorSex(ancID)
				if as == "F" {
					blockSex = "F"
				}
				sb.WriteString(fmt.Sprintf("%s %s\n", ancDisplay, linkLabel))
			}

			isPlural := len(block.AncestorIDs) > 1
			if isPlural {
				sb.WriteString("sont en même temps\n")
			} else {
				sb.WriteString("est en même temps\n")
			}

			isCouple := len(block.AncestorIDs) > 1
			labelPrefix := "des "
			if !isCouple {
				sex := getAncestorSex(block.AncestorIDs[0])
				if sex == "F" {
					labelPrefix = "une "
				} else {
					labelPrefix = "un "
				}
			}

			labelB := gedcom.AncestorLabel(block.DepthB, isCouple, blockSex)
			labelA := gedcom.AncestorLabel(block.DepthA, isCouple, blockSex)

			sb.WriteString(fmt.Sprintf("%s%s %s %s\n", labelPrefix, labelB, display.DeName(nameInfoB.first), nameInfoB.surName))
			sb.WriteString(fmt.Sprintf("%s%s %s %s\n", labelPrefix, labelA, display.DeName(nameInfoA.first), nameInfoA.surName))
		}
	}

	// Coefficient with comma as decimal separator
	coeffStr := strings.Replace(fmt.Sprintf("%.1f", coefficient*100), ".", ",", 1)
	sb.WriteString(fmt.Sprintf("Parenté: %s%%", coeffStr))

	result := map[string]interface{}{
		"blocks":      blocks,
		"coefficient": coefficient,
		"total_links": totalAncestors,
	}

	return mcp.NewCallToolResultWithStructured(sb.String(), result)
}

func formatLoadGedcomFileResult(result map[string]interface{}) *mcp.CallToolResult {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("GEDCOM file loaded: %s\n", result["path"]))
	sb.WriteString(fmt.Sprintf("- Total individuals: %v\n", result["total_individuals"]))
	sb.WriteString(fmt.Sprintf("- Total families: %v\n", result["total_families"]))
	return mcp.NewCallToolResultWithStructured(sb.String(), result)
}


