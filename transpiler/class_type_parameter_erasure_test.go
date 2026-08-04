package transpiler

import (
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestDirectOwnerTypeParameterErasure_FirstBoundRules(t *testing.T) {
	tests := []struct {
		name       string
		parameters func() []symbol.TypeParam
		target     int
		want       string
	}{
		{
			name: "unbounded erases to Object",
			parameters: func() []symbol.TypeParam {
				return []symbol.TypeParam{symbol.NewTypeParam("T", nil)}
			},
			want: "Object",
		},
		{
			name: "interface first bound",
			parameters: func() []symbol.TypeParam {
				return []symbol.TypeParam{symbol.NewTypeParam("T", []symbol.JavaType{{Original: "Numbered"}})}
			},
			want: "Numbered",
		},
		{
			name: "concrete first bound",
			parameters: func() []symbol.TypeParam {
				return []symbol.TypeParam{symbol.NewTypeParam("T", []symbol.JavaType{{Original: "Base"}})}
			},
			want: "Base",
		},
		{
			name: "dependent bound follows declaration identity",
			parameters: func() []symbol.TypeParam {
				parameters := []symbol.TypeParam{
					symbol.NewTypeParam("B", []symbol.JavaType{{Original: "Root"}}),
					symbol.NewTypeParam("T", []symbol.JavaType{{Original: "B"}}),
				}
				symbol.BindTypeParameterBounds(parameters, parameters)
				return parameters
			},
			target: 1,
			want:   "Root",
		},
		{
			name: "intersection uses only first bound erasure",
			parameters: func() []symbol.TypeParam {
				return []symbol.TypeParam{symbol.NewTypeParam("T", []symbol.JavaType{
					{Original: "Base"},
					{Original: "Ranked"},
				})}
			},
			want: "Base",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parameters := test.parameters()
			target := parameters[test.target]
			owner := &symbol.ClassScope{TypeParameters: parameters}
			definition := &symbol.Definition{
				OriginalType:        target.Name,
				DirectTypeParameter: target.Declaration,
			}

			got, ok := directOwnerTypeParameterErasure(owner, definition)
			if !ok {
				t.Fatal("direct owner type parameter was not recognized")
			}
			if got != test.want {
				t.Fatalf("erasure = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDirectOwnerTypeParameterErasure_RequiresBareRankZeroUse(t *testing.T) {
	parameter := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "Root"}})
	owner := &symbol.ClassScope{TypeParameters: []symbol.TypeParam{parameter}}

	tests := []struct {
		name     string
		javaType string
		direct   *symbol.TypeParamDeclaration
		wantOK   bool
	}{
		{name: "bare", javaType: "T", direct: parameter.Declaration, wantOK: true},
		{name: "array", javaType: "T[]", direct: parameter.Declaration},
		{name: "nested", javaType: "Box<T>", direct: parameter.Declaration},
		{name: "qualified spelling", javaType: "pkg.T", direct: parameter.Declaration},
		{name: "missing provenance", javaType: "T"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, ok := directOwnerTypeParameterErasure(owner, &symbol.Definition{
				OriginalType:        test.javaType,
				DirectTypeParameter: test.direct,
			})
			if ok != test.wantOK {
				t.Fatalf("recognized = %t, want %t", ok, test.wantOK)
			}
		})
	}
}

func TestDirectOwnerTypeParameterErasure_ClassParameterBeatsNameShadowing(t *testing.T) {
	classT := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "ClassRoot"}})
	methodT := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "MethodRoot"}})
	owner := &symbol.ClassScope{TypeParameters: []symbol.TypeParam{classT}}

	classDefinition := &symbol.Definition{
		OriginalType:        "T",
		DirectTypeParameter: classT.Declaration,
	}
	if got, ok := directOwnerTypeParameterErasure(owner, classDefinition); !ok || got != "ClassRoot" {
		t.Fatalf("class-owned T erasure = %q, %t; want ClassRoot, true", got, ok)
	}

	shadowingMethodDefinition := &symbol.Definition{
		OriginalType:        "T",
		DirectTypeParameter: methodT.Declaration,
	}
	if got, ok := directOwnerTypeParameterErasure(owner, shadowingMethodDefinition); ok {
		t.Fatalf("same-named method T was treated as owner T with erasure %q", got)
	}

	legacyOwner := &symbol.ClassScope{TypeParameters: []symbol.TypeParam{{
		Name:   "T",
		Bounds: []symbol.JavaType{{Original: "LegacyRoot"}},
	}}}
	if got, ok := directOwnerTypeParameterErasure(legacyOwner, &symbol.Definition{OriginalType: "T"}); ok {
		t.Fatalf("identity-free legacy T was treated as proven owner T with erasure %q", got)
	}
}

func TestDirectOwnerTypeParameterDeclarationsUseErasedBound(t *testing.T) {
	out := normalizeSpaces(renderGoFileFromJava(t, `
interface ErasedDeclarationRoot {
    int number();
}

class ErasedDeclarationHolder<T extends ErasedDeclarationRoot> {
	T value;

	T read() {
		return value;
	}
}

abstract class ErasedDeclarationBridge<T extends ErasedDeclarationRoot> {
	abstract T abstractRead();
}

interface ErasedDeclarationSource<T extends ErasedDeclarationRoot> {
    T fetch();
}
`))

	for _, want := range []string{
		"value erasedDeclarationRoot",
		"read() erasedDeclarationRoot",
		"readJava2goExecution(__java2goExecution *stdjava.Execution) erasedDeclarationRoot",
		"abstractRead() T",
		"Fetch() T",
		"FetchJava2goExecution(__java2goExecution *stdjava.Execution) T",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("erased declaration ABI missing %q:\n%s", want, out)
		}
	}
	for _, stale := range []string{
		"value T",
		"read() T",
	} {
		if strings.Contains(out, stale) {
			t.Fatalf("declaration retained invariant source type %q:\n%s", stale, out)
		}
	}
}

func TestDirectOwnerTypeParameterFieldProjectionUsesAssignmentTarget(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class ErasedFieldTargetProjectionProgram {
    interface Bound {
        int number();
    }

	interface Extra {
		int extra();
	}

	interface Source<T> {
		T get();
	}

	static class First implements Bound, Extra {
		int firstField = 111;

		public int number() { return 1; }
        public int extra() { return 11; }
    }

    static class Second implements Bound, Extra {
        public int number() { return 2; }
        public int extra() { return 22; }
    }

	static class Box<T extends Bound> {
        T value;

		Box(T value) {
			this.value = value;
		}

		T read() {
			return value;
		}
	}

	static <X extends Bound & Extra> int readExtra(Box<X> box) {
		return box.value.extra();
	}

	static <X extends Bound & Extra> int readExtraResult(Box<X> box) {
		return box.read().extra();
	}

	static <X extends Bound> Source<X> bind(Box<X> box) {
		return box::read;
	}

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Box<First> typed = new Box<First>(new First());
        Box raw = typed;
		raw.value = new Second();

		Extra projected = typed.value;
		boolean projectedAlias = (Object) projected == (Object) raw.value;
		int genericExtra = readExtra(typed);
		int genericResultExtra = readExtraResult(typed);

		String exactRead;
		try {
			First exact = (typed.value);
			exactRead = "unexpected:" + exact.number();
		} catch (ClassCastException expected) {
			exactRead = "ClassCastException";
		}

		String exactFieldSelection;
		try {
			exactFieldSelection = "unexpected:" + typed.value.firstField;
		} catch (ClassCastException expected) {
			exactFieldSelection = "ClassCastException";
		}

		Source<First> source = typed::read;
		String methodReference;
		try {
			methodReference = "unexpected:" + source.get().number();
		} catch (ClassCastException expected) {
			methodReference = "ClassCastException";
		}

		Source<First> genericSource = bind(typed);
		String genericMethodReference;
		try {
			genericMethodReference = "unexpected:" + genericSource.get().number();
		} catch (ClassCastException expected) {
			genericMethodReference = "ClassCastException";
		}

		First replacement = new First();
		First assignmentResult = (typed.value = replacement);
		boolean assignmentAlias = (Object) assignmentResult == (Object) raw.value;
		return projected.extra() + ":" + genericExtra + ":" + genericResultExtra + ":" + projectedAlias + ":" + exactRead
				+ ":" + exactFieldSelection + ":" + methodReference + ":" + genericMethodReference
				+ ":" + assignmentAlias;
	}
}
`, "22:22:22:true:ClassCastException:ClassCastException:ClassCastException:ClassCastException:true")
}

func TestDirectOwnerTypeParameterResultDefersBridgeRequiredInterface(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class ErasedResultBridgeProgram {
    interface Numbered {
        int number();
    }

    interface Reader<T extends Numbered> {
        T read();
    }

    static class First implements Numbered {
        public int number() {
            return 7;
        }
    }

    static class FirstReader implements Reader<First> {
        public First read() {
            return new First();
        }
    }

    public static String run() {
        Reader<First> reader = new FirstReader();
        return "value=" + reader.read().number();
    }
}
`, "value=7")
}
