package transpiler

import "testing"

// Method-local classes are absent from the global symbol tree. An anonymous
// subclass still makes the local superclass constructor most-derived-aware, and
// its override must observe the exact local superclass subobject being built.
func TestConstructorSubobjectAttachment_AnonymousSubclassOfLocalClass(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
public class LocalAnonymousConstructorProgram {
    public static int run() {
        class LocalBase {
            int inherited;
            int observed;

            LocalBase(int value) {
                inherited = value;
                observed = probe();
            }

            int probe() {
                return -1;
            }
        }

        var instance = new LocalBase(7) {
            int probe() {
                return inherited + 2;
            }
        };
        return instance.observed * 10 + instance.inherited;
    }
}
`, 97)
}

// A raw generic superclass is embedded using the transpiler's erased
// upper-bound/Object instantiation. The installer signature must use that same
// concrete Go type rather than emitting an invalid bare generic pointer.
func TestConstructorSubobjectAttachment_RawGenericSuperclassUsesErasedHookType(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
class RawConstructorBase<T> {
    int observed;

    RawConstructorBase() {
        observed = probe();
    }

    int probe() {
        return -1;
    }
}

class RawConstructorChild extends RawConstructorBase {
    int probe() {
        return 8;
    }
}

public class RawConstructorProgram {
    public static int run() {
        return new RawConstructorChild().observed;
    }
}
`, 8)
}
