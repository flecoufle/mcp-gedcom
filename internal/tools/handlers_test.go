package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flecoufle/mcp-gedcom/internal/gedcom"
)

func setupTestGedcom(t *testing.T, content string) {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.ged")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test GEDCOM: %v", err)
	}
	if err := gedcom.Init(tmpFile); err != nil {
		t.Fatalf("failed to init gedcom: %v", err)
	}
}

func TestExtractYear(t *testing.T) {
	tests := []struct {
		dateStr string
		want    int
	}{
		{"1970", 1970},
		{"1970-05-15", 1970},
		{"ABT 1970", 1970},
		{"BET 1968 AND 1972", 1968}, // first year found
		{"15 MAY 1970", 1970},
		{"", 0},
		{"NO YEAR", 0},
		{"BIRT 1975", 1975},
	}
	for _, tt := range tests {
		got := extractYear(tt.dateStr)
		if got != tt.want {
			t.Errorf("extractYear(%q) = %d, want %d", tt.dateStr, got, tt.want)
		}
	}
}

func TestHandleSearchPerson_ByNameOnly(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
2 DATE 1970
0 @I2@ INDI
1 NAME Jane /Doe/
1 BIRT
2 DATE 1975
0 @I3@ INDI
1 NAME Bob /Smith/
1 BIRT
2 DATE 1980
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	// Search by pattern "Doe" should return John and Jane
	results, err := HandleSearchPerson("Doe", 0, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	// Check names
	found := map[string]bool{}
	for _, r := range results {
		if name, ok := r["name"].(string); ok {
			found[name] = true
		}
	}
	if !found["John Doe"] || !found["Jane Doe"] {
		t.Errorf("expected John and Jane, got %v", found)
	}
}

func TestHandleSearchPerson_WithBirthYear(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
2 DATE 1970
0 @I2@ INDI
1 NAME Jane /Doe/
1 BIRT
2 DATE 1975
0 @I3@ INDI
1 NAME Bob /Smith/
1 BIRT
2 DATE 1980
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	// Search with birthYear 1970 (±2) should return John (1970) and maybe Jane (1975) is outside range?
	// Actually 1975 is within 1970±2? 1975-1970=5 >2, so not included.
	// John's birth 1970 is within range. Jane's 1975 is not.
	results, err := HandleSearchPerson("Doe", 1970, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if len(results) > 0 {
		if name, ok := results[0]["name"].(string); !ok || !strings.Contains(name, "John") {
			t.Errorf("expected John, got %v", results[0])
		}
	}
}

func TestHandleSearchPerson_NoMatches(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
2 DATE 1970
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	results, err := HandleSearchPerson("Smith", 0, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result (message), got %d", len(results))
	}
	if len(results) > 0 {
		if msg, ok := results[0]["message"].(string); !ok || !strings.Contains(msg, "No individuals found") {
			t.Errorf("expected not found message, got %v", results[0])
		}
	}
}

func TestHandleSearchPerson_MissingPattern(t *testing.T) {
	results, err := HandleSearchPerson("", 0, true)
	if err == nil {
		t.Error("expected error for missing pattern")
	}
	if results != nil {
		t.Errorf("expected nil results, got %v", results)
	}
}

func TestHandleSearchPerson_BirthYearFormats(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
2 DATE ABT 1970
0 @I2@ INDI
1 NAME Jane /Doe/
1 BIRT
2 DATE 1975
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	// "ABT 1970" should be extracted as 1970, within ±2 of 1970
	results, err := HandleSearchPerson("Doe", 1970, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should include John (ABT 1970) but not Jane (1975)
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestHandleSearchSurnames(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
0 @I2@ INDI
1 NAME Jane /Doe/
0 @I3@ INDI
1 NAME Bob /Smith/
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleSearchSurnames("Do", 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	surnames, ok := result["surnames"].([]map[string]interface{})
	if !ok || len(surnames) != 1 {
		t.Fatalf("expected 1 surname, got %v", surnames)
	}
	if surnames[0]["name"] != "Doe" {
		t.Errorf("expected Doe, got %v", surnames[0]["name"])
	}
}

func TestHandleSearchSurnames_NoMatch(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleSearchSurnames("Smith", 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	surnames, ok := result["surnames"].([]map[string]interface{})
	if !ok || len(surnames) != 0 {
		t.Errorf("expected 0 surnames, got %v", surnames)
	}
}

func TestHandleSearchSurnames_Pagination(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
0 @I2@ INDI
1 NAME Jane /Doe/
0 @I3@ INDI
1 NAME Bob /Smith/
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleSearchSurnames("e", 0, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pagination, ok := result["pagination"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected pagination info")
	}
	if pagination["total"] != 1 {
		t.Errorf("expected total 1, got %v", pagination["total"])
	}
}

func TestHandleGetPersonDetails(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
1 BIRT
2 DATE 1970
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetPersonDetails("I1", true, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["id"] != "I1" {
		t.Errorf("expected id I1, got %v", result["id"])
	}
}

func TestHandleGetPersonDetails_NotFound(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetPersonDetails("I999", true, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result["error"]; !ok {
		t.Errorf("expected error for non-existent individual")
	}
}

func TestHandleGetPersonDetails_NoSpouse(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Jane /Doe/
1 SEX F
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetPersonDetails("I1", false, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result["families"]; ok {
		t.Errorf("expected no families when both flags are false")
	}
}

func TestHandleGetPersonDetails_WithSiblings(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
1 FAMC @F1@
0 @I2@ INDI
1 NAME Jane /Doe/
1 SEX F
1 FAMC @F1@
0 @I3@ INDI
1 NAME Baby /Doe/
1 SEX M
1 FAMC @F1@
0 @I4@ INDI
1 NAME Dad /Doe/
1 SEX M
0 @I5@ INDI
1 NAME Mom /Doe/
1 SEX F
0 @F1@ FAM
1 HUSB @I4@
1 WIFE @I5@
1 CHIL @I1@
1 CHIL @I2@
1 CHIL @I3@
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetPersonDetails("I1", true, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["id"] != "I1" {
		t.Errorf("expected id I1, got %v", result["id"])
	}
	sgs, ok := result["siblings_groups"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected siblings_groups to be present")
	}
	if len(sgs) != 1 {
		t.Fatalf("expected 1 siblings group, got %d", len(sgs))
	}
	sg := sgs[0]
	if fmt.Sprintf("%v", sg["type"]) != "full" {
		t.Errorf("expected type full, got %v", sg["type"])
	}
	if fmt.Sprintf("%v", sg["label"]) != "same father and same mother" {
		t.Errorf("expected label 'same father and same mother', got %v", sg["label"])
	}
	if fmt.Sprintf("%v", sg["id"]) != "F1" {
		t.Errorf("expected id F1, got %v", sg["id"])
	}
	parents, ok := sg["parents"].(map[string]interface{})
	if !ok {
		t.Fatal("expected parents to be present")
	}
	if fmt.Sprintf("%v", parents["father_id"]) != "I4" {
		t.Errorf("expected father_id I4, got %v", parents["father_id"])
	}
	if fmt.Sprintf("%v", parents["mother_id"]) != "I5" {
		t.Errorf("expected mother_id I5, got %v", parents["mother_id"])
	}
	list, ok := sg["list"].([]interface{})
	if !ok {
		t.Fatal("expected list to be present")
	}
	if len(list) != 2 {
		t.Errorf("expected 2 siblings in list, got %d", len(list))
	}
}

func TestHandleGetPersonDetails_MaternalHalf(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I10@ INDI
1 NAME Mom /Smith/
1 SEX F
1 FAMS @F10@
1 FAMS @F11@
0 @I11@ INDI
1 NAME Dad1 /Smith/
1 SEX M
0 @I12@ INDI
1 NAME Dad2 /Jones/
1 SEX M
0 @I20@ INDI
1 NAME Target /Smith/
1 SEX F
1 FAMC @F10@
0 @I21@ INDI
1 NAME Half /Jones/
1 SEX M
1 FAMC @F11@
0 @I22@ INDI
1 NAME Full /Smith/
1 SEX M
1 FAMC @F10@
0 @F10@ FAM
1 HUSB @I11@
1 WIFE @I10@
1 CHIL @I20@
1 CHIL @I22@
0 @F11@ FAM
1 HUSB @I12@
1 WIFE @I10@
1 CHIL @I21@
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetPersonDetails("I20", true, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sgs, ok := result["siblings_groups"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected siblings_groups to be present")
	}
	if len(sgs) != 2 {
		t.Fatalf("expected 2 siblings groups, got %d", len(sgs))
	}
	sg0 := sgs[0]
	if fmt.Sprintf("%v", sg0["type"]) != "full" {
		t.Errorf("expected first group type full, got %v", sg0["type"])
	}
	if fmt.Sprintf("%v", sg0["id"]) != "F10" {
		t.Errorf("expected id F10, got %v", sg0["id"])
	}
	list0 := sg0["list"].([]interface{})
	if len(list0) != 1 {
		t.Fatalf("expected 1 sibling in full group, got %d", len(list0))
	}
	entry0 := list0[0].(map[string]interface{})
	if fmt.Sprintf("%v", entry0["id"]) != "I22" {
		t.Errorf("expected sibling id I22, got %v", entry0["id"])
	}
	sg1 := sgs[1]
	if fmt.Sprintf("%v", sg1["type"]) != "maternal_half" {
		t.Errorf("expected second group type maternal_half, got %v", sg1["type"])
	}
	if fmt.Sprintf("%v", sg1["label"]) != "same mother" {
		t.Errorf("expected label 'same mother', got %v", sg1["label"])
	}
	if fmt.Sprintf("%v", sg1["id"]) != "F11" {
		t.Errorf("expected id F11, got %v", sg1["id"])
	}
	parents := sg1["parents"].(map[string]interface{})
	if fmt.Sprintf("%v", parents["mother_id"]) != "I10" {
		t.Errorf("expected mother_id I10, got %v", parents["mother_id"])
	}
	if fmt.Sprintf("%v", parents["father_id"]) != "I12" {
		t.Errorf("expected father_id I12, got %v", parents["father_id"])
	}
	list := sg1["list"].([]interface{})
	if len(list) != 1 {
		t.Fatalf("expected 1 sibling in maternal_half group, got %d", len(list))
	}
	entry := list[0].(map[string]interface{})
	if fmt.Sprintf("%v", entry["id"]) != "I21" {
		t.Errorf("expected sibling id I21, got %v", entry["id"])
	}
}

func TestHandleGetPersonDetails_AncestorFamilies(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
1 FAMC @F1@
0 @I2@ INDI
1 NAME Jane /Doe/
1 SEX F
0 @I3@ INDI
1 NAME Baby /Doe/
1 SEX M
1 FAMC @F1@
0 @F1@ FAM
1 HUSB @I2@
1 WIFE @I3@
1 CHIL @I1@
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetPersonDetails("I1", true, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	afamilies, ok := result["ancestor_families"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected ancestor_families to be present")
	}
	if len(afamilies) != 1 {
		t.Fatalf("expected 1 ancestor family, got %d", len(afamilies))
	}
	af := afamilies[0]
	if fmt.Sprintf("%v", af["id"]) != "F1" {
		t.Errorf("expected family id F1, got %v", af["id"])
	}
	if fmt.Sprintf("%v", af["children_count"]) != "1" {
		t.Errorf("expected children_count 1, got %v", af["children_count"])
	}
	father, ok := af["father"].(map[string]interface{})
	if !ok {
		t.Fatal("expected father to be present")
	}
	if fmt.Sprintf("%v", father["id"]) != "I2" {
		t.Errorf("expected father id I2, got %v", father["id"])
	}
	mother, ok := af["mother"].(map[string]interface{})
	if !ok {
		t.Fatal("expected mother to be present")
	}
	if fmt.Sprintf("%v", mother["id"]) != "I3" {
		t.Errorf("expected mother id I3, got %v", mother["id"])
	}
}

func TestHandleGetRelatives(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Jane /Doe/
1 SEX F
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetRelatives("I1", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	relatives, ok := result["relatives"].(map[string][]string)
	if !ok {
		t.Fatalf("expected relatives map")
	}
	if len(relatives["spouse"]) == 0 {
		t.Errorf("expected spouse relations")
	}
}

func TestHandleGetRelatives_NotFound(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetRelatives("I999", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result["error"]; !ok {
		t.Errorf("expected error for non-existent individual")
	}
}

func TestHandleGetFamilyDetails(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
1 BIRT
2 DATE 15 MAR 1980
2 PLAC Springfield, Illinois, USA
0 @I2@ INDI
1 NAME Jane /Doe/
1 SEX F
1 BIRT
2 DATE 22 JUN 1982
2 PLAC Shelbyville, Illinois, USA
0 @I3@ INDI
1 NAME Baby /Doe/
1 SEX M
1 BIRT
2 DATE 10 JAN 2010
2 PLAC Springfield, Illinois, USA
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
1 MARR
2 DATE 14 FEB 2005
2 PLAC Springfield, Illinois, USA
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetFamilyDetails("F1", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["id"] != "F1" {
		t.Errorf("expected id F1, got %v", result["id"])
	}
	if result["child_count"] != 1 {
		t.Errorf("expected child_count 1, got %v", result["child_count"])
	}
	timeline, ok := result["timeline"].([]interface{})
	if !ok {
		t.Fatal("expected timeline to be a slice")
	}
	if len(timeline) != 3 {
		t.Fatalf("expected 3 timeline segments, got %d", len(timeline))
	}
	seg0 := timeline[0].(map[string]interface{})
	if fmt.Sprintf("%v", seg0["city"]) != "Springfield" {
		t.Errorf("expected first segment city Springfield, got %v", seg0["city"])
	}
	if len(seg0["events"].([]interface{})) != 1 {
		t.Errorf("expected 1 event in first segment, got %d", len(seg0["events"].([]interface{})))
	}
	seg1 := timeline[1].(map[string]interface{})
	if fmt.Sprintf("%v", seg1["city"]) != "Shelbyville" {
		t.Errorf("expected second segment city Shelbyville, got %v", seg1["city"])
	}
	seg2 := timeline[2].(map[string]interface{})
	if fmt.Sprintf("%v", seg2["city"]) != "Springfield" {
		t.Errorf("expected third segment city Springfield, got %v", seg2["city"])
	}
	evts := seg2["events"].([]interface{})
	if len(evts) != 2 {
		t.Errorf("expected 2 events in Springfield segment, got %d", len(evts))
	}
}

func TestHandleGetFamilyDetails_NotFound(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetFamilyDetails("F999", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result["error"]; !ok {
		t.Errorf("expected error for non-existent family")
	}
}

func TestHandleGetChildren(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Jane /Doe/
1 SEX F
1 FAMS @F1@
0 @I3@ INDI
1 NAME Bob /Doe/
1 SEX M
1 FAMC @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetChildren("I1", 0, 10, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	families, ok := result["families"].([]map[string]interface{})
	if !ok || len(families) == 0 {
		t.Fatalf("expected families with children")
	}
}

func TestHandleGetChildren_NoChildren(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetChildren("I1", 0, 10, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	families, ok := result["families"].([]map[string]interface{})
	if !ok || len(families) != 0 {
		t.Errorf("expected no families for individual with no children")
	}
}

func TestHandleGetParents(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
0 @I2@ INDI
1 NAME Jane /Doe/
1 SEX F
0 @I3@ INDI
1 NAME Bob /Doe/
1 SEX M
1 FAMC @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetParents("I3", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	families, ok := result["families"].([]map[string]interface{})
	if !ok || len(families) == 0 {
		t.Fatalf("expected families with parents")
	}
}

func TestHandleGetParents_NotFound(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetParents("I999", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result["error"]; !ok {
		t.Errorf("expected error for non-existent individual")
	}
}

func TestHandleSearchByDateRange_Birth(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
2 DATE 1970
0 @I2@ INDI
1 NAME Jane /Doe/
1 BIRT
2 DATE 1980
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	results, err := HandleSearchByDateRange(1965, 1975, "birth", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestHandleSearchByDateRange_Death(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 DEAT
2 DATE 2000
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	results, err := HandleSearchByDateRange(1995, 2005, "death", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestHandleSearchByDateRange_NoResults(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 BIRT
2 DATE 1970
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	results, err := HandleSearchByDateRange(2000, 2010, "birth", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 || len(results) > 1 {
		t.Errorf("expected message result, got %d", len(results))
	}
}

func TestHandleGetAncestors(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
0 @I2@ INDI
1 NAME Jane /Doe/
1 SEX F
0 @I3@ INDI
1 NAME Bob /Doe/
1 SEX M
1 FAMC @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetAncestors("I3", 2, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ancestors, ok := result["ancestors"].(map[string][]map[string]interface{})
	if !ok {
		t.Fatalf("expected ancestors map")
	}
	if len(ancestors["1"]) == 0 {
		t.Errorf("expected generation 1 ancestors")
	}
}

func TestHandleGetAncestors_NotFound(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetAncestors("I999", 3, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result["error"]; !ok {
		t.Errorf("expected error for non-existent individual")
	}
}

func TestHandleGetDescendants(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Jane /Doe/
1 SEX F
1 FAMS @F1@
0 @I3@ INDI
1 NAME Bob /Doe/
1 SEX M
1 FAMC @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetDescendants("I1", 2, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	descendants, ok := result["descendants"].(map[string][]map[string]interface{})
	if !ok {
		t.Fatalf("expected descendants map")
	}
	if len(descendants["1"]) == 0 {
		t.Errorf("expected generation 1 descendants")
	}
}

func TestHandleGetDescendants_NotFound(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleGetDescendants("I999", 3, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result["error"]; !ok {
		t.Errorf("expected error for non-existent individual")
	}
}

func TestHandleGetStatistics(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
0 @I2@ INDI
1 NAME Jane /Doe/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result := HandleGetStatistics()
	if result["total_individuals"] != 2 {
		t.Errorf("expected 2 individuals, got %v", result["total_individuals"])
	}
	if result["total_families"] != 1 {
		t.Errorf("expected 1 family, got %v", result["total_families"])
	}
}

func TestHandleFindRelationshipPath(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Jane /Doe/
1 SEX F
1 FAMS @F1@
0 @I3@ INDI
1 NAME Bob /Doe/
1 SEX M
1 FAMC @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	// Test with AncestorsOnly=true (default) - should use parent path
	result, err := HandleFindRelationshipPath("I3", "I1", true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["relationship"] == "no relationship found" {
		t.Errorf("expected to find relationship between parent and child")
	}
}

func TestHandleFindRelationshipPath_AncestorsOnlyFalse(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Jane /Doe/
1 SEX F
1 FAMS @F1@
0 @I3@ INDI
1 NAME Bob /Doe/
1 SEX M
1 FAMC @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	// Test with AncestorsOnly=false - should also find path
	result, err := HandleFindRelationshipPath("I3", "I1", false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["relationship"] == "no relationship found" {
		t.Errorf("expected to find relationship")
	}
}

func TestHandleFindRelationshipPath_NotFound(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleFindRelationshipPath("I1", "I999", true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result["error"]; !ok {
		t.Errorf("expected error for non-existent individual")
	}
}

func TestHandleFindRelationshipPath_SamePerson(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleFindRelationshipPath("I1", "I1", true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["relationship"] != "same person" {
		t.Errorf("expected 'same person', got %v", result["relationship"])
	}
}

func TestHandleLoadGedcomFile(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Doe/
0 @I2@ INDI
1 NAME Jane /Doe/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	result, err := HandleLoadGedcomFile("")
	if err == nil {
		t.Error("expected error for empty path")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestHandleFindAllRelationships(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Grandpa /Doe/
1 SEX M
0 @I2@ INDI
1 NAME Grandma /Doe/
1 SEX F
0 @I3@ INDI
1 NAME ParentA /Doe/
1 SEX M
1 FAMC @F1@
0 @I4@ INDI
1 NAME ParentB /Doe/
1 SEX F
1 FAMC @F1@
0 @I5@ INDI
1 NAME CousinA /Doe/
1 SEX M
1 FAMC @F2@
0 @I6@ INDI
1 NAME CousinB /Doe/
1 SEX M
1 FAMC @F3@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
1 CHIL @I4@
0 @F2@ FAM
1 HUSB @I3@
1 CHIL @I5@
0 @F3@ FAM
1 HUSB @I4@
1 CHIL @I6@
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	links, coeff, err := HandleFindAllRelationships("I5", "I6", 10, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(links) == 0 {
		t.Fatal("expected at least one relationship link")
	}
	if coeff <= 0 {
		t.Errorf("expected positive kinship coefficient, got %f", coeff)
	}
}

func TestHandleFindAllRelationships_NotFound(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME PersonA /Doe/
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	_, _, err := HandleFindAllRelationships("I1", "I999", 5, 10)
	if err == nil {
		t.Error("expected error for non-existent individual")
	}
}

func TestHandleFindAllRelationships_NoRelation(t *testing.T) {
	gedcomContent := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME PersonA /Doe/
0 @I2@ INDI
1 NAME PersonB /Smith/
0 TRLR
`
	setupTestGedcom(t, gedcomContent)

	links, coeff, err := HandleFindAllRelationships("I1", "I2", 5, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("expected no links for unrelated individuals, got %d", len(links))
	}
	if coeff != 0 {
		t.Errorf("expected 0 coefficient, got %f", coeff)
	}
}
