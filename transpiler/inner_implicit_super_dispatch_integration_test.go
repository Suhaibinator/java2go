package transpiler

import "testing"

func TestInnerImplicitSuperPreservesMostDerivedConstructorDispatch(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class InnerImplicitSuperDispatchProgram {
    class Parent {
        int observed;

        Parent() {
            observed = value();
        }

        int value() {
            return 1;
        }
    }

    class Child extends Parent {
        Child() {}

        int value() {
            return 9;
        }
    }

    String exercise() {
        return "" + new Child().observed;
    }

    public static String run() {
        return new InnerImplicitSuperDispatchProgram().exercise();
    }
}
`)

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestInnerImplicitSuperDispatchRuntime(t *testing.T) {
    if got := Run(); got != "9" {
        t.Fatalf("Run() = %q, want 9", got)
    }
}
`)
}
