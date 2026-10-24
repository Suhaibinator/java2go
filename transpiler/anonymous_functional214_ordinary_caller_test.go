package transpiler

import (
	"go/ast"
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
)

func TestCI214AnonymousFunctionalOrdinaryCallerOwnership(t *testing.T) {
	sources := []struct{ name, source string }{
		{"Function binder", `class Main{static<Function>Object make(){return new Function(){public Object apply(Object value){return value;}};}}`},
		{"java binder", `class Main{static<java>Object make(){return new java.util.function.Supplier<String>(){public String get(){return "value";}};}}`},
	}
	modes := []string{"foreign allocation", "foreign type", "foreign body role", "copied source", "stale source", "wrong file", "wrong class", "wrong method", "foreign hoist", "foreign counter", "foreign physical ABI", "mutated binder", "copied info", "copied scope", "copied class", "retired role"}
	for _, row := range sources {
		for _, mode := range modes {
			t.Run(row.name+"/"+mode, func(t *testing.T) {
				ctx, _, _, info := anonymousFunctional209Lower(t, row.source, "")
				role := info.lexicalOwner
				if role == nil || info.functional != nil {
					t.Fatal("actual ordinary role premise failed")
				}
				node, typ, body := role.allocation, role.typ, role.body
				bytes := ctx.currentFile.Source
				key, _ := anonymousClassSourceKey(node)
				foreign := parsing.SourceFile{Name: "Other.java", Source: []byte(row.source)}
				if err := foreign.ParseAST(); err != nil {
					t.Fatal(err)
				}
				allocation := findNode(foreign.Ast, "object_creation_expression")
				if allocation == nil || allocation.Equal(node) || allocation.StartByte() != node.StartByte() || allocation.EndByte() != node.EndByte() {
					t.Fatal("actual independent equal-span AST missing")
				}
				switch mode {
				case "foreign allocation":
					node = allocation
				case "foreign type":
					typ = allocation.ChildByFieldName("type")
				case "foreign body role":
					role.body = objectCreationClassBody(allocation)
				case "copied source":
					bytes = append([]byte(nil), bytes...)
				case "stale source":
					bytes = append([]byte(nil), bytes...)
					bytes[0] = ' '
				case "wrong file":
					ctx.currentFile = foreign.ParseSymbols()
				case "wrong class":
					ctx.currentClass = foreign.ParseSymbols().FindClassScope("Main")
				case "wrong method":
					ctx.localScope = foreign.ParseSymbols().FindClassScope("Main").FindMethod().ByOriginalName("make")[0]
				case "foreign hoist":
					copied := append([]ast.Decl(nil), (*ctx.hoistedDecls)...)
					ctx.hoistedDecls = &copied
				case "foreign counter":
					copied := *ctx.anonClassCounter
					ctx.anonClassCounter = &copied
				case "foreign physical ABI":
					ctx.rawGenericParameterTypes = map[string]string{"Donated": "Object"}
				case "mutated binder":
					ctx.localScope.TypeParameters[0].Declaration.GoName += "Donated"
				case "copied info":
					copied := *info
					info = &copied
					ctx.anonymousClasses[key] = info
				case "copied scope":
					copiedScope, copiedInfo := *info.scope, *info
					copiedInfo.scope = &copiedScope
					info = &copiedInfo
					ctx.anonymousClasses[key] = info
				case "copied class":
					copied := *info.scope.Class
					info.scope.Class = &copied
				case "retired role":
					role.phase = anonymousFunctionalFailed
				}
				before := anonymousFunctionalAST(&ast.File{Name: ast.NewIdent("snapshot"), Decls: *ctx.hoistedDecls})
				counter := *ctx.anonClassCounter
				entries := map[anonymousClassKey]*anonymousClassInfo{}
				for k, v := range ctx.anonymousClasses {
					entries[k] = v
				}
				if anonymousClassConstructionExpr(node, typ, info, nil, bytes, ctx) != nil {
					t.Fatal("ordinary direct caller donated allocation type source context or session")
				}
				if lowerAnonymousClassToStruct(node, typ, body, bytes, ctx) != nil {
					t.Fatal("ordinary cached replay accepted foreign caller or retired owner")
				}
				if anonymousFunctionalAST(&ast.File{Name: ast.NewIdent("snapshot"), Decls: *ctx.hoistedDecls}) != before || *ctx.anonClassCounter != counter || !reflect.DeepEqual(ctx.anonymousClasses, entries) {
					t.Fatal("ordinary ownership refusal mutated hoists counter or map")
				}
			})
		}
	}
}
