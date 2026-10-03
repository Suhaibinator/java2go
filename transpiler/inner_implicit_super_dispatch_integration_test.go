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

import ("slices"; "testing")

func TestInnerImplicitSuperDispatchRuntime(t *testing.T) {
    got := Run()
    if got == nil {
        t.Fatal("Run() returned null")
    }
    if !slices.Equal(got.UTF16Copy(), []uint16{'9'}) {
        t.Fatalf("Run() UTF16 = %v, want 9", got.UTF16Copy())
    }
}
`)
}
