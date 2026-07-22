package symbol_test

import (
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func TestTypeParameterIdentityRetainsSameNamedDeclarations(t *testing.T) {
	classParameter := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "ClassBound"}})
	methodParameter := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "MethodBound"}})
	localParameter := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "LocalBound"}})

	parameters := symbol.AppendTypeParamsByDeclaration(
		[]symbol.TypeParam{classParameter},
		[]symbol.TypeParam{methodParameter},
		[]symbol.TypeParam{localParameter},
	)
	symbol.DisambiguateTypeParamGoNames(parameters)

	if got, want := symbol.TypeParamNames(parameters), []string{"T", "T", "T"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("source parameter names = %#v, want %#v", got, want)
	}
	if got, want := symbol.GoTypeParamNames(parameters), []string{"T", "T2", "T3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("generated parameter names = %#v, want %#v", got, want)
	}

	methodCopy := methodParameter
	if methodCopy.Declaration != methodParameter.Declaration {
		t.Fatal("ordinary TypeParam copies must retain declaration identity")
	}
	if got, want := methodCopy.EmittedName(), "T2"; got != want {
		t.Fatalf("copied method parameter generated name = %q, want %q", got, want)
	}
	if got := symbol.AppendTypeParamsByDeclaration(parameters, []symbol.TypeParam{methodCopy}); len(got) != 3 {
		t.Fatalf("declaration-identical copy was not deduplicated: %#v", got)
	}
}

func TestTypeParameterDisambiguationPreservesOrdinaryNames(t *testing.T) {
	parameters := []symbol.TypeParam{
		symbol.NewTypeParam("T", nil),
		symbol.NewTypeParam("U", nil),
	}
	symbol.DisambiguateTypeParamGoNames(parameters)

	if got, want := symbol.GoTypeParamNames(parameters), []string{"T", "U"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("non-shadowed generated parameter names = %#v, want %#v", got, want)
	}
}

func TestLegacyTypeParameterFallbackIsDeterministic(t *testing.T) {
	first := symbol.TypeParam{Name: "T", Bounds: []symbol.JavaType{{Original: "First"}}}
	firstCopy := first
	second := symbol.TypeParam{Name: "T", Bounds: []symbol.JavaType{{Original: "Second"}}}

	parameters := symbol.AppendTypeParamsByDeclaration(
		[]symbol.TypeParam{first, firstCopy},
		[]symbol.TypeParam{second},
	)
	if got, want := len(parameters), 2; got != want {
		t.Fatalf("retained legacy parameter count = %d, want %d", got, want)
	}
	symbol.DisambiguateTypeParamGoNames(parameters)

	if got, want := symbol.GoTypeParamNames(parameters), []string{"T", "T2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("legacy generated parameter names = %#v, want %#v", got, want)
	}
}

func TestTypeParameterDisambiguationFreezesOuterNamesAndReservesSourceSpellings(t *testing.T) {
	outer := symbol.NewTypeParam("T", nil)
	firstMethod := symbol.NewTypeParam("T1", nil)
	firstContext := symbol.AppendTypeParamsByDeclaration(
		[]symbol.TypeParam{outer},
		[]symbol.TypeParam{firstMethod},
	)
	symbol.DisambiguateTypeParamGoNames(firstContext)

	shadowingMethod := symbol.NewTypeParam("T", nil)
	secondContext := symbol.AppendTypeParamsByDeclaration(
		[]symbol.TypeParam{outer},
		[]symbol.TypeParam{firstMethod},
		[]symbol.TypeParam{shadowingMethod},
	)
	symbol.DisambiguateTypeParamGoNames(secondContext)

	if got, want := symbol.GoTypeParamNames(firstContext), []string{"T", "T1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("previously allocated context changed to %#v, want %#v", got, want)
	}
	if got, want := symbol.GoTypeParamNames(secondContext), []string{"T", "T1", "T2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("new shadowing context names = %#v, want %#v", got, want)
	}

	reservedSuffix := symbol.NewTypeParam("T2", nil)
	secondShadow := symbol.NewTypeParam("T", nil)
	reservedContext := []symbol.TypeParam{outer, reservedSuffix, secondShadow}
	symbol.DisambiguateTypeParamGoNames(reservedContext)
	if got, want := symbol.GoTypeParamNames(reservedContext), []string{"T", "T2", "T3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("reserved source spelling was reused: got %#v, want %#v", got, want)
	}
}

func TestParseSymbolsTracksShadowedDeclarationProvenance(t *testing.T) {
	src := `
class Outer<T> {
    T outerValue;

    class Inner<T> {
        List<T> innerValues;

        <T> List<T> identity(List<T> value) {
			List<T> copy = value;
			return copy;
        }
    }

    static class Nested<T> {
        T value;
    }
}
`
	file := parsing.SourceFile{Name: "Outer.java", Source: []byte(src)}
	if err := file.ParseAST(); err != nil {
		t.Fatalf("failed to parse AST: %v", err)
	}

	outer := file.ParseSymbols().FindClassScope("Outer")
	if outer == nil || len(outer.Subclasses) != 2 {
		t.Fatalf("expected Outer with Inner and Nested, got %#v", outer)
	}
	inner := outer.Subclasses[0]
	if got, want := symbol.TypeParamNames(inner.TypeParameters), []string{"T", "T"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Inner source parameter names = %#v, want %#v", got, want)
	}
	if got, want := inner.GoTypeParameterNames(), []string{"T", "T2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Inner generated parameter names = %#v, want %#v", got, want)
	}
	if got := inner.Fields[0].TypeParameterBindings["T"]; got != inner.OwnTypeParameters()[0].Declaration {
		t.Fatal("nested List<T> field did not bind to Inner's declaration")
	}

	method := inner.Methods[0]
	if got, want := method.GoTypeParameterNames(), []string{"T3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("method generated parameter names = %#v, want %#v", got, want)
	}
	if got := method.TypeParameterBindings["T"]; got != method.TypeParameters[0].Declaration {
		t.Fatal("nested List<T> result did not bind to the method declaration")
	}
	if got := method.Parameters[0].TypeParameterBindings["T"]; got != method.TypeParameters[0].Declaration {
		t.Fatal("nested List<T> parameter did not bind to the method declaration")
	}
	if len(method.Children) != 1 || method.Children[0].TypeParameterBindings["T"] != method.TypeParameters[0].Declaration {
		t.Fatal("nested List<T> local did not retain the method declaration for capture")
	}

	nested := outer.Subclasses[1]
	if got, want := nested.TypeParameterNames(), []string{"T"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("static Nested generated parameters = %#v, want %#v", got, want)
	}
	if nested.TypeParameters[0].Declaration == outer.TypeParameters[0].Declaration {
		t.Fatal("static nested class incorrectly carried Outer's declaration")
	}
}

func TestParseSymbolsTreatsInterfaceMemberClassAsImplicitlyStatic(t *testing.T) {
	src := `
interface Container<T> {
    class Member<U> {
        U value;
    }
}
`
	file := parsing.SourceFile{Name: "Container.java", Source: []byte(src)}
	if err := file.ParseAST(); err != nil {
		t.Fatalf("failed to parse AST: %v", err)
	}

	container := file.ParseSymbols().FindClassScope("Container")
	if container == nil || len(container.Subclasses) != 1 {
		t.Fatalf("expected Container.Member, got %#v", container)
	}
	member := container.Subclasses[0]
	if member.IsInner {
		t.Fatal("interface member class must be implicitly static")
	}
	if got, want := symbol.TypeParamNames(member.TypeParameters), []string{"U"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("interface member class carried outer parameters: got %#v, want %#v", got, want)
	}
	if member.TypeParameters[0].Declaration == container.TypeParameters[0].Declaration {
		t.Fatal("interface member class reused the interface type-parameter declaration")
	}
}

func TestBindTypeParameterBoundsRecordsDependentDeclaration(t *testing.T) {
	parameters := []symbol.TypeParam{
		symbol.NewTypeParam("B", []symbol.JavaType{{Original: "Root"}}),
		symbol.NewTypeParam("T", []symbol.JavaType{{Original: "B"}}),
	}
	symbol.BindTypeParameterBounds(parameters, parameters)

	if got := parameters[1].Bounds[0].TypeParameterBindings["B"]; got != parameters[0].Declaration {
		t.Fatal("T extends B did not retain B's declaration identity")
	}
}
