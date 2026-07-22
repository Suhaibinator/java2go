package transpiler

import "testing"

func TestMethodLocalClassCapturesEnclosingInstance(t *testing.T) {
	src := `
public class LocalClassEnclosingInstanceProgram {
    int base;

    LocalClassEnclosingInstanceProgram(int base) {
        this.base = base;
    }

    int bump(int value) {
        return base + value;
    }

    int compute(int seed) {
        class LocalValue {
            int encoded() {
                return base * 100
                    + bump(seed) * 10
                    + LocalClassEnclosingInstanceProgram.this.base;
            }
        }

        return new LocalValue().encoded();
    }

    public static int run() {
        return new LocalClassEnclosingInstanceProgram(3).compute(4);
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestLocalClassEnclosingInstanceParity(t *testing.T) {
    if got := Run(); got != 373 {
        t.Fatalf("Run() = %d, want outer field/method/qualified-this parity", got)
    }
}
`)
}
