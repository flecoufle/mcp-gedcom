package gedcom

import (
	"os"
	"path/filepath"
	"testing"
)

func setupAncestorGedcom(t *testing.T, content string) {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.ged")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test GEDCOM: %v", err)
	}
	if err := Init(tmpFile); err != nil {
		t.Fatalf("failed to init gedcom: %v", err)
	}
}

func TestGetAllAncestors_SimpleLineage(t *testing.T) {
	content := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Child /Doe/
1 SEX M
1 FAMC @F1@
0 @I2@ INDI
1 NAME Father /Doe/
1 SEX M
1 FAMC @F2@
0 @I3@ INDI
1 NAME Mother /Doe/
1 SEX F
0 @I4@ INDI
1 NAME Grandfather /Doe/
1 SEX M
0 @F1@ FAM
1 HUSB @I2@
1 WIFE @I3@
1 CHIL @I1@
0 @F2@ FAM
1 HUSB @I4@
1 CHIL @I2@
0 TRLR
`
	setupAncestorGedcom(t, content)

	ancestors := GetAllAncestors(Get(), "I1", 3)

	if _, ok := ancestors["I1"]; !ok {
		t.Error("expected I1 in ancestors (self)")
	}
	if _, ok := ancestors["I2"]; !ok {
		t.Error("expected I2 (father) in ancestors")
	}
	if _, ok := ancestors["I3"]; !ok {
		t.Error("expected I3 (mother) in ancestors")
	}
	if _, ok := ancestors["I4"]; !ok {
		t.Error("expected I4 (grandfather) in ancestors")
	}

	if len(ancestors["I1"]) != 1 || ancestors["I1"][0].Depth != 0 {
		t.Errorf("expected I1 depth 0, got %+v", ancestors["I1"])
	}
	if len(ancestors["I2"]) != 1 || ancestors["I2"][0].Depth != 1 {
		t.Errorf("expected I2 depth 1, got %+v", ancestors["I2"])
	}
	if len(ancestors["I4"]) != 1 || ancestors["I4"][0].Depth != 2 {
		t.Errorf("expected I4 depth 2, got %+v", ancestors["I4"])
	}
}

func TestGetAllAncestors_MultiplePaths(t *testing.T) {
	content := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Child /Doe/
1 SEX M
1 FAMC @F1@
0 @I2@ INDI
1 NAME Father /Doe/
1 SEX M
1 FAMC @F2@
0 @I3@ INDI
1 NAME Mother /Doe/
1 SEX F
1 FAMC @F3@
0 @I4@ INDI
1 NAME PaternalGrandpa /Doe/
1 SEX M
0 @I5@ INDI
1 NAME PaternalGrandma /Doe/
1 SEX F
0 @I6@ INDI
1 NAME MaternalGrandpa /Doe/
1 SEX M
0 @I7@ INDI
1 NAME MaternalGrandma /Doe/
1 SEX F
0 @F1@ FAM
1 HUSB @I2@
1 WIFE @I3@
1 CHIL @I1@
0 @F2@ FAM
1 HUSB @I4@
1 WIFE @I5@
1 CHIL @I2@
0 @F3@ FAM
1 HUSB @I6@
1 WIFE @I7@
1 CHIL @I3@
0 TRLR
`
	setupAncestorGedcom(t, content)

	ancestors := GetAllAncestors(Get(), "I1", 3)

	expected := []string{"I1", "I2", "I3", "I4", "I5", "I6", "I7"}
	for _, id := range expected {
		if _, ok := ancestors[id]; !ok {
			t.Errorf("expected %s in ancestors, not found", id)
		}
	}
	if len(ancestors) != len(expected) {
		t.Errorf("expected %d entries, got %d", len(expected), len(ancestors))
	}
}

func TestGetAllAncestors_MaxDepth(t *testing.T) {
	content := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Child /Doe/
1 SEX M
1 FAMC @F1@
0 @I2@ INDI
1 NAME Father /Doe/
1 SEX M
1 FAMC @F2@
0 @I3@ INDI
1 NAME Grandfather /Doe/
1 SEX M
0 @F1@ FAM
1 HUSB @I2@
1 CHIL @I1@
0 @F2@ FAM
1 HUSB @I3@
1 CHIL @I2@
0 TRLR
`
	setupAncestorGedcom(t, content)

	ancestors1 := GetAllAncestors(Get(), "I1", 0)
	if _, ok := ancestors1["I2"]; ok {
		t.Error("expected no I2 at depth 0")
	}

	ancestors2 := GetAllAncestors(Get(), "I1", 1)
	if _, ok := ancestors2["I2"]; !ok {
		t.Error("expected I2 (parent) at depth 1")
	}

	ancestors3 := GetAllAncestors(Get(), "I1", 2)
	if _, ok := ancestors3["I3"]; !ok {
		t.Error("expected I3 (grandparent) at depth 2")
	}
}

func TestGetAllAncestors_NoParents(t *testing.T) {
	content := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Alone /Doe/
1 SEX M
0 TRLR
`
	setupAncestorGedcom(t, content)

	ancestors := GetAllAncestors(Get(), "I1", 3)
	if _, ok := ancestors["I1"]; !ok {
		t.Error("expected self in ancestors")
	}
	if len(ancestors) != 1 {
		t.Errorf("expected only self, got %d entries", len(ancestors))
	}
}

func TestGetAllAncestors_ZeroDepth(t *testing.T) {
	content := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Child /Doe/
0 TRLR
`
	setupAncestorGedcom(t, content)
	ancestors := GetAllAncestors(Get(), "I1", 0)
	if len(ancestors) != 0 {
		t.Errorf("expected empty for depth 0, got %d entries", len(ancestors))
	}
}

func TestGetAllAncestors_FiltersNonBirthParentLinks(t *testing.T) {
	content := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Child /Doe/
1 FAMC @F1@
1 FAMC @F2@
2 PEDI adopted
0 @I2@ INDI
1 NAME BioParent /Doe/
1 SEX M
0 @I3@ INDI
1 NAME AdoptiveParent /Doe/
1 SEX M
0 @F1@ FAM
1 HUSB @I2@
1 CHIL @I1@
0 @F2@ FAM
1 HUSB @I3@
1 CHIL @I1@
0 TRLR
`
	setupAncestorGedcom(t, content)

	ancestors := GetAllAncestors(Get(), "I1", 2)

	if _, ok := ancestors["I2"]; !ok {
		t.Error("expected bio parent (birth) in ancestors")
	}
	if _, ok := ancestors["I3"]; ok {
		t.Error("expected adoptive parent (non-birth) to be filtered out")
	}
}

func TestFindAllRelationships_FirstCousins(t *testing.T) {
	content := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Grandpa /Doe/
1 SEX M
0 @I2@ INDI
1 Name Grandma /Doe/
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
	setupAncestorGedcom(t, content)

	links := FindAllRelationships(Get(), "I5", "I6", 10, 10)

	if len(links) == 0 {
		t.Fatal("expected at least one relationship link between cousins")
	}
	found := false
	for _, link := range links {
		if link.DepthA == 2 && link.DepthB == 2 {
			found = true
			if link.RelationLabel != "cousin" {
				t.Errorf("expected 'cousin', got %q", link.RelationLabel)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected depth 2/2 cousin link, got links: %+v", links)
	}
}

func TestFindAllRelationships_NoRelation(t *testing.T) {
	content := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME PersonA /Doe/
0 @I2@ INDI
1 NAME PersonB /Smith/
0 TRLR
`
	setupAncestorGedcom(t, content)

	links := FindAllRelationships(Get(), "I1", "I2", 5, 10)
	if len(links) != 0 {
		t.Errorf("expected no links, got %d", len(links))
	}
}

func TestFindAllRelationships_SamePerson(t *testing.T) {
	content := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Person /Doe/
0 TRLR
`
	setupAncestorGedcom(t, content)

	links := FindAllRelationships(Get(), "I1", "I1", 5, 10)
	if links != nil {
		t.Errorf("expected nil for same person, got %d links", len(links))
	}
}

func TestFindAllRelationships_Ancestor(t *testing.T) {
	content := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Child /Doe/
1 SEX M
1 FAMC @F1@
0 @I2@ INDI
1 NAME Father /Doe/
1 SEX M
0 @F1@ FAM
1 HUSB @I2@
1 CHIL @I1@
0 TRLR
`
	setupAncestorGedcom(t, content)

	links := FindAllRelationships(Get(), "I1", "I2", 5, 10)

	if len(links) == 0 {
		t.Fatal("expected relationship link between child and father")
	}
	if links[0].DepthA != 1 || links[0].DepthB != 0 {
		t.Errorf("expected depths 1/0 (parent-child), got %d/%d", links[0].DepthA, links[0].DepthB)
	}
}

func TestFindAllRelationships_UncleNephew(t *testing.T) {
	content := `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Grandpa /Doe/
1 SEX M
0 @I2@ INDI
1 NAME Parent /Doe/
1 SEX M
1 FAMC @F1@
0 @I3@ INDI
1 NAME Uncle /Doe/
1 SEX M
1 FAMC @F1@
0 @I4@ INDI
1 NAME Nephew /Doe/
1 SEX M
1 FAMC @F2@
0 @F1@ FAM
1 HUSB @I1@
1 CHIL @I2@
1 CHIL @I3@
0 @F2@ FAM
1 HUSB @I2@
1 CHIL @I4@
0 TRLR
`
	setupAncestorGedcom(t, content)

	links := FindAllRelationships(Get(), "I3", "I4", 10, 10)

	if len(links) == 0 {
		t.Fatal("expected relationship link between uncle and nephew")
	}
	if links[0].DepthA != 1 || links[0].DepthB != 2 {
		t.Errorf("expected depths 1/2 (uncle/nephew), got %d/%d", links[0].DepthA, links[0].DepthB)
	}
}

func TestFilterShadowedLinks(t *testing.T) {
	links := []RelationshipLink{
		{
			CommonAncestors: []string{"I1"},
			PathFromA:       []string{"A", "P", "GP", "I1"},
			PathFromB:       []string{"B", "Q", "GP", "I1"},
			DepthA:          3,
			DepthB:          3,
		},
		{
			CommonAncestors: []string{"GP"},
			PathFromA:       []string{"A", "P", "GP"},
			PathFromB:       []string{"B", "Q", "GP"},
			DepthA:          2,
			DepthB:          2,
		},
	}

	filtered := filterShadowedLinks(links)

	if len(filtered) != 1 {
		t.Fatalf("expected 1 link after filtering, got %d", len(filtered))
	}
	if filtered[0].CommonAncestors[0] != "GP" {
		t.Errorf("expected GP (closer ancestor) to survive, got %s", filtered[0].CommonAncestors[0])
	}
}

func TestFilterShadowedLinks_NoShadow(t *testing.T) {
	links := []RelationshipLink{
		{
			CommonAncestors: []string{"I1"},
			PathFromA:       []string{"A", "I1"},
			PathFromB:       []string{"B", "I1"},
			DepthA:          1,
			DepthB:          1,
		},
		{
			CommonAncestors: []string{"I2"},
			PathFromA:       []string{"A", "X", "I2"},
			PathFromB:       []string{"B", "Y", "I2"},
			DepthA:          2,
			DepthB:          2,
		},
	}

	filtered := filterShadowedLinks(links)

	if len(filtered) != 2 {
		t.Errorf("expected 2 links (different lineages), got %d", len(filtered))
	}
}

func TestComputeKinshipCoefficient_MultiplePaths(t *testing.T) {
	link := RelationshipLink{
		DepthA:          2,
		DepthB:          2,
		CommonAncestors: []string{"I1", "I2"},
	}
	coeff := ComputeKinshipCoefficient([]RelationshipLink{link})
	expected := 2.0 * 0.5 * 0.5 * 0.5 * 0.5 * 0.5 // 2 * (0.5)^5 = 0.0625
	if coeff != expected {
		t.Errorf("expected %f, got %f", expected, coeff)
	}
}
