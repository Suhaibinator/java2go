package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestDefinitionJavaTypePreservesNestedShadowedTypeParameterDeclaration(t *testing.T) {
	parameters := []symbol.TypeParam{
		symbol.NewTypeParam("T", nil),
		symbol.NewTypeParam("T", nil),
		symbol.NewTypeParam("T", nil),
	}
	symbol.DisambiguateTypeParamGoNames(parameters)

	tests := []struct {
		name        string
		declaration *symbol.TypeParamDeclaration
		want        string
	}{
		{name: "class", declaration: parameters[0].Declaration, want: "List<T>"},
		{name: "method", declaration: parameters[1].Declaration, want: "List<T2>"},
		{name: "local", declaration: parameters[2].Declaration, want: "List<T3>"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition := &symbol.Definition{
				OriginalType:          "List<T>",
				TypeParameterBindings: map[string]*symbol.TypeParamDeclaration{"T": test.declaration},
			}
			if got := definitionJavaType(definition); got != test.want {
				t.Fatalf("definitionJavaType() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDefinitionJavaTypeSubstitutesWildcardsAndArrays(t *testing.T) {
	parameter := symbol.NewTypeParam("T", nil)
	parameter.Declaration.GoName = "T2"
	definition := &symbol.Definition{
		OriginalType:          "Map<String, ? extends T[]>",
		TypeParameterBindings: map[string]*symbol.TypeParamDeclaration{"T": parameter.Declaration},
	}

	if got, want := definitionJavaType(definition), "Map<String, ? extends T2[]>"; got != want {
		t.Fatalf("definitionJavaType() = %q, want %q", got, want)
	}
}

// Class, method, and local-class declarations may all shadow T while values of
// each type travel through a nested generic Box<T>. This exercises persisted
// declaration provenance through parsing, local capture, hoisting, and runtime
// method/field lowering rather than only the string substitution helper.
func TestShadowedTypeParameterProvenanceThroughNestedGenericCapture(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
interface NestedClassMark {
    int classCode();
}

interface NestedMethodMark {
    int methodCode();
}

interface NestedLocalMark {
    int localCode();
}

public class ShadowedNestedGenericProgram<T extends NestedClassMark> {
    static class Box<X> {
        X value;

        Box(X value) {
            this.value = value;
        }

        X get() {
            return value;
        }
    }

    static class ClassValue implements NestedClassMark {
        int code;
        ClassValue(int code) { this.code = code; }
        public int classCode() { return code; }
    }

    static class MethodValue implements NestedMethodMark {
        int code;
        MethodValue(int code) { this.code = code; }
        public int methodCode() { return code; }
    }

    static class LocalValue implements NestedLocalMark {
        int code;
        LocalValue(int code) { this.code = code; }
        public int localCode() { return code; }
    }

    Box<T> classBox;

    ShadowedNestedGenericProgram(Box<T> classBox) {
        this.classBox = classBox;
    }

    <T extends NestedMethodMark> int score(Box<T> methodBox, int localCode) {
		Box<T> capturedMethodBox = methodBox;

        class Local<T extends NestedLocalMark> {
            Box<T> localBox;

            Local(Box<T> localBox) {
                this.localBox = localBox;
            }

            int read() {
                return classBox.get().classCode() * 100
					+ capturedMethodBox.get().methodCode() * 10
                    + localBox.get().localCode();
            }
        }

        return new Local<LocalValue>(
                new Box<LocalValue>(new LocalValue(localCode))).read();
    }

    public static String run() {
        return new ShadowedNestedGenericProgram<ClassValue>(
                    new Box<ClassValue>(new ClassValue(2)))
                .score(new Box<MethodValue>(new MethodValue(3)), 4)
            + ":"
            + new ShadowedNestedGenericProgram<ClassValue>(
                    new Box<ClassValue>(new ClassValue(5)))
                .score(new Box<MethodValue>(new MethodValue(6)), 7);
    }
}
`, "234:567")
}

const typeParameterNamingLifecycleSource = `
interface NamingMark {
    int code();
}

public class TypeParameterNamingLifecycleProgram<T extends NamingMark> {
    static class Value implements NamingMark {
        int number;
        Value(int number) { this.number = number; }
        public int code() { return number; }
    }

    T outer;

    TypeParameterNamingLifecycleProgram(T outer) {
        this.outer = outer;
    }

    <T1 extends NamingMark> int first(T1 value) {
        return outer.code() * 10 + value.code();
    }

    <T extends NamingMark> int shadow(T methodValue, int localCode) {
        class Local<T extends NamingMark> {
            T localValue;
            Local(T localValue) { this.localValue = localValue; }

            int read() {
                return outer.code() * 100
                    + methodValue.code() * 10
                    + localValue.code();
            }
        }
        return new Local<Value>(new Value(localCode)).read();
    }

    public static String run() {
        TypeParameterNamingLifecycleProgram<Value> value =
            new TypeParameterNamingLifecycleProgram<Value>(new Value(1));
        return value.first(new Value(2)) + ":" + value.shadow(new Value(3), 4);
    }
}
`

func TestTypeParameterNamingLifecycleKeepsPreviouslyEmittedBindersStable(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, typeParameterNamingLifecycleSource, "12:134")
}

func TestTypeParameterNamingLifecycleRepeatedConversionIsByteIdentical(t *testing.T) {
	first := renderGoFileFromJava(t, typeParameterNamingLifecycleSource)
	second := renderGoFileFromJava(t, typeParameterNamingLifecycleSource)
	if first != second {
		t.Fatal("repeated conversion changed generated binder allocation")
	}
}

func TestEnhancedForGenericCaptureRetainsOuterDeclaration(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
interface LoopCaptureMark {
    int code();
}

public class EnhancedForGenericCaptureProgram<T extends LoopCaptureMark> {
    static class Value implements LoopCaptureMark {
        int number;
        Value(int number) { this.number = number; }
        public int code() { return number; }
    }

    int score(T[] values) {
        int total = 0;
        for (T item : values) {
            class Local<T extends LoopCaptureMark> {
                int read() {
                    return item.code();
                }
            }
            total += new Local<Value>().read();
        }
        return total;
    }

    public static String run() {
        return "" + new EnhancedForGenericCaptureProgram<Value>()
            .score(new Value[] { new Value(2), new Value(3) });
    }
}
`, "5")
}
