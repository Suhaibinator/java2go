package transpiler

import (
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
	"testing"
)

func anonymousFunctional214NestedSource(parent, child string) (string, string, string) {
	edge, method, call := "java.util.function.Function<String,String>", `public String apply(String value){return value;}`, `new Local().apply("body")`
	if child == "Supplier" {
		edge, method, call = "java.util.function.Supplier<String>", `public String get(){return "body";}`, `new Local().get()`
	}
	local := `class Local implements ` + edge + `{` + method + `}`
	header, parentMethod, parentName := "java.util.function.Function<String,String>", `public String apply(String value){`+local+`return `+call+`;}`, "apply"
	if parent == "Supplier" {
		header, parentMethod, parentName = "java.util.function.Supplier<String>", `public String get(){`+local+`return `+call+`;}`, "get"
	}
	if parent == "Runnable" {
		header, parentMethod, parentName = "java.lang.Runnable", `public void run(){`+local+call+`;}`, "run"
	}
	return `class Main{static ` + header + ` make(){return new ` + header + `(){` + parentMethod + `};}}`, header, parentName
}

// A fresh actual pipeline owns the direct-consumer fixture. A harmless ordinary
// anonymous allocation in the local method retains its real lexical block map;
// the helper never inserts Local or synthesizes a declaration/scope by hand.
func anonymousFunctional214NestedContext(t *testing.T, parent, child string) (Ctx, *anonymousClassInfo, *localClassInfo) {
	t.Helper()
	source, header, _ := anonymousFunctional214NestedSource(parent, child)
	needle := "return value;"
	if child == "Supplier" {
		needle = `return "body";`
	}
	if strings.Count(source, needle) != 1 {
		t.Fatal("actual child method insertion anchor missing")
	}
	source = strings.Replace(source, needle, "Object ownershipWitness=new Object(){};"+needle, 1)
	ctx, _, _, owner := anonymousFunctional209Lower(t, source, header)
	localNode := findNode(owner.scope.Class.DeclarationNode, "class_declaration")
	if localNode == nil {
		t.Fatal("actual nested local declaration missing")
	}
	witness := findNode(localNode, "object_creation_expression")
	witnessCount := 0
	pendingNodes := []*sitter.Node{localNode}
	for len(pendingNodes) > 0 {
		node := pendingNodes[len(pendingNodes)-1]
		pendingNodes = pendingNodes[:len(pendingNodes)-1]
		if node.Type() == "object_creation_expression" {
			witnessCount++
			if !anonymousFunctionalEqual(node, witness) {
				t.Fatal("actual retention allocation ambiguous")
			}
		}
		for i := 0; i < int(node.NamedChildCount()); i++ {
			pendingNodes = append(pendingNodes, node.NamedChild(i))
		}
	}
	if witnessCount != 1 {
		t.Fatal("actual unique retention allocation missing")
	}
	key, ok := anonymousClassSourceKey(witness)
	retained := ctx.anonymousClasses[key]
	if !ok || retained == nil || retained.scope == nil || retained.functional != nil || retained.lexicalOwner == nil || retained.lexicalOwner.phase != anonymousFunctionalComplete {
		t.Fatal("actual block retention allocation missing")
	}
	role := retained.lexicalOwner
	if role.self != role || role.info != retained || role.graph != symbol.GlobalScope || role.file != ctx.currentFile || role.lexical.currentFile != ctx.currentFile || role.lexical.hoistedDecls != ctx.hoistedDecls || role.lexical.anonClassCounter != ctx.anonClassCounter || role.lexical.anonymousClasses[key] != retained {
		t.Fatal("retention role borrowed a different graph source or emission session")
	}
	local := role.lexical.localClasses["Local"]
	if local == nil || local.scope == nil || role.lexical.currentClass != local.scope || local.scope.Class == nil || !anonymousFunctionalEqual(local.scope.Class.DeclarationNode, localNode) || local.scope.Enclosing != owner.scope || retained.scope.Enclosing != local.scope || !anonymousFunctionalEqual(role.allocation, witness) {
		t.Fatal("actual owning block map or child source role missing")
	}
	if ctx.localClasses["Local"] != nil {
		t.Fatal("block local class escaped into enclosing scope")
	}
	queryFixture := ctx.Clone()
	queryFixture.localClasses = role.lexical.localClasses
	if !anonymousLexicalOwnerValid(role, queryFixture, map[*symbol.ClassScope]bool{}) {
		t.Fatal("actual retained lexical block owner invalid")
	}
	return queryFixture, owner, local
}

// The original full-parent source is lowered once. Its local struct and native
// view must precede the parent body while the anonymous owner is in progress.
// Completed direct-consumer queries use a separate fresh actual block fixture.
func TestCI214AnonymousFunctionalNestedLocalOwnerPipeline(t *testing.T) {
	for _, parent := range []string{"Function", "Supplier", "Runnable"} {
		for _, child := range []string{"Function", "Supplier"} {
			t.Run(parent+"/"+child, func(t *testing.T) {
				source, header, parentName := anonymousFunctional214NestedSource(parent, child)
				ctx, _, hoists, owner := anonymousFunctional209Lower(t, source, header)
				if ctx.localClasses["Local"] != nil || !owner.scope.HasAnonymousSourceOrigin() {
					t.Fatal("actual block scoping or anonymous source origin changed")
				}
				localStruct := ""
				for _, d := range hoists {
					group, ok := d.(*ast.GenDecl)
					if !ok {
						continue
					}
					for _, declaration := range group.Specs {
						typ, ok := declaration.(*ast.TypeSpec)
						if !ok || typ.Name.Name == owner.structName {
							continue
						}
						structure, ok := typ.Type.(*ast.StructType)
						if !ok {
							continue
						}
						for _, field := range structure.Fields.List {
							enclosing, ok := field.Type.(*ast.StarExpr)
							if ok && samIdentity205ReceiverBase(enclosing.X) == owner.structName {
								if localStruct != "" {
									t.Fatal("actual local child struct ambiguous")
								}
								localStruct = typ.Name.Name
							}
						}
					}
				}
				if localStruct == "" {
					t.Fatal("actual named-child/anonymous-owner source pipeline missing")
				}
				if child == "Function" {
					childViewAt, parentBodyAt := -1, -1
					var discoveredMethods []string
					methods := owner.scope.FindMethod().ByOriginalName(parentName)
					if len(methods) != 1 {
						t.Fatal("actual parent method descriptor missing")
					}
					implementation := executionImplementationName(methods[0], owner.scope, ctx)
					for i, d := range hoists {
						f, ok := d.(*ast.FuncDecl)
						if !ok || f.Recv == nil || len(f.Recv.List) != 1 {
							continue
						}
						receiver := samIdentity205ReceiverBase(f.Recv.List[0].Type)
						if receiver == localStruct || receiver == owner.structName {
							discoveredMethods = append(discoveredMethods, anonymousFunctionalAST(f.Recv.List[0].Type)+"."+f.Name.Name)
						}
						if receiver == localStruct && f.Name.Name == generatedObjectViewMethod && anonymousFunctional209Calls(f, "NewFunctionSourceView") == 1 {
							childViewAt = i
						}
						if receiver == owner.structName && f.Name.Name == implementation {
							parentBodyAt = i
						}
					}
					if childViewAt < 0 || parentBodyAt <= childViewAt {
						t.Fatalf("child view did not receive actual in-progress lexical-owner proof: childViewAt=%d parentBodyAt=%d childReceiver=%q methods=%q", childViewAt, parentBodyAt, localStruct, discoveredMethods)
					}
				}
				// Retain the actual owning block in a wholly fresh emission, after the
				// original pipeline AST assertions have finished using their graph/session.
				ctx, owner, local := anonymousFunctional214NestedContext(t, parent, child)
				if local.scope.Enclosing != owner.scope || !owner.scope.HasAnonymousSourceOrigin() || local.scope.HasAnonymousSourceOrigin() {
					t.Fatal("actual named-child/anonymous-owner source pipeline missing")
				}
				query := ctx.Clone()
				query.currentClass = local.scope
				query.localScope = nil
				if !anonymousFunctionalActualNamed(local.scope, query) {
					t.Fatal("actual anonymous lexical owner was forced into namedness")
				}
				if _, anonymous := anonymousFunctionalLookup(local.scope, query); anonymous {
					t.Fatal("real named child was classified as anonymous")
				}
				if child == "Function" {
					contracts := sourceNativeFunctionalContracts(local.scope, query)
					if len(contracts) != 1 || sourceNativeFunctionalViewExpr(local.scope, contracts[0], ast.NewIdent("receiver"), query) == nil {
						t.Fatal("completed local named Function route lost actual ownership")
					}
				}
				if child == "Supplier" {
					if _, owned := supplierSourceDeclarationCtx(local.scope, query); owned {
						t.Fatal("lexical-owner proof widened named Supplier registered-file authority")
					}
				}
			})
		}
	}
}

func TestCI214AnonymousFunctionalNestedOwnerRefusal(t *testing.T) {
	for _, parent := range []string{"Function", "Supplier", "Runnable"} {
		for _, child := range []string{"Function", "Supplier"} {
			for _, mode := range []string{"missing owner info", "copied owner info", "copied owner role", "named owner node", "foreign owner body", "foreign owner method", "owner binder donation", "owner failed phase", "owner local donation", "owner graph donation", "foreign hoist", "foreign counter", "stale input"} {
				t.Run(parent+"/"+child+"/"+mode, func(t *testing.T) {
					ctx, owner, local := anonymousFunctional214NestedContext(t, parent, child)
					source := string(ctx.currentFile.Source)
					role := owner.lexicalOwner
					if role == nil || local == nil || !anonymousFunctionalActualNamed(local.scope, ctx) {
						t.Fatal("actual nested lexical owner premise failed")
					}
					key, _ := anonymousClassSourceKey(role.allocation)
					// Independent equal-span parser root cannot donate an owner body/method.
					foreign := parsing.SourceFile{Name: "Other.java", Source: []byte(source)}
					if err := foreign.ParseAST(); err != nil {
						t.Fatal(err)
					}
					foreignAllocation := findNode(foreign.Ast, "object_creation_expression")
					if foreignAllocation == nil || foreignAllocation.Equal(role.allocation) {
						t.Fatal("independent owner parser premise failed")
					}
					switch mode {
					case "missing owner info":
						delete(ctx.anonymousClasses, key)
					case "copied owner info":
						copied := *owner
						ctx.anonymousClasses[key] = &copied
					case "copied owner role":
						copied := *role
						owner.lexicalOwner = &copied
					case "named owner node":
						owner.scope.Class.DeclarationNode = ctx.currentClass.Class.DeclarationNode
					case "foreign owner body":
						role.body = objectCreationClassBody(foreignAllocation)
					case "foreign owner method":
						owner.scope.Methods[0].DeclarationNode = findNode(foreignAllocation, "method_declaration")
					case "owner binder donation":
						owner.scope.TypeParameters = append(owner.scope.TypeParameters, symbol.NewTypeParam("Donated", nil))
					case "owner failed phase":
						role.phase = anonymousFunctionalFailed
					case "owner local donation":
						delete(ctx.anonymousClasses, key)
						owner.scope.Class.DeclarationNode = ctx.currentClass.Class.DeclarationNode
						ctx.localClasses["donated"] = &localClassInfo{scope: owner.scope}
					case "owner graph donation":
						delete(ctx.anonymousClasses, key)
						owner.scope.Class.DeclarationNode = ctx.currentClass.Class.DeclarationNode
						ctx.currentFile.TopLevelClasses = append(ctx.currentFile.TopLevelClasses, owner.scope)
					case "foreign hoist":
						copied := append([]ast.Decl(nil), (*ctx.hoistedDecls)...)
						ctx.hoistedDecls = &copied
					case "foreign counter":
						copied := *ctx.anonClassCounter
						ctx.anonClassCounter = &copied
					case "stale input":
						ctx.currentFile.SourceFilename = "Other.java"
					}
					before := anonymousFunctionalAST(&ast.File{Name: ast.NewIdent("snapshot"), Decls: *ctx.hoistedDecls})
					beforeCounter := *ctx.anonClassCounter
					query := ctx.Clone()
					query.currentClass = local.scope
					query.localScope = nil
					if anonymousFunctionalActualNamed(local.scope, query) {
						t.Fatal("invalid or donated anonymous owner authorized named child")
					}
					if sourceNativeFunctionalContracts(local.scope, query) != nil {
						t.Fatal("invalid owner granted child native contract")
					}
					c, _ := nativeFunctionalContract(child, func() []string {
						if child == "Function" {
							return []string{"java.lang.String", "java.lang.String"}
						}
						return []string{"java.lang.String"}
					}())
					if sourceNativeFunctionalViewExpr(local.scope, c, ast.NewIdent("receiver"), query) != nil {
						t.Fatal("invalid owner exported child view")
					}
					if anonymousFunctionalAST(&ast.File{Name: ast.NewIdent("snapshot"), Decls: *ctx.hoistedDecls}) != before || *ctx.anonClassCounter != beforeCounter {
						t.Fatal("lexical-owner refusal mutated output or completion state")
					}
				})
			}
		}
	}
}
