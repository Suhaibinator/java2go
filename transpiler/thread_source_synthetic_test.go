package transpiler

import (
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"testing"
)

func TestThreadSourceSyntheticDeclarationFileCertificate(t *testing.T) {
	for _, test := range []struct {
		source string
		want   bool
	}{
		{`class Host{void make(){new java.lang.Thread(){public void run(){}};}}`, true},
		{`import foreign.Thread;class Host{void make(){new Thread(){public void run(){}};}}`, false},
		{`class Host<Thread>{void make(){new Thread(){public void run(){}};}}`, false},
		{`class Host{<Thread>void make(){new Thread(){public void run(){}};}}`, false},
		{`class Host<java>{void make(){new java.lang.Thread(){public void run(){}};}}`, false},
	} {
		t.Run(test.source, func(t *testing.T) {
			h := setupParseHelper(t, test.source)
			owner := resolveClassScopeByQualifiedName(h.Ctx, "Host")
			method := owner.FindMethodByName("make", nil)
			node := findNode(method.DeclarationNode, "object_creation_expression")
			scope := &symbol.ClassScope{Class: &symbol.Definition{Name: "Synthetic", DeclarationNode: node}, Enclosing: owner, Superclass: node.ChildByFieldName("type").Content(h.File.Source)}
			scope.TypeParameters = append(append([]symbol.TypeParam(nil), owner.TypeParameters...), method.TypeParameters...)
			ctx := h.Ctx.Clone()
			ctx.currentClass = scope
			ctx.localScope = method
			if got := threadSourceDirectRoot(scope, ctx); got != test.want {
				t.Fatalf("synthetic native edge=%v want%v", got, test.want)
			}
			if test.want {
				detached := parsing.SourceFile{Name: "Other.java", Source: append([]byte(nil), h.File.Source...)}
				if err := detached.ParseAST(); err != nil {
					t.Fatal(err)
				}
				detachedNode := findNode(detached.Ast, "object_creation_expression")
				copyScope := *scope
				copyClass := *scope.Class
				copyClass.DeclarationNode = detachedNode
				copyScope.Class = &copyClass
				if threadSourceDirectRoot(&copyScope, ctx) {
					t.Fatal("same-span different-tree declaration admitted")
				}
			}
		})
	}
}

func TestThreadSourceLocalMethodBinderOrigin(t *testing.T) {
	h := setupParseHelper(t, `class Host {static <T> void make(T value) {class Local extends java.lang.Thread {public void run(){if(value==null)throw new AssertionError();}} Local local=new Local();local.run();}}`)
	ctx := h.Ctx.Clone()
	ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Host")
	ctx.className = ctx.currentClass.Class.Name
	ctx.localScope = ctx.currentClass.FindMethodByName("make", nil)
	var declarations []ast.Decl
	var counter int
	ctx.hoistedDecls = &declarations
	ctx.anonClassCounter = &counter
	ctx.localClasses = make(map[string]*localClassInfo)
	ctx.anonymousClasses = make(map[anonymousClassKey]*anonymousClassInfo)
	localNode := findNode(ctx.localScope.DeclarationNode, "class_declaration")
	hoistLocalClass(localNode, h.File.Source, ctx)
	local := ctx.localClasses["Local"]
	if local == nil || len(local.scope.TypeParameters) != 1 || len(ctx.localScope.TypeParameters) != 1 {
		t.Fatal("actual hoisted method capture missing")
	}
	methodBinder := ctx.localScope.TypeParameters[0].Declaration
	if local.scope.TypeParameters[0].Declaration != methodBinder {
		t.Fatal("hoisted capture lost original method declaration identity")
	}
	var origin inferredJavaTypeOrigin
	typ, known := inferIdentifierJavaType("local", ctx, &origin)
	t.Logf("actual=%s/%v unresolved=%v captured=%+v method=%+v", typ, known, origin.unresolved, local.scope.TypeParameters[0].Declaration, methodBinder)
	if !known || origin.unresolved || origin.nominalScope != local.scope || len(origin.arguments) != 1 || origin.arguments[0].parameter != methodBinder {
		t.Fatal("local nominal result failed to retain actual visible method binder")
	}
}
