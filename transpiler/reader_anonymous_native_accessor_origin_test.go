package transpiler

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

const reader132OriginalJava = "package members;import java.io.Reader;\npublic class Main{static boolean check(Object __java2goAnonymous){return new Reader(__java2goAnonymous){public Object ReaderBase;boolean first=lock==__java2goAnonymous;public int read(char[]b,int o,int n){return -1;}public void close(){}boolean same(){return first&&lock==__java2goAnonymous;}}.same();}\npublic static void main(String[]args){boolean boxed=new Reader(7){public int read(char[]b,int o,int n){return -1;}public void close(){}boolean same(){return lock instanceof java.lang.Integer&&((java.lang.Integer)lock).intValue()==7;}}.same();System.out.println(check(new Object())+\":\"+boxed);}}"

func reader132NativeInvocation(node *sitter.Node, source []byte) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "method_invocation" {
		if name := node.ChildByFieldName("name"); name != nil && name.Content(source) == "intValue" {
			return node
		}
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		if found := reader132NativeInvocation(node.NamedChild(i), source); found != nil {
			return found
		}
	}
	return nil
}

func reader132Render(t *testing.T, expression ast.Expr) string {
	t.Helper()
	if expression == nil {
		return "<nil>"
	}
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), expression); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestReader132AnonymousNativeAccessorDeclarationOrigin(t *testing.T) {
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(reader132OriginalJava))); got != "23f1ca6506aac292b00d8d124addfb51a1f4de0c1c2e0faca4ae50b70b089495" {
		t.Fatal("original Java byte pin changed", got)
	}
	t.Run("registered_owner", func(t *testing.T) {
		h := setupParseHelper(t, `class Named {int read(Object x){return ((java.lang.Integer)x).intValue();}}`)
		ctx := h.Ctx.Clone()
		ctx.className = ctx.currentClass.Class.Name
		ctx.localScope = ctx.currentClass.FindMethodByName("read", nil)
		invocation := reader132NativeInvocation(ctx.localScope.DeclarationNode, h.File.Source)
		if invocation == nil {
			t.Fatal("registered source invocation missing")
		}
		key, known := intrinsicReceiverTypeName(invocation.ChildByFieldName("object"), ctx, h.File.Source)
		lowered, rewritten := tryInstanceIntrinsic(invocation.ChildByFieldName("object"), "intValue", h.File.Source, ctx)
		rendered := reader132Render(t, lowered)
		t.Logf("registered owner=%p directFile=%p actualFile=%p key=%q known=%v rewritten=%v expression=%s", ctx.currentClass, findFileScopeForClassScope(ctx.currentClass, ctx), ctx.currentFile, key, known, rewritten, rendered)
		if !known || key != "Integer" || !rewritten || !strings.Contains(rendered, "NumberIntValue(") {
			t.Fatal("existing registered numeric accessor callback missing")
		}
	})
	h := setupParseHelper(t, reader132OriginalJava)
	ctx := h.Ctx.Clone()
	ctx.className = ctx.currentClass.Class.Name
	var declarations []ast.Decl
	var counter int
	ctx.hoistedDecls = &declarations
	ctx.anonClassCounter = &counter
	ctx.anonymousClasses = make(map[anonymousClassKey]*anonymousClassInfo)
	ctx.localClasses = make(map[string]*localClassInfo)
	ctx.executionContextName = "__java2goExecution"
	for _, name := range []string{"check", "main"} {
		ctx.localScope = ctx.currentClass.FindMethodByName(name, nil)
		if ctx.localScope == nil {
			t.Fatal("actual original method absent", name)
		}
		ParseStmt(ctx.localScope.DeclarationNode.ChildByFieldName("body"), h.File.Source, ctx)
	}
	if counter != 2 || len(ctx.anonymousClasses) != 2 {
		t.Fatalf("actual source must produce both original anonymous scopes: counter=%d registry=%d", counter, len(ctx.anonymousClasses))
	}
	main := ctx.currentClass.FindMethodByName("main", nil)
	original := reader132NativeInvocation(main.DeclarationNode, h.File.Source)
	if original == nil {
		t.Fatal("original anonymous intValue invocation missing")
	}
	var creation *sitter.Node
	for current := original; current != nil; current = current.Parent() {
		if current.Type() == "object_creation_expression" {
			creation = current
			break
		}
	}
	key, hasKey := anonymousClassSourceKey(creation)
	if !hasKey {
		t.Fatal("original anonymous source key missing")
	}
	info := ctx.anonymousClasses[key]
	if info == nil || info.scope == nil || info.scope.Class == nil || info.structName != "MainAnon2" {
		t.Fatal("actual second anonymous scope missing")
	}
	scope := info.scope
	method := scope.FindMethodByName("same", nil)
	if method == nil {
		t.Fatal("actual registered anonymous same declaration missing")
	}
	invocation := reader132NativeInvocation(method.DeclarationNode, h.File.Source)
	if invocation == nil || !invocation.Equal(original) {
		t.Fatal("synthetic method lost original invocation tree identity")
	}
	actual := ctx.Clone()
	actual.currentClass = scope
	actual.localScope = method
	actual.className = info.structName
	receiver := invocation.ChildByFieldName("object")
	var origin inferredJavaTypeOrigin
	display, known := inferExprJavaType(receiver, actual, h.File.Source, &origin)
	proof := actual.Clone()
	proof.currentFile = nil
	proof.syntheticTypeParameters = nil
	declaring := classScopeCtx(origin.declaringOwner, proof)
	direct := findFileScopeForClassScope(scope, proof)
	enclosing := findFileScopeForClassScope(scope.Enclosing, proof)
	nativeKey, nativeKnown := intrinsicReceiverTypeName(receiver, actual, h.File.Source)
	lowered, rewritten := tryInstanceIntrinsic(receiver, "intValue", h.File.Source, actual)
	emitted := reader132Render(t, ParseExpr(invocation, h.File.Source, actual))
	t.Logf("registry=%p scope=%p declaringOwner=%p enclosingOwner=%p method=%p objectType=%s display=%q external=%q complete=%v unresolved=%v nominal=%p binder=%p rank=%d wildcard=%v args=%d directFile=%p classScopeCtxFile=%p enclosingFile=%p currentFile=%p scopeNodeEqual=%v actualInvocationEqual=%v exactAncestor=%v selectedKey=%q selectedKnown=%v callback=%v result=%s emitted=%s", info.scope, scope, origin.declaringOwner, scope.Enclosing, method, receiver.Type(), display, origin.externalName, completeJavaTypeOrigin(origin), origin.unresolved, origin.nominalScope, origin.parameter, origin.arrayRank, origin.wildcard, len(origin.arguments), direct, declaring.currentFile, enclosing, actual.currentFile, scope.Class.DeclarationNode.Equal(creation), invocation.Equal(original), threadSourceNodeWithin(scope.Class.DeclarationNode, scope.Enclosing.Class.DeclarationNode), nativeKey, nativeKnown, rewritten, reader132Render(t, lowered), emitted)
	t.Run("late_anonymous_certificate", func(t *testing.T) {
		if !known || display != "java.lang.Integer" || origin.externalName != "java.lang.Integer" || !completeJavaTypeOrigin(origin) || origin.declaringOwner != scope || origin.nominalScope != nil || origin.parameter != nil || origin.arrayRank != 0 || origin.wildcard || len(origin.arguments) != 0 {
			t.Fatal("complete original canonical cast origin not proved")
		}
		if scope.Enclosing != ctx.currentClass || !scope.Class.DeclarationNode.Equal(creation) || !threadSourceNodeWithin(creation, scope.Enclosing.Class.DeclarationNode) {
			t.Fatal("actual lexical source owner not proved")
		}
		if enclosing != h.File.Symbols || declaring.currentFile != enclosing || direct != nil {
			t.Fatal("specific direct registration certificate discrepancy not reproduced")
		}
		if instanceIntrinsics[intrinsicKey{"Integer", "intValue"}] == nil || !intrinsicOwners["Integer"]["java.lang.Integer"] {
			t.Fatal("existing canonical callback absent")
		}
	})
	t.Run("canonical_accessor_emission", func(t *testing.T) {
		if !nativeKnown || nativeKey != "Integer" || !rewritten || !strings.Contains(reader132Render(t, lowered), "NumberIntValue(") || strings.Contains(emitted, ".intValue(") {
			t.Fatalf("canonical native accessor lost after exact anonymous source ownership: key=%q known=%v callback=%v emitted=%s", nativeKey, nativeKnown, rewritten, emitted)
		}
	})
}
