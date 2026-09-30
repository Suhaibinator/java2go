package transpiler

import (
	"fmt"
	"testing"
)

// A Java superclass constructor can write inherited state and then dispatch
// virtually into the most-derived receiver. The override must observe that
// write through the exact superclass subobject that is still under
// construction. Cover explicit super(...), implicit super(), and a terminal
// this(...) delegation in the superclass without changing the WithSelf ABI.
func TestConstructorSubobjectAttachment_InheritedStateVisibleDuringVirtualDispatch(t *testing.T) {
	out := renderGoFileFromJava(t, `
class ConstructorStateBase {
    int inherited;
    int observed;

    ConstructorStateBase() {
        this(4);
    }

    ConstructorStateBase(int value) {
        inherited = value;
        observed = probe();
    }

    int probe() {
        return -1;
    }
}

class ExplicitConstructorStateChild extends ConstructorStateBase {
    ExplicitConstructorStateChild(int value) {
        super(value);
    }

    int probe() {
        return inherited + 2;
    }
}

class ImplicitConstructorStateChild extends ConstructorStateBase {
    ImplicitConstructorStateChild() {
    }

    int probe() {
        return inherited + 3;
    }
}

public class ConstructorSubobjectAttachmentProgram {
    public static int run() {
        ExplicitConstructorStateChild explicit =
            new ExplicitConstructorStateChild(7);
        ImplicitConstructorStateChild implicit =
            new ImplicitConstructorStateChild();
        return explicit.observed * 1000
            + explicit.inherited * 100
            + implicit.observed * 10
            + implicit.inherited;
    }
}
`)
	runGeneratedWithStdjava(t, out, fmt.Sprintf(`
package main

import "testing"

func TestConstructorSubobjectAttachmentResult(t *testing.T) {
	if got := Run(); got != int32(%d) {
		t.Fatalf("Run() = %%d, want %%d", got, int32(%d))
	}
}
`, 9774, 9774))
}
