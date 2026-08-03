package transpiler

import (
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestGenericCast_TypedConstructionRoundTripsThroughObject(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class GenericTypedCastProgram {
    interface Numbered {
        int number();
    }

    static class First implements Numbered {
        final int value;

        First(int value) {
            this.value = value;
        }

        public int number() {
            return value;
        }
    }

    static class Box<T extends Numbered> {
        final T value;

        Box(T value) {
            this.value = value;
        }
    }

    @SuppressWarnings("unchecked")
    public static String run() {
        Box<First> original = new Box<>(new First(7));
        Object erased = original;
        Box<First> restored = (Box<First>) erased;
        return "value=" + restored.value.number();
    }
}
`, "value=7")
}

func TestPlanGenericClassRepresentation_UsesCanonicalDeclarationErasures(t *testing.T) {
	tests := []struct {
		name        string
		parameters  func() []symbol.TypeParam
		source      []string
		wantSource  []string
		wantRuntime []string
	}{
		{
			name: "unbounded",
			parameters: func() []symbol.TypeParam {
				return []symbol.TypeParam{symbol.NewTypeParam("T", nil)}
			},
			source:      []string{"String"},
			wantSource:  []string{"String"},
			wantRuntime: []string{"Object"},
		},
		{
			name: "interface first bound",
			parameters: func() []symbol.TypeParam {
				return []symbol.TypeParam{symbol.NewTypeParam("T", []symbol.JavaType{{Original: "Numbered"}})}
			},
			source:      []string{"First"},
			wantSource:  []string{"First"},
			wantRuntime: []string{"Numbered"},
		},
		{
			name: "concrete class first bound",
			parameters: func() []symbol.TypeParam {
				return []symbol.TypeParam{symbol.NewTypeParam("T", []symbol.JavaType{{Original: "Base"}})}
			},
			source:      []string{"Child"},
			wantSource:  []string{"Child"},
			wantRuntime: []string{"Base"},
		},
		{
			name: "dependent bound follows transitively",
			parameters: func() []symbol.TypeParam {
				parameters := []symbol.TypeParam{
					symbol.NewTypeParam("B", []symbol.JavaType{{Original: "Root"}}),
					symbol.NewTypeParam("T", []symbol.JavaType{{Original: "B"}}),
				}
				symbol.BindTypeParameterBounds(parameters, parameters)
				return parameters
			},
			source:      []string{"RootImpl", "Leaf"},
			wantSource:  []string{"RootImpl", "Leaf"},
			wantRuntime: []string{"Root", "Root"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parameters := test.parameters()
			scope := &symbol.ClassScope{
				TypeParameters:         parameters,
				DeclaredTypeParameters: parameters,
			}
			got := planGenericClassRepresentation(scope, test.source, nil, nil)
			if !reflect.DeepEqual(got.sourceArguments, test.wantSource) {
				t.Fatalf("source arguments = %#v, want %#v", got.sourceArguments, test.wantSource)
			}
			if !reflect.DeepEqual(got.runtimeArguments, test.wantRuntime) {
				t.Fatalf("runtime arguments = %#v, want %#v", got.runtimeArguments, test.wantRuntime)
			}
		})
	}
}

func TestPlanGenericClassRepresentation_RawAndParameterizedShareRuntimeType(t *testing.T) {
	parameter := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "Numbered"}})
	scope := &symbol.ClassScope{
		TypeParameters:         []symbol.TypeParam{parameter},
		DeclaredTypeParameters: []symbol.TypeParam{parameter},
	}

	raw := planGenericClassRepresentation(scope, nil, nil, nil)
	parameterized := planGenericClassRepresentation(scope, []string{"First"}, nil, nil)

	if want := []string{"Numbered"}; !reflect.DeepEqual(raw.runtimeArguments, want) {
		t.Fatalf("raw runtime arguments = %#v, want %#v", raw.runtimeArguments, want)
	}
	if !reflect.DeepEqual(parameterized.runtimeArguments, raw.runtimeArguments) {
		t.Fatalf(
			"parameterized runtime arguments = %#v, want canonical raw representation %#v",
			parameterized.runtimeArguments,
			raw.runtimeArguments,
		)
	}
	if want := []string{"First"}; !reflect.DeepEqual(parameterized.sourceArguments, want) {
		t.Fatalf("parameterized source arguments = %#v, want %#v", parameterized.sourceArguments, want)
	}
	if want := []string{"Numbered"}; !reflect.DeepEqual(raw.sourceArguments, want) {
		t.Fatalf("raw source arguments = %#v, want normalized erasure %#v", raw.sourceArguments, want)
	}
}

func TestPlanGenericClassRepresentation_DeclarationShadowingIsIdentitySafe(t *testing.T) {
	outerB := symbol.NewTypeParam("B", []symbol.JavaType{{Original: "OuterRoot"}})
	innerB := symbol.NewTypeParam("B", []symbol.JavaType{{Original: "InnerRoot"}})
	innerT := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "B"}})
	parameters := []symbol.TypeParam{outerB, innerB, innerT}
	symbol.DisambiguateTypeParamGoNames(parameters)
	symbol.BindTypeParameterBounds(parameters[2:], parameters)

	scope := &symbol.ClassScope{
		TypeParameters:         parameters,
		DeclaredTypeParameters: []symbol.TypeParam{innerB, innerT},
	}
	got := planGenericClassRepresentation(
		scope,
		[]string{"InnerImpl", "Leaf"},
		&symbol.ClassScope{TypeParameters: []symbol.TypeParam{outerB}},
		[]string{"OuterImpl"},
	)

	if want := []string{"OuterImpl", "InnerImpl", "Leaf"}; !reflect.DeepEqual(got.sourceArguments, want) {
		t.Fatalf("source arguments = %#v, want %#v", got.sourceArguments, want)
	}
	if want := []string{"OuterRoot", "InnerRoot", "InnerRoot"}; !reflect.DeepEqual(got.runtimeArguments, want) {
		t.Fatalf("runtime arguments = %#v, want declaration-resolved %#v", got.runtimeArguments, want)
	}
}

func TestPlanGenericClassRepresentation_MemberClassSeparatesCarriedSourceAndRuntimeViews(t *testing.T) {
	outerT := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "OuterBase"}})
	innerU := symbol.NewTypeParam("U", []symbol.JavaType{{Original: "InnerBase"}})
	outer := &symbol.ClassScope{
		TypeParameters:         []symbol.TypeParam{outerT},
		DeclaredTypeParameters: []symbol.TypeParam{outerT},
	}
	inner := &symbol.ClassScope{
		TypeParameters:         []symbol.TypeParam{outerT, innerU},
		DeclaredTypeParameters: []symbol.TypeParam{innerU},
		Enclosing:              outer,
		IsInner:                true,
	}

	got := planGenericClassRepresentation(inner, []string{"InnerImpl"}, outer, []string{"OuterImpl"})
	if want := []string{"OuterImpl", "InnerImpl"}; !reflect.DeepEqual(got.sourceArguments, want) {
		t.Fatalf("member source arguments = %#v, want %#v", got.sourceArguments, want)
	}
	if want := []string{"OuterBase", "InnerBase"}; !reflect.DeepEqual(got.runtimeArguments, want) {
		t.Fatalf("member runtime arguments = %#v, want %#v", got.runtimeArguments, want)
	}

	fullyQualified := planGenericClassRepresentation(
		inner,
		[]string{"QualifiedOuter", "QualifiedInner"},
		outer,
		[]string{"IgnoredReceiver"},
	)
	if want := []string{"QualifiedOuter", "QualifiedInner"}; !reflect.DeepEqual(fullyQualified.sourceArguments, want) {
		t.Fatalf("qualified member source arguments = %#v, want %#v", fullyQualified.sourceArguments, want)
	}
	if !reflect.DeepEqual(fullyQualified.runtimeArguments, got.runtimeArguments) {
		t.Fatalf("qualified member runtime arguments = %#v, want canonical %#v", fullyQualified.runtimeArguments, got.runtimeArguments)
	}
}

func TestPlanGenericClassRepresentation_DocumentsIntersectionAndFBoundBoundary(t *testing.T) {
	intersection := symbol.NewTypeParam("T", []symbol.JavaType{
		{Original: "Base"},
		{Original: "Ranked"},
	})
	fBound := symbol.NewTypeParam("U", []symbol.JavaType{{Original: "Comparable<U>"}})
	cycleA := symbol.NewTypeParam("A", []symbol.JavaType{{Original: "B"}})
	cycleB := symbol.NewTypeParam("B", []symbol.JavaType{{Original: "A"}})
	parameters := []symbol.TypeParam{intersection, fBound, cycleA, cycleB}
	symbol.BindTypeParameterBounds(parameters, parameters)

	got := planGenericClassRepresentation(&symbol.ClassScope{
		TypeParameters:         parameters,
		DeclaredTypeParameters: parameters,
	}, []string{"Impl", "Ordered", "Left", "Right"}, nil, nil)

	// Only the first intersection bound participates in Java erasure. Comparable
	// deliberately remains a Java spelling for the downstream converter to raw-
	// instantiate; a bare parameter cycle has no sound class root and is Object.
	want := []string{"Base", "Comparable", "Object", "Object"}
	if !reflect.DeepEqual(got.runtimeArguments, want) {
		t.Fatalf("boundary runtime arguments = %#v, want %#v", got.runtimeArguments, want)
	}
}
