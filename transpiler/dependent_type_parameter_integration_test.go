package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestJavaDependentTypeParameterAssignable_UsesBoundDeclarationIdentity(t *testing.T) {
	outerB := symbol.NewTypeParam("B", []symbol.JavaType{{Original: "OuterRoot"}})
	methodB := symbol.NewTypeParam("B", []symbol.JavaType{{Original: "MethodRoot"}})
	methodT := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "B"}})
	localB := symbol.NewTypeParam("B", []symbol.JavaType{{Original: "LocalRoot"}})

	parameters := []symbol.TypeParam{outerB, methodB, methodT, localB}
	symbol.DisambiguateTypeParamGoNames(parameters)
	symbol.BindTypeParameterBounds(parameters[2:3], parameters[:3])

	ctx := Ctx{currentClass: &symbol.ClassScope{TypeParameters: parameters}}
	if !javaDependentTypeParameterAssignable(
		methodT.EmittedName(),
		methodB.EmittedName(),
		ctx,
	) {
		t.Fatal("T bound to the method B was rebound to a later same-named B")
	}
	if javaDependentTypeParameterAssignable(
		methodT.EmittedName(),
		localB.EmittedName(),
		ctx,
	) {
		t.Fatal("T was incorrectly treated as extending the later same-named B")
	}
}

// A dependent T-to-B widening must evaluate the source expression once, keep
// the most precise inferred return type at an ordinary call, and canonicalize a
// typed nil pointer back to Java null when B is an interface view.
func TestDependentTypeParameterWidening_NullPrecisionAndSingleEvaluation(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class DependentTypeParameterWideningProgram {
    interface Root {
        int value();
    }

    static class Impl implements Root {
        int number;

        Impl(int number) {
            this.number = number;
        }

        public int value() {
            return number;
        }
    }

    static int effects;

    static <B extends Root, T extends B> B widen(T value) {
        return effects++ == 0 ? value : value;
    }

    static <X> boolean isNull(X value) {
        return value == null;
    }

    public static String run() {
        effects = 0;
        Impl precise = widen(new Impl(7));
        Root nullValue = DependentTypeParameterWideningProgram.<Root, Impl>widen(null);
        return precise.value() + ":" + effects + ":" + isNull(nullValue);
    }
}
`, "7:2:true")
}

// A concrete Java class upper bound admits subclasses, while the representable
// Go constraint *Base denotes its exact generated subobject. Inference and
// argument target-typing must therefore agree on that Base view.
func TestDependentTypeParameterWidening_ConcreteClassErasure(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class DependentConcreteBoundProgram {
    static class Base {
        int number;

        Base(int number) {
            this.number = number;
        }
    }

    static class Child extends Base {
        Child(int number) {
            super(number);
        }
    }

    static <B extends Base, T extends B> B widen(T value) {
        return value;
    }

    public static String run() {
        Base widened = widen(new Child(9));
        return "" + widened.number;
    }
}
`, "9")
}
