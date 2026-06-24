package gedcom

import (
	"fmt"
	"math"
	"strings"
)

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func absInt(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func ComputeRelationLabel(depthA, depthB int, sex string) string {
	if depthA < 1 || depthB < 1 {
		if depthA == 0 && depthB > 0 {
			return AncestorLabel(depthB, false, sex)
		}
		if depthB == 0 && depthA > 0 {
			return AncestorLabel(depthA, false, sex)
		}
		if depthA == 0 && depthB == 0 {
			return "same person"
		}
		return "unknown"
	}

	if depthA == 1 && depthB == 1 {
		return "frère/soeur"
	}

	deg := minInt(depthA, depthB)
	ret := absInt(depthA - depthB)
	// si sex M alors "cousin", si F alors "cousine"
	sexLabel := ""
	if sex == "F" {
		sexLabel = "e"
	}

	switch {
	case deg == 2 && ret == 0:
		return "cousin" + sexLabel
	case deg == 2 && ret == 1:
		return "cousin" + sexLabel + " d'un parent"
	case deg == 3 && ret == 0:
		return "cousin" + sexLabel + " issu de germains"
	case deg == 3 && ret == 1:
		return "cousin" + sexLabel + " issu de germains d'un parent"
	default:
		degStr := fmt.Sprintf("%de", deg)
		if deg == 1 {
			degStr = "1er"
		}
		if ret == 0 {
			return fmt.Sprintf("cousin"+sexLabel+" au %s degré", degStr)
		}
		retStr := fmt.Sprintf("%d fois", ret)
		if ret == 1 {
			retStr = "une fois"
		}
		return fmt.Sprintf("cousin"+sexLabel+" au %s degré, %s retiré", degStr, retStr)
	}
}

var ordinals = map[int]string{
	1: "1re", 2: "2e", 3: "3e", 4: "4e", 5: "5e",
	6: "6e", 7: "7e", 8: "8e", 9: "9e", 10: "10e",
}

func AncestorLabel(depth int, isCouple bool, sex string) string {
	if isCouple {
		switch depth {
		case 1:
			return "parents"
		case 2:
			return "grands-parents"
		case 3:
			return "arrière-grands-parents"
		default:
			suffix, ok := ordinals[depth]
			if !ok {
				suffix = fmt.Sprintf("%de", depth)
			}
			return fmt.Sprintf("ancêtres à la %s génération", suffix)
		}
	}

	switch depth {
	case 1:
		if sex == "F" {
			return "mère"
		}
		return "père"
	case 2:
		if sex == "F" {
			return "grand-mère"
		}
		return "grand-père"
	case 3:
		if sex == "F" {
			return "arrière-grand-mère"
		}
		return "arrière-grand-père"
	default:
		suffix, ok := ordinals[depth]
		if !ok {
			suffix = fmt.Sprintf("%de", depth)
		}
		return fmt.Sprintf("ancêtre à la %s génération", suffix)
	}
}

func ComputeKinshipCoefficient(links []RelationshipLink) float64 {
	var coeff float64
	for _, link := range links {
		n := float64(len(link.CommonAncestors))
		coeff += n * math.Pow(0.5, float64(link.DepthA+link.DepthB+1))
	}
	return coeff
}

func FormatNameWithID(name, id string) string {
	name = strings.ReplaceAll(strings.ReplaceAll(name, "\"", ""), "/", "")
	name = strings.TrimSpace(name)
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return id
	}
	surName := strings.ToUpper(parts[len(parts)-1])
	givenNames := strings.Join(parts[:len(parts)-1], " ")
	return givenNames + " " + surName + " (" + id + ")"
}
