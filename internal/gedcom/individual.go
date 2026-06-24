package gedcom

import (
	"strings"

	"github.com/iand/gedcom"
)

func CleanName(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, "\"", ""), "/", "")
}

func ExtractGivenName(rawName string) string {
	slashIdx := strings.Index(rawName, "/")
	if slashIdx == -1 {
		return CleanName(rawName)
	}
	given := strings.TrimSpace(rawName[:slashIdx])
	return strings.ReplaceAll(given, "\"", "")
}

func ExtractUsageName(rawName string) string {
	start := strings.Index(rawName, "\"")
	if start == -1 {
		return ""
	}
	end := strings.Index(rawName[start+1:], "\"")
	if end == -1 {
		return ""
	}
	return rawName[start+1 : start+1+end]
}

func ExtractSurname(rawName string) string {
	start := strings.Index(rawName, "/")
	if start == -1 {
		return ""
	}
	end := strings.Index(rawName[start+1:], "/")
	if end == -1 {
		return ""
	}
	return rawName[start+1 : start+1+end]
}

func ResolveName(rawName string, useGiven bool) string {
	usage := ExtractUsageName(rawName)
	given := ExtractGivenName(rawName)
	surname := ExtractSurname(rawName)
	if useGiven {
		if given == "" {
			return CleanName(rawName)
		}
		if surname == "" {
			return given
		}
		return given + " " + surname
	}
	if usage == "" {
		usage = given
	}
	if surname == "" {
		return usage
	}
	return usage + " " + surname
}

func MakePersonSummary(ind *gedcom.IndividualRecord) map[string]interface{} {
	rawName := ""
	if len(ind.Name) > 0 {
		rawName = ind.Name[0].Name
	}
	usageName := ExtractUsageName(rawName)
	givenName := ExtractGivenName(rawName)
	surname := ExtractSurname(rawName)
	if usageName == "" {
		usageName = givenName
	}
	result := map[string]interface{}{
		"id":         ind.Xref,
		"name":       CleanName(rawName),
		"given_name": givenName,
		"usage_name": usageName,
		"surname":    surname,
		"sex":        ind.Sex,
	}
	if ind.Event != nil {
		for _, e := range ind.Event {
			if e.Date != "" {
				switch e.Tag {
				case "BIRT":
					result["birth"] = e.Date
				case "DEAT":
					result["death"] = e.Date
				}
			}
		}
	}
	return result
}

func PersonMap(ind *gedcom.IndividualRecord) map[string]interface{} {
	rawName := ""
	if len(ind.Name) > 0 {
		rawName = ind.Name[0].Name
	}
	usageName := ExtractUsageName(rawName)
	givenName := ExtractGivenName(rawName)
	surname := ExtractSurname(rawName)
	if usageName == "" {
		usageName = givenName
	}
	return map[string]interface{}{
		"id":         ind.Xref,
		"name":       CleanName(rawName),
		"given_name": givenName,
		"usage_name": usageName,
		"surname":    surname,
		"sex":        ind.Sex,
	}
}

func PersonMapWithBirth(ind *gedcom.IndividualRecord) map[string]interface{} {
	rawName := ""
	if len(ind.Name) > 0 {
		rawName = ind.Name[0].Name
	}
	usageName := ExtractUsageName(rawName)
	givenName := ExtractGivenName(rawName)
	surname := ExtractSurname(rawName)
	if usageName == "" {
		usageName = givenName
	}
	m := map[string]interface{}{
		"id":         ind.Xref,
		"name":       CleanName(rawName),
		"given_name": givenName,
		"usage_name": usageName,
		"surname":    surname,
		"sex":        ind.Sex,
	}
	if birth := getBirthDate(ind); birth != "" {
		m["birth"] = birth
	}
	if death := getDeathDate(ind); death != "" {
		m["death"] = death
	}
	return m
}

func GetBirthDate(ind *gedcom.IndividualRecord) string {
	return getBirthDate(ind)
}

func getBirthDate(ind *gedcom.IndividualRecord) string {
	if ind.Event != nil {
		for _, e := range ind.Event {
			if e.Tag == "BIRT" && e.Date != "" {
				return e.Date
			}
		}
	}
	return ""
}

func GetDeathDate(ind *gedcom.IndividualRecord) string {
	return getDeathDate(ind)
}

func getDeathDate(ind *gedcom.IndividualRecord) string {
	if ind.Event != nil {
		for _, e := range ind.Event {
			if e.Tag == "DEAT" && e.Date != "" {
				return e.Date
			}
		}
	}
	return ""
}

func SpouseInFamily(ind *gedcom.IndividualRecord, family *gedcom.FamilyRecord) *gedcom.IndividualRecord {
	if ind.Sex == "M" && family.Wife != nil {
		return family.Wife
	}
	if ind.Sex == "F" && family.Husband != nil {
		return family.Husband
	}
	if family.Husband != nil && family.Husband.Xref != ind.Xref {
		return family.Husband
	}
	if family.Wife != nil && family.Wife.Xref != ind.Xref {
		return family.Wife
	}
	return nil
}

func ChildMap(child *gedcom.IndividualRecord, familyID string) map[string]interface{} {
	m := PersonMapWithBirth(child)
	for _, childFl := range child.Parents {
		if childFl.Family != nil && childFl.Family.Xref == familyID {
			if childFl.Type != "" && childFl.Type != "birth" {
				m["pedigree"] = childFl.Type
			}
			break
		}
	}
	return m
}

func MakePagination(offset, limit, total int) map[string]interface{} {
	return map[string]interface{}{
		"offset": offset,
		"limit":  limit,
		"total":  total,
	}
}
