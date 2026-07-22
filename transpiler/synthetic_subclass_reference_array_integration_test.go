package transpiler

import (
	"strings"
	"testing"
)

func TestReferenceArrayIdentityDetectsAnonymousConcreteSubclass(t *testing.T) {
	src := `
public class AnonymousSubclassReferenceArrayProgram {
    static class Base {
        int inherited() { return 40; }
        int overridden() { return 1; }
    }

    static class Wrong { }

    public static int run() {
        Base value = new Base() {
            int overridden() { return 2; }
            int secondMethod() { return 3; }
        };

        Base[] actual = new Base[1];
        actual[0] = value;
        Object[] erased = actual;
        erased[0] = value;

        int score = actual[0].inherited() + actual[0].overridden();
        try {
            erased[0] = new Wrong();
            score += 1000;
        } catch (ArrayStoreException expected) {
            score += 100;
        }
        return score;
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	for _, fragment := range []string{
		`type AnonymousSubclassReferenceArrayProgrambase struct {`,
		`*stdjava.ObjectInfo`,
		`Java2goReferenceView`,
		`stdjava.ReferenceArrayAssign[*AnonymousSubclassReferenceArrayProgrambase]`,
	} {
		if !strings.Contains(flat, fragment) {
			t.Fatalf("anonymous concrete subclass did not keep Base hierarchy identity; missing %q:\n%s", fragment, out)
		}
	}

	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestAnonymousSubclassReferenceArrayParity(t *testing.T) {
    if got := Run(); got != 142 {
        t.Fatalf("Run() = %d, want inherited + overridden dispatch and rejected incompatible store", got)
    }
}
`)
}
