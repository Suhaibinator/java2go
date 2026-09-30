package transpiler

import "testing"

func TestMethodLocalLeafClassRetainsExactArrayComponentIdentity(t *testing.T) {
	src := `
public class LocalLeafArrayIdentityProgram {
    static class Wrong { }

    public static int run() {
        class LocalValue {
            int value() { return 7; }
        }

        LocalValue[] exact = new LocalValue[] { new LocalValue() };
        Object[] erased = exact;
        erased[0] = new LocalValue();
        int score = exact[0].value();
        try {
            erased[0] = new Wrong();
            score += 100;
        } catch (ArrayStoreException expected) {
            score += 10;
        }
        return score;
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestLocalLeafArrayIdentityParity(t *testing.T) {
    if got := Run(); got != 17 {
        t.Fatalf("Run() = %d, want exact leaf-local array identity and store checks", got)
    }
}
`)
}
