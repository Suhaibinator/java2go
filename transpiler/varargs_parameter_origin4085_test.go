package transpiler

import (
	"go/ast"
	"reflect"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestVarargsParameter4085OriginRankAndIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, src, owner string
		rank             int
	}{
		{"source", `class Part {} class Use {void probe(Part... value){}}`, "Use", 1},
		{"native reference", `class Use {void probe(java.lang.String... value){}}`, "Use", 1},
		{"source native homonym", `class String {} class Use {void probe(String... value){}}`, "Use", 1},
		{"primitive", `class Use {void probe(int... value){}}`, "Use", 1},
		{"ordinary array", `class Part {} class Use {void probe(Part[] value){}}`, "Use", 1},
		{"ordinary scalar", `class Part {} class Use {void probe(Part value){}}`, "Use", 0},
		{"array spread", `class Part {} class Use {void probe(Part[]... value){}}`, "Use", 2},
		{"multiarray spread", `class Part {} class Use {void probe(Part[][]... value){}}`, "Use", 3},
		{"primitive multiarray spread", `class Use {void probe(int[][]... value){}}`, "Use", 3},
		{"method binder", `class Use {<T> void probe(T... value){}}`, "Use", 1},
		{"method binder multiarray", `class Use {<T> void probe(T[][]... value){}}`, "Use", 3},
		{"class binder", `class Use<T> {void probe(T... value){}}`, "Use", 1},
		{"method shadows class binder", `class Use<T> {<T> void probe(T... value){}}`, "Use", 1},
		{"captured class binder", `class Outer<T> {class Use {void probe(T... value){}}}`, "Use", 1},
		{"nested argument binder", `class Box<T> {} class Use<T> {void probe(Box<T>... value){}}`, "Use", 1},
		{"unknown reference stays unknown", `class Use {void probe(Absent... value){}}`, "Use", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := setupParseHelper(t, tc.src)
			owner := h.File.Symbols.FindClassScope(tc.owner)
			if owner == nil {
				t.Fatal("missing parsed owner")
			}
			var method *symbol.Definition
			for _, m := range owner.Methods {
				if m.OriginalName == "probe" {
					method = m
				}
			}
			if method == nil || len(method.Parameters) != 1 {
				t.Fatal("missing parsed method")
			}
			ctx := h.Ctx.Clone()
			ctx.currentClass = owner
			ctx.localScope = method
			param := method.Parameters[0]
			before := *param
			var origin inferredJavaTypeOrigin
			typ, known := inferIdentifierJavaType("value", ctx, &origin)
			_, rank := javaArrayTypeParts(typ)
			if !known || rank != tc.rank || origin.arrayRank != tc.rank {
				t.Fatalf("display %q rank %d origin rank %d want %d", typ, rank, origin.arrayRank, tc.rank)
			}
			var expected inferredJavaTypeOrigin
			recordDefinitionTypeOrigin([]*inferredJavaTypeOrigin{&expected}, param, nil, ctx, nil, typ)
			if executionParameterIsVariadic(method, 0) {
				expected.arrayRank++
			}
			if !reflect.DeepEqual(origin, expected) {
				t.Fatalf("origin identity changed beyond one proved dimension: got %#v want %#v", origin, expected)
			}
			if !reflect.DeepEqual(before, *param) {
				t.Fatal("parameter definition mutated")
			}
			if plain, ok := inferIdentifierJavaType("value", ctx); !ok || plain != typ {
				t.Fatal("no-origin inference differs")
			}
			if plain, ok := inferIdentifierJavaType("value", ctx, nil); !ok || plain != typ {
				t.Fatal("nil output inference differs")
			}
			if tc.name == "unknown reference stays unknown" && !origin.unresolved {
				t.Fatal("unknown source gained identity")
			}
			if strings.Contains(tc.name, "binder") && tc.name != "nested argument binder" && origin.parameter == nil {
				t.Fatal("parsed binder identity lost")
			}
			if tc.name == "nested argument binder" && (len(origin.arguments) != 1 || origin.arguments[0].parameter == nil) {
				t.Fatal("nested binder identity lost")
			}
		})
	}
}

func TestVarargsParameter4085RefusalAndSyntheticControls(t *testing.T) {
	h := setupParseHelper(t, `class Use {void probe(int value){Object empty=null;}}`)
	owner := h.File.Symbols.FindClassScope("Use")
	method := owner.Methods[0]
	ctx := h.Ctx.Clone()
	ctx.currentClass = owner
	ctx.localScope = method
	method.IsVariadic = true
	var origin inferredJavaTypeOrigin
	typ, known := inferIdentifierJavaType("value", ctx, &origin)
	if !known || typ != "int[]" || origin.arrayRank != 1 {
		t.Fatal("existing synthetic variadic proof not preserved")
	}
	if executionParameterIsVariadic(method, -1) || executionParameterIsVariadic(method, 1) {
		t.Fatal("out of bounds variadic parameter admitted")
	}
	sentinel := inferredJavaTypeOrigin{javaType: "sentinel", arrayRank: 7, unresolved: true}
	unknown := sentinel
	if _, known := inferIdentifierJavaType("missing", ctx, &unknown); known || !reflect.DeepEqual(unknown, sentinel) {
		t.Fatal("unknown identifier gained type or origin")
	}
	method.IsVariadic = false
	ctx.rawGenericParameterTypes = map[string]string{"value": "int[][]"}
	origin = sentinel
	if got, ok := inferIdentifierJavaType("value", ctx, &origin); !ok || got != "int[][]" || !reflect.DeepEqual(origin, sentinel) {
		t.Fatal("existing representation-rewrite precedence changed")
	}
	ctx.rawGenericParameterTypes = nil
	origin = inferredJavaTypeOrigin{}
	if got, ok := inferExprJavaType(findNode(method.DeclarationNode, "null_literal"), ctx, h.File.Source, &origin); !ok || (got != "null" && got != ternaryNullJavaType) || origin.arrayRank != 0 {
		t.Fatal("null acquired array dimension")
	}
}

func TestVarargsParameter4085CapturedExecutableBinder(t *testing.T) {
	h := setupParseHelper(t, `class Outer<T> {<T> void factory(){class Local {void probe(T[][]... value){}}}}`)
	ctx := h.Ctx.Clone()
	ctx.currentClass = h.File.Symbols.FindClassScope("Outer")
	ctx.className = ctx.currentClass.Class.Name
	ctx.localScope = ctx.currentClass.FindMethodByName("factory", nil)
	binder := ctx.localScope.TypeParameters[0].Declaration
	var declarations []ast.Decl
	var counter int
	ctx.hoistedDecls = &declarations
	ctx.anonClassCounter = &counter
	ctx.localClasses = make(map[string]*localClassInfo)
	ctx.anonymousClasses = make(map[anonymousClassKey]*anonymousClassInfo)
	hoistLocalClass(findNode(ctx.localScope.DeclarationNode, "class_declaration"), h.File.Source, ctx)
	local := ctx.localClasses["Local"]
	if local == nil {
		t.Fatal("actual local producer absent")
	}
	ctx.currentClass = local.scope
	ctx.localScope = local.scope.FindMethodByName("probe", nil)
	if ctx.localScope == nil {
		t.Fatal("actual local method absent")
	}
	var origin inferredJavaTypeOrigin
	typ, known := inferIdentifierJavaType("value", ctx, &origin)
	if !known || typ == "" || origin.arrayRank != 3 || origin.parameter != binder {
		t.Fatalf("captured executable identity/rank lost: type=%s rank=%d binder=%p want=%p", typ, origin.arrayRank, origin.parameter, binder)
	}
	for _, p := range h.File.Symbols.FindClassScope("Outer").TypeParameters {
		if p.Declaration == origin.parameter {
			t.Fatal("class homonym replaced method binder")
		}
	}
}
