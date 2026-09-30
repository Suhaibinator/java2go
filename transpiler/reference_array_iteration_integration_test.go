package transpiler

import (
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/stdjava"
)

func TestEnhancedForArrayIterationPreservesJavaTimingAndViews(t *testing.T) {
	src := `
public class ReferenceArrayIterationProgram {
    private static int calls;

    static class Base {
        int value;
        Base(int value) { this.value = value; }
    }

    static class Child extends Base {
        Child(int value) { super(value); }
    }

    private static String[] next() {
        calls++;
        return new String[] { "a", "b", "c" };
    }

    public static String run() {
        calls = 0;
        String trace = "";
        String[] selected = next();
        int index = 0;
        for (String value : selected) {
            trace = trace + value;
            if (index == 0) selected[1] = "z";
            index++;
            if (index == 2) break;
        }
		String inferred = "";
		for (var item : new String[] { "v", "w" }) {
			inferred = inferred + item;
		}

        Base[] objects = new Child[] { new Child(1), new Child(2) };
        int objectScore = 0;
        index = 0;
        for (Base object : objects) {
            objectScore = objectScore * 10 + object.value;
            if (index == 0) objects[1] = new Child(9);
            index++;
        }

        int[][] matrix = new int[][] {
            new int[] { 1, 2 },
            new int[] { 3, 4 }
        };
        int digits = 0;
        for (int[] row : matrix) {
            for (int value : row) {
                digits = digits * 10 + value;
            }
        }

        int nullScore = 0;
        String[] missingReferences = null;
        try {
            for (String ignored : missingReferences) nullScore++;
        } catch (NullPointerException expected) {
            nullScore += 10;
        }
        int[] missingPrimitives = null;
        try {
            for (int ignored : missingPrimitives) nullScore++;
        } catch (NullPointerException expected) {
            nullScore += 20;
        }

        for (String ignored : next()) {
            break;
        }
        try {
            for (String ignored : next()) {
                throw new IllegalStateException("stop");
            }
        } catch (IllegalStateException expected) {
            trace = trace + "t";
        }
        return trace + ":" + inferred + ":" + objectScore + ":" + digits + ":" + nullScore + ":" + calls;
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	for _, fragment := range []string{
		`range stdjava.ReferenceArrayIterationElements(selected)`,
		`value := stdjava.ObjectView[string]`,
		`item := stdjava.ObjectView[string]`,
		`object := stdjava.ObjectView[*ReferenceArrayIterationProgrambase]`,
		`row := stdjava.ObjectView[*stdjava.PrimitiveArray[int32]]`,
		`range stdjava.PrimitiveArrayIterationElements(row)`,
		`range stdjava.ReferenceArrayIterationElements(missingReferences)`,
		`range stdjava.PrimitiveArrayIterationElements(missingPrimitives)`,
	} {
		if !strings.Contains(flat, fragment) {
			t.Fatalf("enhanced-for lowering is missing %q:\n%s", fragment, out)
		}
	}
	if strings.Contains(flat, `stdjava.TypeID("var")`) || strings.Contains(flat, `*var`) {
		t.Fatalf("reference-array var binding retained the keyword instead of its inferred element type:\n%s", out)
	}

	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestEnhancedForArrayRuntime(t *testing.T) {
    if got := Run(); got != "azt:vw:19:1234:30:3" {
        t.Fatalf("Run() = %q, want exact Java enhanced-for behavior", got)
    }
}
`)
}

type enhancedForIterationView struct {
	value int
}

type enhancedForIterationStored struct {
	*stdjava.ObjectInfo
	view *enhancedForIterationView
}

func newEnhancedForIterationStored(dynamicType, requestedType stdjava.TypeID, value int, conversions *int) *enhancedForIterationStored {
	stored := &enhancedForIterationStored{view: &enhancedForIterationView{value: value}}
	stored.ObjectInfo = stdjava.NewObjectInfo(dynamicType, func(requested stdjava.TypeID) any {
		(*conversions)++
		if requested == requestedType {
			return stored.view
		}
		return stored
	})
	return stored
}

func TestReferenceArrayIterationConvertsOnlyVisitedCurrentElements(t *testing.T) {
	baseType := stdjava.TypeID("java2go.test.EnhancedForIterationBase")
	childType := stdjava.TypeID("java2go.test.EnhancedForIterationChild")
	stdjava.RegisterJavaType(baseType, stdjava.ObjectTypeID)
	stdjava.RegisterJavaType(childType, baseType)

	conversions := 0
	array := stdjava.NewReferenceArray(3, baseType)
	first := newEnhancedForIterationStored(childType, baseType, 1, &conversions)
	stdjava.ReferenceArraySet(array, 0, first)
	stdjava.ReferenceArraySet(array, 1, newEnhancedForIterationStored(childType, baseType, 2, &conversions))
	stdjava.ReferenceArraySet(array, 2, newEnhancedForIterationStored(childType, baseType, 3, &conversions))
	if got := stdjava.ObjectView[*enhancedForIterationView](first, stdjava.ObjectTypeID); got != first.view {
		t.Fatal("erased Object descriptor did not recover the instantiated superclass Go view")
	}
	conversions = 0

	visited := make([]int, 0, 2)
	for index, raw := range stdjava.ReferenceArrayIterationElements(array) {
		current := stdjava.ObjectView[*enhancedForIterationView](raw, baseType)
		visited = append(visited, current.value)
		if index == 0 {
			stdjava.ReferenceArraySet(array, 1, newEnhancedForIterationStored(childType, baseType, 9, &conversions))
		}
		if index == 1 {
			break
		}
	}
	if got := len(visited); got != 2 || visited[0] != 1 || visited[1] != 9 {
		t.Fatalf("visited values = %v, want [1 9] after mutating the future slot", visited)
	}
	if conversions != 2 {
		t.Fatalf("break converted %d elements, want only the 2 visited elements", conversions)
	}

	conversions = 0
	marker := &struct{}{}
	func() {
		defer func() {
			if recovered := recover(); recovered != marker {
				t.Fatalf("recovered %v, want iteration marker", recovered)
			}
		}()
		for _, raw := range stdjava.ReferenceArrayIterationElements(array) {
			_ = stdjava.ObjectView[*enhancedForIterationView](raw, baseType)
			panic(marker)
		}
	}()
	if conversions != 1 {
		t.Fatalf("throw converted %d elements, want only the current element", conversions)
	}
}
