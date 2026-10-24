package transpiler

import (
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"testing"
)

func TestLocalCapturedMethodOriginRetainsDeclaration(t *testing.T) {
	for _, source := range []string{
		`class Host<T>{<T>void make(T value){class Local extends java.lang.Thread{public void run(){if(value==null)throw new AssertionError();}}Local local=new Local();} <T>void other(T value){}}`,
		`class Host{static class Value{int mark(){return 1;}}static<T extends Value>void make(T value){class Local{T get(){return value;}}Local local=new Local();}static<T>void other(T value){}}`,
		`class Host{static<T extends java.lang.CharSequence>void make(T value){class Local extends java.lang.Thread{public void run(){value.length();}}Local local=new Local();}static<T>void other(T value){}}`,
		`class Host{static<T>void make(T[] value){class Local{T[] get(){return value;}}Local local=new Local();}static<T>void other(T value){}}`,
	} {
		t.Run(source, func(t *testing.T) {
			h := setupParseHelper(t, source)
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
			hoistLocalClass(findNode(ctx.localScope.DeclarationNode, "class_declaration"), h.File.Source, ctx)
			local := ctx.localClasses["Local"]
			if local == nil {
				t.Fatal("actual local producer absent")
			}
			methodBinder := ctx.localScope.TypeParameters[0].Declaration
			captured := -1
			for i, p := range local.scope.TypeParameters {
				if p.Declaration == methodBinder {
					captured = i
				}
			}
			if captured < 0 {
				t.Fatal("actual method binder not captured")
			}
			var origin inferredJavaTypeOrigin
			_, known := inferIdentifierJavaType("local", ctx, &origin)
			if !known || origin.unresolved || captured >= len(origin.arguments) || origin.arguments[captured].parameter != methodBinder {
				t.Errorf("origin failed original method pointer: unresolved=%v arguments=%+v", origin.unresolved, origin.arguments)
			}
			for _, classBinder := range ctx.currentClass.TypeParameters {
				if classBinder.Declaration == methodBinder {
					t.Fatal("class/method same-name declarations collapsed")
				}
			}
			other := ctx.Clone()
			other.localScope = ctx.currentClass.FindMethodByName("other", nil)
			foreign := declaredJavaTypeOrigin(symbol.JavaType{Original: "Local", TypeParameterBindings: symbol.VisibleTypeParamBindings(other.localScope.TypeParameters)}, other)
			if !foreign.unresolved || foreign.nominalScope != nil {
				t.Fatal("other executable's same-name binder rebound old local type")
			}
		})
	}
}

func TestLocalCapturedMethodOriginRejectsForeignDeclarations(t *testing.T) {
	h := setupParseHelper(t, `class Host {static<T>void make(T value){class Local{T get(){return value;}}Local local=new Local();}static<T>void other(T value){}}`)
	ctx := h.Ctx.Clone()
	ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Host")
	ctx.className = ctx.currentClass.Class.Name
	ctx.localScope = ctx.currentClass.FindMethodByName("make", nil)
	var declarations []ast.Decl
	ctx.hoistedDecls = &declarations
	ctx.localClasses = make(map[string]*localClassInfo)
	ctx.anonymousClasses = make(map[anonymousClassKey]*anonymousClassInfo)
	node := findNode(ctx.localScope.DeclarationNode, "class_declaration")
	hoistLocalClass(node, h.File.Source, ctx)
	scope := ctx.localClasses["Local"].scope
	original := ctx.localScope.TypeParameters[0].Declaration
	if value, ok := localCapturedMethodTypeOrigin(scope, original, ctx); !ok || value.parameter != original {
		t.Fatal("original method declaration rejected")
	}
	sameSpelling := *original
	if _, ok := localCapturedMethodTypeOrigin(scope, &sameSpelling, ctx); ok {
		t.Fatal("different declaration with same spelling admitted")
	}
	other := ctx.Clone()
	other.localScope = ctx.currentClass.FindMethodByName("other", nil)
	if _, ok := localCapturedMethodTypeOrigin(scope, original, other); ok {
		t.Fatal("foreign executable rebound capture")
	}
	unknown := ctx.Clone()
	unknown.localScope = nil
	unknown.currentClass = nil
	if _, ok := localCapturedMethodTypeOrigin(scope, original, unknown); ok {
		t.Fatal("unknown executable admitted")
	}
	detached := parsing.SourceFile{Name: "Detached.java", Source: append([]byte(nil), h.File.Source...)}
	if err := detached.ParseAST(); err != nil {
		t.Fatal(err)
	}
	copyScope := *scope
	copyClass := *scope.Class
	copyClass.DeclarationNode = findNode(findNode(detached.Ast, "method_declaration"), "class_declaration")
	copyScope.Class = &copyClass
	if _, ok := localCapturedMethodTypeOrigin(&copyScope, original, ctx); ok {
		t.Fatal("same-span detached tree admitted")
	}
}

func TestLocalCapturedConstructorBinderOrigin(t *testing.T) {
	h := setupParseHelper(t, `class Host {<T>Host(T value){class Local{T get(){return value;}}Local local=new Local();}}`)
	ctx := h.Ctx.Clone()
	ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Host")
	ctx.className = ctx.currentClass.Class.Name
	for _, method := range ctx.currentClass.Methods {
		if method.Constructor {
			ctx.localScope = method
			break
		}
	}
	if ctx.localScope == nil || len(ctx.localScope.TypeParameters) != 1 {
		t.Fatal("registered original constructor binder missing")
	}
	var declarations []ast.Decl
	ctx.hoistedDecls = &declarations
	ctx.localClasses = make(map[string]*localClassInfo)
	ctx.anonymousClasses = make(map[anonymousClassKey]*anonymousClassInfo)
	hoistLocalClass(findNode(ctx.localScope.DeclarationNode, "class_declaration"), h.File.Source, ctx)
	var origin inferredJavaTypeOrigin
	_, known := inferIdentifierJavaType("local", ctx, &origin)
	if !known || origin.unresolved || len(origin.arguments) != 1 || origin.arguments[0].parameter != ctx.localScope.TypeParameters[0].Declaration {
		t.Fatal("registered constructor binder not retained", origin)
	}
}
