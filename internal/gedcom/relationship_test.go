package gedcom

import (
	"math"
	"testing"
)

func TestComputeRelationLabel(t *testing.T) {
	tests := []struct {
		depthA, depthB int
		sex            string
		expected       string
	}{
		{1, 1, "M", "frère/soeur"},
		{2, 2, "M", "cousin"},
		{3, 2, "M", "cousin d'un parent"},
		{3, 3, "M", "cousin issu de germains"},
		{4, 3, "M", "cousin issu de germains d'un parent"},
		{4, 4, "M", "cousin au 4e degré"},
		{5, 3, "M", "cousin au 3e degré, 2 fois retiré"},
		{1, 2, "M", "cousin au 1er degré, une fois retiré"},
		{0, 4, "M", "ancêtre à la 4e génération"},
		{0, 4, "F", "ancêtre à la 4e génération"},
		{0, 1, "M", "père"},
		{0, 1, "F", "mère"},
		{0, 2, "M", "grand-père"},
		{0, 2, "F", "grand-mère"},
		{0, 3, "M", "arrière-grand-père"},
		{0, 3, "F", "arrière-grand-mère"},
		{4, 0, "M", "ancêtre à la 4e génération"},
		{0, 0, "M", "same person"},
		{-1, 2, "M", "unknown"},
		{1, -1, "M", "unknown"},
	}

	for _, tt := range tests {
		got := ComputeRelationLabel(tt.depthA, tt.depthB, tt.sex)
		if got != tt.expected {
			t.Errorf("ComputeRelationLabel(%d, %d, %q) = %q; want %q",
				tt.depthA, tt.depthB, tt.sex, got, tt.expected)
		}
	}
}

func TestAncestorLabel(t *testing.T) {
	tests := []struct {
		depth    int
		isCouple bool
		sex      string
		expected string
	}{
		{1, false, "M", "père"},
		{1, false, "F", "mère"},
		{1, true, "", "parents"},
		{2, false, "M", "grand-père"},
		{2, false, "F", "grand-mère"},
		{2, true, "", "grands-parents"},
		{3, false, "F", "arrière-grand-mère"},
		{3, false, "M", "arrière-grand-père"},
		{3, true, "", "arrière-grands-parents"},
		{4, false, "M", "ancêtre à la 4e génération"},
		{4, false, "F", "ancêtre à la 4e génération"},
		{5, true, "", "ancêtres à la 5e génération"},
		{6, false, "M", "ancêtre à la 6e génération"},
	}

	for _, tt := range tests {
		got := AncestorLabel(tt.depth, tt.isCouple, tt.sex)
		if got != tt.expected {
			t.Errorf("AncestorLabel(%d, %v, %q) = %q; want %q",
				tt.depth, tt.isCouple, tt.sex, got, tt.expected)
		}
	}
}

func TestComputeKinshipCoefficient(t *testing.T) {
	tests := []struct {
		name     string
		links    []RelationshipLink
		expected float64
		delta    float64
	}{
		{
			name: "single cousin germain ancestor",
			links: []RelationshipLink{
				{DepthA: 2, DepthB: 2, CommonAncestors: []string{"I1"}},
			},
			expected: 1 * math.Pow(0.5, 2+2+1), // 0.03125
			delta:    0.0001,
		},
		{
			name: "couple at depth 2/2",
			links: []RelationshipLink{
				{DepthA: 2, DepthB: 2, CommonAncestors: []string{"I1", "I2"}},
			},
			expected: 2 * math.Pow(0.5, 2+2+1), // 0.0625
			delta:    0.0001,
		},
		{
			name: "three links mixed depths",
			links: []RelationshipLink{
				{DepthA: 2, DepthB: 2, CommonAncestors: []string{"I1", "I2"}},
				{DepthA: 3, DepthB: 2, CommonAncestors: []string{"I3"}},
				{DepthA: 4, DepthB: 3, CommonAncestors: []string{"I4", "I5"}},
			},
			expected: 2*math.Pow(0.5, 5) + 1*math.Pow(0.5, 6) + 2*math.Pow(0.5, 8),
			delta:    0.0001,
		},
		{
			name:     "empty links",
			links:    []RelationshipLink{},
			expected: 0,
			delta:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeKinshipCoefficient(tt.links)
			if math.Abs(got-tt.expected) > tt.delta {
				t.Errorf("ComputeKinshipCoefficient = %f; want %f ± %f", got, tt.expected, tt.delta)
			}
		})
	}
}
