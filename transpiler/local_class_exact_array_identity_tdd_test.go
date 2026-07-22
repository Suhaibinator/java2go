package transpiler

import "testing"

func TestMethodLocalConcreteClassRetainsExactArrayComponentIdentity(t *testing.T) {
	src := `
public class LocalExactArrayIdentityProgram {
    static class Base {
        int value() { return 1; }
    }

    public static int run() {
        class Child extends Base {
            int value() { return 2; }
        }

        Child[] exact = new Child[] { new Child() };
        Object[] erased = exact;
        int score = exact[0].value();
        try {
            erased[0] = new Base();
            score += 100;
        } catch (ArrayStoreException expected) {
            score += 10;
        }
        erased[0] = new Child();
        return score * 10 + exact[0].value();
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestLocalExactArrayIdentityParity(t *testing.T) {
    if got := Run(); got != 122 {
        t.Fatalf("Run() = %d, want exact local Child[] identity and store checks", got)
    }
}
`)
}
