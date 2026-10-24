package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"os"
	"strings"
	"testing"
)

func TestCanonicalGenericStorageOriginalPhysical(t *testing.T) {
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	data := []byte(canonicalGenericStorageOriginalJava)
	if path := os.Getenv("RAW_GENERIC_JAVA_INPUT"); path != "" {
		actual, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actual, data) {
			t.Fatal("original Java fixture bytes changed")
		}
	}
	out := renderGoFileFromJava(t, string(data))
	if path := os.Getenv("RAW_GENERIC_GENERATED_OUTPUT"); path != "" {
		if err := os.WriteFile(path, []byte(out), 0644); err != nil {
			t.Fatal(err)
		}
	}
	runGoTestInTempModule(t, out, `
package main
import("testing";"slices";"github.com/NickyBoy89/java2go/stdjava")
func TestOriginalJDKValue(t *testing.T){var got *stdjava.JavaString=Run();if got==nil||!slices.Equal(got.UTF16Copy(),[]uint16{'v','a','l','u','e','=','7'}){t.Fatal("original JDK value differs")}}
`)
}

const canonicalGenericStorageOriginalJava = `
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

func canonicalStorageOriginalContext(t *testing.T) (Ctx, *symbol.ClassScope, *symbol.ClassScope) {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	h := setupParseHelper(t, canonicalGenericStorageOriginalJava+`class Plain<P>{P value;Plain<P> next;}`)
	owner := h.File.Symbols.FindClassScope("RawGenericProgram")
	var member *symbol.ClassScope
	for _, candidate := range allSourceClassScopes() {
		if candidate.Enclosing == owner {
			member = candidate
		}
	}
	if owner == nil || member == nil || !canonicalGenericClass(owner, h.Ctx) || !canonicalGenericClass(member, h.Ctx) {
		t.Fatal("actual original canonical declarations absent")
	}
	return h.Ctx, owner, member
}

func canonicalStorageOwnedField(owner *symbol.ClassScope, name string, ctx Ctx) *ast.Field {
	for _, definition := range owner.Fields {
		if definition != nil && definition.OriginalName == name {
			return &ast.Field{Names: []*ast.Ident{ast.NewIdent(definition.Name)}, Type: javaTypeStringToGoTypeExpr(definition.OriginalType, inScopeTypeParameters(ctx), ctx)}
		}
	}
	return nil
}

func TestCanonicalGenericStorageDeclarationAndEnclosingPointers(t *testing.T) {
	ctx, owner, member := canonicalStorageOriginalContext(t)
	for _, test := range []struct {
		owner  *symbol.ClassScope
		name   string
		target *symbol.ClassScope
	}{{owner, "head", member}, {member, "next", member}} {
		current := classScopeCtx(test.owner, ctx)
		field := canonicalStorageOwnedField(test.owner, test.name, current)
		if field == nil {
			t.Fatal("actual original source field absent")
		}
		physical, ok := canonicalGenericStorageFieldType(field, current)
		want := "*" + availableCanonicalGenericName(test.target.Class.Name+"Java2goErased")
		if !ok || boundedArgumentTestGoType(t, physical) != want {
			t.Fatalf("%s physical storage does not name proved canonical declaration", test.name)
		}
		var definition *symbol.Definition
		for _, d := range test.owner.Fields {
			if d.Name == field.Names[0].Name {
				definition = d
			}
		}
		before := declaredJavaTypeOrigin(definitionDeclaredJavaType(definition), current)
		canonicalGenericStorageFields(&ast.FieldList{List: []*ast.Field{field}}, current)
		if !sameJavaTypeOrigin(before, declaredJavaTypeOrigin(definitionDeclaredJavaType(definition), current)) {
			t.Fatal("physical storage rewrite changed Java declaration origin")
		}
	}
	current := classScopeCtx(member, ctx)
	outer := buildEnclosingInstanceField(current)
	physical, ok := canonicalGenericStorageFieldType(outer, current)
	if !ok || boundedArgumentTestGoType(t, physical) != "*"+availableCanonicalGenericName(owner.Class.Name+"Java2goErased") {
		t.Fatal("actual enclosing pointer did not use existing canonical carrier")
	}
	value := canonicalStorageOwnedField(member, "value", current)
	if _, ok := canonicalGenericStorageFieldType(value, current); ok {
		t.Fatal("class binder storage was inferred as a source nominal class")
	}
}

func TestCanonicalGenericStorageAmbiguousIncompleteAndPhysicalShields(t *testing.T) {
	ctx, owner, _ := canonicalStorageOriginalContext(t)
	ctx = classScopeCtx(owner, ctx)
	field := canonicalStorageOwnedField(owner, "head", ctx)
	var definition *symbol.Definition
	for _, d := range owner.Fields {
		if d.OriginalName == "head" {
			definition = d
		}
	}
	if definition == nil {
		t.Fatal("original field definition absent")
	}
	reject := func(label string, f *ast.Field, c Ctx) {
		t.Helper()
		if _, ok := canonicalGenericStorageFieldType(f, c); ok {
			t.Fatalf("%s acquired canonical nominal storage without proof", label)
		}
	}
	reject("nil", nil, ctx)
	reject("anonymous Go embed", &ast.Field{Type: field.Type}, ctx)
	reject("multiple declaration names", &ast.Field{Names: []*ast.Ident{field.Names[0], ast.NewIdent("another")}, Type: field.Type}, ctx)
	oldFields := owner.Fields
	duplicate := *definition
	owner.Fields = append(append([]*symbol.Definition(nil), oldFields...), &duplicate)
	reject("ambiguous association", field, ctx)
	owner.Fields = oldFields
	original := *definition
	for label, typ := range map[string]string{"unknown": "Missing", "array": "Node[]", "external": "java.nio.ByteBuffer", "classbinder": "T"} {
		definition.OriginalType = typ
		reject(label, field, ctx)
		*definition = original
	}
	definition.IsStatic = true
	reject("static definition", field, ctx)
	*definition = original
	definition.DeclarationNode = nil
	reject("detached definition", field, ctx)
	*definition = original
	reject("nonpointer representation", &ast.Field{Names: field.Names, Type: ast.NewIdent("any")}, ctx)
	foreign := ctx.Clone()
	foreign.currentFile = &symbol.FileScope{}
	reject("foreign sourcefile", field, foreign)
	detached := *owner
	foreign = ctx.Clone()
	foreign.currentClass = &detached
	reject("detached class scope", field, foreign)
	plain := resolveClassScopeByQualifiedName(ctx, "Plain")
	plainCtx := classScopeCtx(plain, ctx)
	if plain == nil || canonicalGenericClass(plain, plainCtx) {
		t.Fatal("ordinary invariant generic fixture was unexpectedly canonicalized")
	}
	ordinary := canonicalStorageOwnedField(plain, "next", plainCtx)
	text := boundedArgumentTestGoType(t, ordinary.Type)
	canonicalGenericStorageFields(&ast.FieldList{List: []*ast.Field{ordinary}}, plainCtx)
	if boundedArgumentTestGoType(t, ordinary.Type) != text {
		t.Fatal("noncanonical generic representation changed")
	}
}

func TestCanonicalGenericStorageSameNameSeparatePackages(t *testing.T) {
	source := func(pkg string) string {
		return "package " + pkg + `;public class Outer<T>{Node head;public class Node{Node next;}public static void use(Outer value){}}`
	}
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{"p/Outer.java": source("p"), "q/Outer.java": source("q")})
	first, second := findQualifiedSourceClass("p.Outer"), findQualifiedSourceClass("q.Outer")
	if first == nil || second == nil || first == second {
		t.Fatal("two actual same-name source declarations absent")
	}
	for _, owner := range []*symbol.ClassScope{first, second} {
		declaring := classScopeCtx(owner, ctx)
		field := canonicalStorageOwnedField(owner, "head", declaring)
		var definition *symbol.Definition
		for _, d := range owner.Fields {
			if d.OriginalName == "head" {
				definition = d
			}
		}
		origin := declaredJavaTypeOrigin(definitionDeclaredJavaType(definition), declaring)
		if origin.nominalScope == nil || origin.nominalScope.Enclosing != owner {
			t.Fatal("source homonym redirected declaration origin")
		}
		physical, ok := canonicalGenericStorageFieldType(field, declaring)
		if !ok || !strings.Contains(boundedArgumentTestGoType(t, physical), "OuterNodeJava2goErased") {
			t.Fatal("separate owned canonical class storage absent")
		}
		other := first
		if owner == first {
			other = second
		}
		foreign := declaring.Clone()
		foreign.currentFile = findFileScopeForClassScope(other, ctx)
		if _, ok := canonicalGenericStorageFieldType(field, foreign); ok {
			t.Fatal("consumer file recaptured same-name declaration")
		}
	}
}
