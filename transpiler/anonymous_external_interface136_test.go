package transpiler

import (
	"encoding/json"
	"go/ast"
	"os"
	"testing"
)

func TestAnonymousExternalInterface136LoweringNominal(t *testing.T) {
	for _, text := range []string{
		`class Main{static Object make(){return new Comparable<Object>(){public int compareTo(Object x){return 0;}};}}`,
		`import java.util.*;class Main{static Object make(){return new Comparator<Object>(){public int compare(Object x,Object y){return 0;}};}}`,
		`class Main{static<T>Object make(T token){return new java.lang.Comparable<T>(){public int compareTo(T value){return value==token?1:0;}};}}`,
	} {
		h := setupParseHelper(t, text)
		ctx := h.Ctx.Clone()
		ctx.className = ctx.currentClass.Class.Name
		ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("make")[0]
		originalClass, originalMethod := ctx.currentClass, ctx.localScope
		var decls []ast.Decl
		var counter int
		ctx.hoistedDecls = &decls
		ctx.anonClassCounter = &counter
		ctx.anonymousClasses = make(map[anonymousClassKey]*anonymousClassInfo)
		ctx.localClasses = make(map[string]*localClassInfo)
		ctx.executionContextName = "execution"
		creation := findNode(h.File.Ast, "object_creation_expression")
		expression := lowerAnonymousClassToStruct(creation, creation.ChildByFieldName("type"), objectCreationClassBody(creation), h.File.Source, ctx)
		if expression == nil {
			t.Fatal("anonymous allocation missing")
		}
		key, _ := anonymousClassSourceKey(creation)
		info := ctx.anonymousClasses[key]
		if info == nil || info.scope == nil {
			t.Fatal("actual anonymous source scope missing")
		}
		if len(info.scope.ImplementedInterfaces) != 1 || info.scope.Superclass != "" {
			t.Errorf("allocation lost actual interface declaration: interfaces=%v superclass=%q", info.scope.ImplementedInterfaces, info.scope.Superclass)
		}
		registrations := 0
		dynamic := 0
		var nativeID string
		if text[0:6] == "import" {
			nativeID = "java.util.Comparator"
		} else {
			nativeID = "java.lang.Comparable"
		}
		for _, declaration := range decls {
			localRegistrations := 0
			ast.Inspect(declaration, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok && volatileTDDCallName(call) == "RegisterJavaType" {
					registrations++
					localRegistrations++
					if len(call.Args) < 2 {
						t.Fatal("missing Object parent")
					} else if parent, ok := call.Args[1].(*ast.SelectorExpr); !ok || parent.Sel.Name != "ObjectTypeID" {
						t.Errorf("wrong parent %T", call.Args[1])
					}
				}
				if typ, ok := n.(*ast.TypeSpec); ok && typ.Name.Name == info.structName {
					if structure, ok := typ.Type.(*ast.StructType); ok {
						for _, field := range structure.Fields.List {
							if len(field.Names) == 0 {
								t.Errorf("external interface incorrectly has concrete embedding %T", field.Type)
							}
						}
					}
				}
				return true
			})
			if fn, ok := declaration.(*ast.FuncDecl); ok && fn.Name.Name == "JavaDynamicTypeID" {
				dynamic++
			}
			if localRegistrations > 0 {
				ids := interface136RegistrationIDs(t, declaration)
				if len(ids) > 0 && ids[nativeID] != 1 {
					t.Errorf("registration must have one actual interface and Object parent, IDs=%v", ids)
				}
			}
		}
		if registrations != 1 || dynamic != 1 {
			t.Errorf("one nominal identity/registration required: registration=%d dynamic=%d", registrations, dynamic)
		}
		if ctx.currentClass != originalClass || ctx.localScope != originalMethod {
			t.Fatal("allocation context owner or executable changed")
		}
	}
}

func TestAnonymousExternalInterface136AllocationContextGuards(t *testing.T) {
	rows := []struct{ name, text, want string }{
		{"implicit", `class Main{static Object make(){return new Comparable<Object>(){public int compareTo(Object v){return 0;}};}}`, "java.lang.Comparable"},
		{"actual method binder", `class Main{static<Comparable>Object make(){return new Comparable(){ };}}`, ""},
		{"qualified binder prefix", `class Main{static<java>Object make(){return new java.lang.Comparable<Object>(){public int compareTo(Object v){return 0;}};}}`, ""},
		{"class vs interface", `class Main{static class Comparable{Comparable(){}}static Object make(){return new Comparable(){};}}`, ""},
		{"source interface", `class Main{interface Comparable{}static Object make(){return new Comparable(){};}}`, ""},
		{"unknown nested external", `class Main{static Object make(){return new foreign.Outer.Comparable(){};}}`, ""},
		{"unknown demand", `import unknown.*;class Main{static Object make(){return new java.lang.Comparable<Object>(){public int compareTo(Object v){return 0;}};}}`, ""},
		{"explicit import owns", `import unknown.*;import java.lang.Comparable;class Main{static Object make(){return new Comparable<Object>(){public int compareTo(Object v){return 0;}};}}`, "java.lang.Comparable"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.text)
			ctx := h.Ctx.Clone()
			ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("make")[0]
			creation := findNode(h.File.Ast, "object_creation_expression")
			typ := creation.ChildByFieldName("type")
			current, method := ctx.currentClass, ctx.localScope
			if got := anonymousExternalInterfaceOwner(creation, typ, h.File.Source, ctx); got != row.want {
				t.Fatalf("got %q want %q", got, row.want)
			}
			if ctx.currentClass != current || ctx.localScope != method {
				t.Fatal("query changed actual declaration/executable")
			}
			missing := ctx.Clone()
			missing.currentFile = nil
			if got := anonymousExternalInterfaceOwner(creation, typ, h.File.Source, missing); got != "" {
				t.Fatal("fileless allocation admitted")
			}
			if got := anonymousExternalInterfaceOwner(creation, nil, h.File.Source, ctx); got != "" {
				t.Fatal("missing actual TYPE admitted")
			}
		})
	}
}

func TestAnonymousExternalInterface136FullPrograms(t *testing.T) {
	data, err := os.ReadFile("testdata/anonymous_external_interface136/full-controls.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Name  string            `json:"name"`
		Entry string            `json:"entry"`
		Want  string            `json:"want"`
		Files map[string]string `json:"files"`
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 8 {
		t.Fatal("all eight full controls required")
	}
	for _, row := range rows {
		t.Run(row.Name, func(t *testing.T) { runCampaignCompilerStrictProjectOracle(t, row.Files, row.Entry, row.Want) })
	}
}
