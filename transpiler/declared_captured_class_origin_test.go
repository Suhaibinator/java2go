package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/symbol"
	"os"
	"testing"
)

func TestDeclaredCapturedClassOriginOriginalNext(t *testing.T) {
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	data := []byte(declaredCapturedClassOriginalJava)
	if path := os.Getenv("RAW_GENERIC_JAVA_INPUT"); path != "" {
		actual, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actual, data) {
			t.Fatal("original Java fixture bytes changed")
		}
	}
	h := setupParseHelper(t, string(data))
	count := 0
	for _, owner := range allSourceClassScopes() {
		for _, field := range owner.Fields {
			if field == nil || field.Name != "next" {
				continue
			}
			count++
			origin := declaredJavaTypeOrigin(definitionDeclaredJavaType(field), classScopeCtx(owner, h.Ctx))
			t.Logf("next unresolved=%v nominal=%p expected=%p arguments=%+v", origin.unresolved, origin.nominalScope, owner, origin.arguments)
			if origin.unresolved || origin.nominalScope != owner || len(origin.arguments) != 1 || origin.arguments[0].parameter != owner.Enclosing.TypeParameters[0].Declaration {
				t.Fatal("lexical enclosing CLASS declaration origin must remain complete and exact")
			}
		}
	}
	if count != 1 {
		t.Fatal("originalnext field absent")
	}
}

const capturedClassOriginFixture = `
class Carrier<T> {
 class Mid<U> {
  class Leaf<V> {
   Leaf<V> next;
   Leaf<V>[] many;
   Carrier.Mid.Leaf raw;
   <T> void probe(T value) {}
   static void staticProbe() {}
  }
 }
}
class Value {}
`

func capturedClassOriginSetup(t *testing.T) (*symbol.ClassScope, Ctx) {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	h := setupParseHelper(t, capturedClassOriginFixture)
	leaf := resolveClassScopeByQualifiedName(h.Ctx, "Carrier.Mid.Leaf")
	if leaf == nil || leaf.Enclosing == nil || len(leaf.TypeParameters) != 3 {
		t.Fatal("actual three-level class declarations absent")
	}
	return leaf, classScopeCtx(leaf, h.Ctx)
}

func TestDeclaredCapturedClassOriginMultiLevelAndArray(t *testing.T) {
	leaf, ctx := capturedClassOriginSetup(t)
	for _, field := range leaf.Fields {
		if field == nil {
			continue
		}
		origin := declaredJavaTypeOrigin(definitionDeclaredJavaType(field), ctx)
		switch field.Name {
		case "next", "many":
			if !completeJavaTypeOrigin(origin) || origin.nominalScope != leaf || len(origin.arguments) != 3 {
				t.Fatalf("%s lost exact member origin: %+v", field.Name, origin)
			}
			for i, p := range leaf.TypeParameters {
				if origin.arguments[i].parameter != p.Declaration {
					t.Fatalf("%s slot%d recaptured a class binder", field.Name, i)
				}
			}
			if field.Name == "many" && origin.arrayRank != 1 || field.Name == "next" && origin.arrayRank != 0 {
				t.Fatal("member array rank changed")
			}
		case "raw":
			if origin.unresolved || origin.nominalScope != leaf || len(origin.arguments) != 0 {
				t.Fatal("explicit raw outer acquired lexical slots")
			}
		}
	}
	for _, method := range leaf.Methods {
		if method.OriginalName == "probe" {
			if len(method.TypeParameters) != 1 || method.TypeParameters[0].Declaration == leaf.TypeParameters[0].Declaration {
				t.Fatal("class/method homonym fixture invalid")
			}
			ctx.localScope = method
			got := declarationCapturedClassTypeOrigins(leaf, leaf, classParameterTypeOrigins(leaf), ctx)
			if got[leaf.TypeParameters[0].Declaration].parameter != leaf.TypeParameters[0].Declaration {
				t.Fatal("method shadow captured original class slot")
			}
		}
	}
}

func TestDeclaredCapturedClassOriginConcreteAndUnknown(t *testing.T) {
	leaf, ctx := capturedClassOriginSetup(t)
	arguments := classParameterTypeOrigins(leaf)
	original := append([]inferredJavaTypeOrigin(nil), arguments...)
	declaration := leaf.TypeParameters[0].Declaration
	concrete := declaredJavaTypeOrigin(symbol.JavaType{Original: "java.lang.String"}, ctx)
	if !completeJavaTypeOrigin(concrete) {
		t.Fatal("canonical String proof absent")
	}
	arguments[0] = concrete
	got := declarationCapturedClassTypeOrigins(leaf, leaf, arguments, ctx)
	if !sameJavaTypeOrigin(got[declaration], concrete) {
		t.Fatal("concrete carried origin was replaced by binder spelling")
	}
	value := resolveClassScopeByQualifiedName(ctx, "Value")
	arguments[0] = inferredJavaTypeOrigin{nominalScope: value, javaType: "Value"}
	if value == nil || declarationCapturedClassTypeOrigins(leaf, leaf, arguments, ctx)[declaration].nominalScope != value {
		t.Fatal("source concrete origin lost")
	}
	for label, bad := range map[string]inferredJavaTypeOrigin{
		"unknown":                  {javaType: "Missing", unresolved: true},
		"same-name-foreign-binder": {javaType: "T", parameter: &symbol.TypeParamDeclaration{SourceName: "T", GoName: "T"}},
		"wildcard":                 {javaType: "java.lang.Object", externalName: "java.lang.Object", wildcard: true},
	} {
		t.Run(label, func(t *testing.T) {
			arguments[0] = bad
			if _, ok := declarationCapturedClassTypeOrigins(leaf, leaf, arguments, ctx)[declaration]; ok {
				t.Fatal("unproved class argument gained origin")
			}
		})
	}
	if original[0].parameter != declaration || leaf.TypeParameters[0].Declaration != declaration {
		t.Fatal("helper mutated declaration origins")
	}
}

func TestDeclaredCapturedClassOriginContextShields(t *testing.T) {
	leaf, ctx := capturedClassOriginSetup(t)
	arguments := classParameterTypeOrigins(leaf)
	reject := func(label string, target, receiver *symbol.ClassScope, args []inferredJavaTypeOrigin, c Ctx) {
		t.Helper()
		if len(declarationCapturedClassTypeOrigins(target, receiver, args, c)) != 0 {
			t.Fatalf("%s accepted unsupported lexical provenance", label)
		}
	}
	reject("nil", nil, leaf, arguments, ctx)
	reject("missing-arguments", leaf, leaf, nil, ctx)
	foreign := ctx.Clone()
	foreign.currentFile = &symbol.FileScope{}
	reject("foreign-file", leaf, leaf, arguments, foreign)
	detached := *leaf
	foreign = ctx.Clone()
	foreign.currentClass = &detached
	reject("detached-receiver", leaf, &detached, arguments, foreign)
	reject("detached-target", &detached, leaf, arguments, ctx)
	foreign = ctx.Clone()
	foreign.currentClass = leaf.Enclosing
	reject("wrong-current-class", leaf, leaf, arguments, foreign)
	foreign = ctx.Clone()
	foreign.localScope = &symbol.Definition{IsStatic: true}
	reject("static-executable", leaf, leaf, arguments, foreign)
	saved := leaf.IsInner
	leaf.IsInner = false
	reject("static-member", leaf, leaf, arguments, ctx)
	leaf.IsInner = saved
	savedParent := leaf.Enclosing
	leaf.Enclosing = leaf
	reject("cyclic-enclosing", leaf, leaf, arguments, ctx)
	leaf.Enclosing = savedParent
	duplicate := append([]symbol.TypeParam(nil), leaf.TypeParameters...)
	savedParameters := leaf.TypeParameters
	leaf.TypeParameters = append(duplicate, duplicate[0])
	duplicatedArgs := append(append([]inferredJavaTypeOrigin(nil), arguments...), arguments[0])
	reject("duplicated-declaration-slot", leaf, leaf, duplicatedArgs, ctx)
	leaf.TypeParameters = savedParameters
}

const declaredCapturedClassOriginalJava = `
public class RawGenericProgram<T> {
    private Node head;

    public class Node {
        T value;
        Node next;
        Node(T value) { this.value = value; }
    }

    RawGenericProgram(T value) {
        this.head = new Node(value);
    }

    T get() {
        if (this.head != null) {
            return this.head.value;
        }
        return null;
    }

    String render() {
        String result = "value=";
        result += this.get();
        return result;
    }

    static void assertValue(RawGenericProgram list, int expected) {
        if ((Integer) list.get() != expected) {
            throw new AssertionError("wrong value");
        }
    }

    public static String run() {
        RawGenericProgram<Integer> list = new RawGenericProgram<>(7);
        assertValue(list, 7);
        return list.render();
    }
}
`
