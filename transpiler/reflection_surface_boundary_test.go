package transpiler

import (
	"go/ast"
	"reflect"
	"testing"
)

func TestReflectionSourceNativeCarrierStrictBoundary(t *testing.T) {
	for _, row := range []struct{ name, source, target string }{
		{"protected-accessible-subclass", `class Manual extends java.lang.reflect.AccessibleObject {}`, "Manual"},
		{"manual-annotation", `class Manual implements java.lang.annotation.Annotation {public Class<? extends java.lang.annotation.Annotation> annotationType(){return null;}}`, "Manual"},
		{"annotation-interface-extension", `interface Manual extends java.lang.annotation.Annotation {}`, "Manual"},
		{"manual-source-annotation", `@interface Tag {int value();} class Manual implements Tag {public int value(){return 1;} public Class<? extends java.lang.annotation.Annotation> annotationType(){return Tag.class;}}`, "Manual"},
		{"manual-member", `abstract class Manual implements java.lang.reflect.Member {}`, "Manual"},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.source)
			scope := h.Ctx.currentFile.FindClassScope(row.target)
			ctx := classScopeCtx(scope, h.Ctx)
			ctx.className = scope.Class.Name
			strictRoutingState(t)
			ctx.importAliases = map[string]string{"kept": "kept"}
			ctx.usedImports = map[string]bool{"kept": true}
			var hoisted []ast.Decl
			ctx.hoistedDecls = &hoisted
			var result []ast.Decl
			var failure any
			func() {
				defer func() { failure = recover() }()
				result = parseDecls(scope.Class.DeclarationNode, h.File.Source, ctx)
			}()
			rejected, ok := failure.(strictModeError)
			if !ok || rejected.diagnostic.Kind != "reflection source capability" {
				t.Fatalf("wrong strict boundary: %T %v", failure, failure)
			}
			if result != nil || len(hoisted) != 0 || !reflect.DeepEqual(ctx.importAliases, map[string]string{"kept": "kept"}) || !reflect.DeepEqual(ctx.usedImports, map[string]bool{"kept": true}) {
				t.Fatal("refusal created partial output")
			}
		})
	}
}

func TestReflectionCarrierHomonymsCannotBorrowNativeIdentity(t *testing.T) {
	h := setupParseHelper(t, `class AccessibleObject {} class Annotation {} interface Member {} class Manual extends AccessibleObject {}`)
	ctx := h.Ctx
	ctx.currentClass = ctx.currentFile.FindClassScope("Manual")
	if reflectionSourceCarrierUnsupported(ctx.currentClass, ctx) {
		t.Fatal("ordinary source inheritance refused as native")
	}
	for _, name := range []string{"AccessibleObject", "Annotation", "Member"} {
		if _, native := reflectionSurfaceRuntimeTypeExpr(name, nil, ctx); native {
			t.Fatalf("homonym %s received native physical carrier", name)
		}
	}
	if reflectionSurfaceReferenceAssignable("java.lang.reflect.Field", "AccessibleObject", ctx) {
		t.Fatal("native Field assigned to source homonym")
	}
	if reflectionSurfaceReferenceAssignable("Annotation", "java.lang.annotation.Annotation", ctx) {
		t.Fatal("source homonym assigned nominal Annotation")
	}
}

func TestReflectionCallerMissingOwnedFrameStrictlyRefuses(t *testing.T) {
	h := setupParseHelper(t, `class Probe {boolean run(java.lang.reflect.AccessibleObject member,Object receiver){return member.canAccess(receiver);}}`)
	ctx := h.Ctx
	ctx.currentClass = ctx.currentFile.FindClassScope("Probe")
	ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
	ctx.runtimeLocationProfile = sourceRuntimeLocationClasspath
	ctx.executionContextName = "execution"
	invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
	if id, ok := reflectionSourceCallerID(invocation, ctx); ok || id != "" {
		t.Fatal("unowned caller got privileged ID")
	}
	strictRoutingState(t)
	var result ast.Expr
	var failure any
	func() { defer func() { failure = recover() }(); result = ParseExpr(invocation, h.File.Source, ctx) }()
	rejected, ok := failure.(strictModeError)
	if !ok || rejected.diagnostic.Kind != "intrinsic invocation" || result != nil {
		t.Fatalf("missing frame boundary=%T %v %T", failure, failure, result)
	}
}

func TestReflectionAnnotationDeclarationCannotBeDonated(t *testing.T) {
	h := setupParseHelper(t, `@interface First {} @interface Second {} class Consumer {}`)
	ctx := h.Ctx
	ctx.currentClass = ctx.currentFile.FindClassScope("Consumer")
	first := ctx.currentFile.FindClassScope("First")
	second := ctx.currentFile.FindClassScope("Second")
	if !reflectionSurfaceReferenceAssignable("First", "java.lang.annotation.Annotation", ctx) {
		t.Fatal("genuine owned annotation edge missing")
	}
	original := first.Class.DeclarationNode
	first.Class.DeclarationNode = second.Class.DeclarationNode
	defer func() { first.Class.DeclarationNode = original }()
	if reflectionSurfaceReferenceAssignable("First", "java.lang.annotation.Annotation", ctx) {
		t.Fatal("sibling annotation declaration donated nominal identity")
	}
}

func TestReflectionRecordDoesNotBorrowObjectTextDefault(t *testing.T) {
	h := setupParseHelper(t, `record Entry(int value) {}`)
	ctx := h.Ctx
	ctx.currentClass = ctx.currentFile.FindClassScope("Entry")
	if sourceInheritsObjectTextDefault(ctx.currentClass, ctx) {
		t.Fatal("implicit record toString misrepresented as Object default")
	}
}

func TestReflectionRecordComponentPropagationStrictRefusal(t *testing.T) {
	h := setupParseHelper(t, `@java.lang.annotation.Retention(java.lang.annotation.RetentionPolicy.RUNTIME) @interface Mark {} record Entry(@Mark int value) {} class Use {void query() throws Exception {Entry.class.getDeclaredField("value");}}`)
	scope := h.Ctx.currentFile.FindClassScope("Entry")
	ctx := classScopeCtx(scope, h.Ctx)
	strictRoutingState(t)
	var result []ast.Decl
	var failure any
	func() {
		defer func() { failure = recover() }()
		result = parseDecls(scope.Class.DeclarationNode, h.File.Source, ctx)
	}()
	refused, ok := failure.(strictModeError)
	if !ok || refused.diagnostic.Kind != "record component annotation propagation" || result != nil {
		t.Fatalf("unproved propagation emitted metadata: %T %v", failure, failure)
	}
}

func TestReflectionRecordUnprovidedStaticStateStrictRefusal(t *testing.T) {
	for _, body := range []string{`static int state=1;`, `static {System.out.print("");}`} {
		t.Run(body, func(t *testing.T) {
			h := setupParseHelper(t, `record Entry(int value) {`+body+`} class Use {void query() throws Exception {Entry.class.getDeclaredField("value");}}`)
			scope := h.Ctx.currentFile.FindClassScope("Entry")
			ctx := classScopeCtx(scope, h.Ctx)
			strictRoutingState(t)
			var result []ast.Decl
			var failure any
			func() {
				defer func() { failure = recover() }()
				result = parseDecls(scope.Class.DeclarationNode, h.File.Source, ctx)
			}()
			refused, ok := failure.(strictModeError)
			if !ok || refused.diagnostic.Kind != "record reflection static state" || result != nil {
				t.Fatalf("unprovided record state emitted callbacks: %T %v", failure, failure)
			}
		})
	}
}

func TestReflectionSurfaceTypeBindersCannotBorrowNativeIdentity(t *testing.T) {
	for _, row := range []struct{ name, source, method string }{
		{"class", `class Probe<Annotation,AccessibleObject,Member> {}`, ""},
		{"method", `class Probe {<Annotation,AccessibleObject,Member> void run(){}}`, "run"},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.source)
			ctx := h.Ctx
			ctx.currentClass = ctx.currentFile.FindClassScope("Probe")
			if row.method != "" {
				ctx.localScope = ctx.currentClass.FindMethodByName(row.method, nil)
			}
			for _, name := range []string{"Annotation", "AccessibleObject", "Member"} {
				if _, native := reflectionSurfaceRuntimeTypeExpr(name, nil, ctx); native {
					t.Fatalf("type binder %s borrowed native carrier", name)
				}
			}
			if reflectionSurfaceReferenceAssignable("java.lang.reflect.Field", "AccessibleObject", ctx) {
				t.Fatal("binder borrowed nominal AccessibleObject edge")
			}
		})
	}
}

func TestReflectionSurfaceCanonicalIdentityAndObjectAncestry(t *testing.T) {
	h := setupParseHelper(t, `import java.lang.reflect.AccessibleObject;import java.lang.reflect.Member;import java.lang.annotation.Annotation;class Probe {}`)
	for _, row := range [][2]string{{"AccessibleObject", "java.lang.reflect.AccessibleObject"}, {"java.lang.reflect.AccessibleObject", "AccessibleObject"}, {"Member", "java.lang.reflect.Member"}, {"Annotation", "java.lang.annotation.Annotation"}, {"AccessibleObject", "java.lang.Object"}, {"Member", "java.lang.Object"}, {"Annotation", "java.lang.Object"}} {
		if !builtinJavaReferenceAssignable(row[0], row[1], h.Ctx) {
			t.Errorf("canonical native reference %s -> %s rejected", row[0], row[1])
		}
	}
}
