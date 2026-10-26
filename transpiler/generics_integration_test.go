package transpiler

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

func renderGoFileFromJava(t *testing.T, src string) string {
	t.Helper()
	helper := setupParseHelper(t, src)
	node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
	file, ok := node.(*ast.File)
	if !ok {
		t.Fatalf("Expected *ast.File, got %T", node)
	}
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), file); err != nil {
		t.Fatalf("Failed to print AST: %v", err)
	}
	return buf.String()
}

func normalizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func TestGenericsIntegration_GenericClassAndNestedTypes(t *testing.T) {
	src := `
package gen.integration;
public class Pair<K extends Number, V> {
    K key;
    V value;
    public Pair(K k, V v) {
        this.key = k;
        this.value = v;
    }
    public K getKey() { return this.key; }
    public V getValue() { return this.value; }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "type Pair[K stdjava.JavaNumber, V any] struct") {
		t.Errorf("Expected generic struct with 2 type params, got:\n%s", out)
	}
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "key stdjava.JavaNumber") || !strings.Contains(flat, "value V") {
		t.Errorf("Expected bounded key storage to retain Java's Number erasure and unbounded value storage V, got:\n%s", out)
	}
	if !strings.Contains(out, "func NewPair[K stdjava.JavaNumber, V any]") {
		t.Errorf("Expected generic constructor function with type params, got:\n%s", out)
	}
	if !strings.Contains(out, "func (pr *Pair[K, V]) GetKey() stdjava.JavaNumber") {
		t.Errorf("Expected instantiated method receiver and erased Number result, got:\n%s", out)
	}
}

func TestGenericsIntegration_NestedGenericTypeExpressions(t *testing.T) {
	assertMapReferenceTypedRead(t, "java.util.List<java.lang.Integer>")
	src := `
package gen.integration2;
import java.util.List;
import java.util.Map;
public class Container {
    Map<String, List<Integer>> m;
}
`
	out := renderGoFileFromJava(t, src)
	// Nested Java generic arguments retain wrapper reference identity.
	if !strings.Contains(strings.Join(strings.Fields(out), " "), "m stdjava.JavaMap") {
		t.Errorf("Expected erased nominal Map field, got:\n%s", out)
	}
	helper := setupParseHelper(t, src)
	if descriptor, known := javaTypeDescriptorExpr("java.util.Map<String,List<Integer>>", helper.Ctx); !known || descriptor == nil {
		t.Fatal("Map descriptor lost")
	}
}

func TestGenericsIntegration_DiamondExplicitAndRawConstructors(t *testing.T) {
	src := `
package gen.integration3;
public class Box<T> {
    T value;
    public Box() {}
    public static void test() {
        Box<String> inferred = new Box<>();
        Box<Integer> explicit = new Box<Integer>();
        Box raw = new Box();
    }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "NewBoxJava2goExecution[*stdjava.JavaString](__java2goExecution)") {
		t.Errorf("Expected diamond operator to infer canonical String type arg, got:\n%s", out)
	}
	if !strings.Contains(out, "NewBoxJava2goExecution[*stdjava.Integer](__java2goExecution)") {
		t.Errorf("Expected explicit type args on constructor call, got:\n%s", out)
	}
	if !strings.Contains(out, "raw := NewBoxJava2goExecution[any](__java2goExecution)") &&
		!strings.Contains(out, "raw = NewBoxJava2goExecution[any](__java2goExecution)") {
		t.Errorf("Expected raw 'new Box()' to carry its erased any argument explicitly, got:\n%s", out)
	}
}

func TestGenericsIntegration_InstanceGenericMethodHelper_EndToEnd(t *testing.T) {
	src := `
package gen.integration4;
public class Box<T> {
    public <R> R identity(R value) { return value; }

    public static Foo callFoo(Box<Foo> box, Foo value) {
        return box.identity(value);
    }

    public static <X> X callGeneric(Box<X> box, X value) {
        return box.identity(value);
    }
}
`
	helper := genericsInstanceParseWithPublicFoo(t, src)

	owner := helper.File.Symbols.FindClassScope("Box")
	if owner == nil || len(owner.TypeParameters) != 1 {
		t.Fatal("original Box owner lost")
	}
	identity := owner.FindMethodByName("identity", nil)
	if identity == nil || identity.IsPrivate || identity.IsStatic || !identity.RequiresHelper || len(identity.TypeParameters) != 1 || len(identity.Parameters) != 1 {
		t.Fatal("original public instance identity declaration lost")
	}
	ownerParameter := owner.TypeParameters[0]
	if ownerParameter.Name != "T" || ownerParameter.Declaration == nil || len(ownerParameter.Bounds) != 0 || identity.OriginalType != "R" || identity.Parameters[0].OriginalType != "R" {
		t.Fatal("original logical owner T or method R signature lost")
	}
	r := identity.TypeParameters[0].Declaration
	if r == nil || r == owner.TypeParameters[0].Declaration || definitionDeclaredJavaType(identity).TypeParameterBindings["R"] != r || identity.Parameters[0].TypeParameterBindings["R"] != r {
		t.Fatal("method-owned R declaration identity lost")
	}
	assertIdent := func(expr ast.Expr, name string) {
		t.Helper()
		id, ok := expr.(*ast.Ident)
		if !ok || id.Name != name {
			t.Fatalf("expected %s identifier, got %#v", name, expr)
		}
	}
	assertRuntime := func(expr ast.Expr, name string) {
		t.Helper()
		selector, ok := expr.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != name {
			t.Fatalf("expected runtime %s selector, got %#v", name, expr)
		}
		assertIdent(selector.X, "stdjava")
	}
	ownerCtx := helper.Ctx.Clone()
	ownerCtx.currentClass = owner
	family := canonicalGenericFamily(owner, ownerCtx)
	if family == nil || !canonicalGenericClass(owner, ownerCtx) {
		t.Fatal("original generic-X demand lost its completed physical family")
	}
	if _, member := family.members[owner]; !member {
		t.Fatal("physical family lost actual source owner membership")
	}
	if _, binder := family.binders[ownerParameter.Declaration]; !binder {
		t.Fatal("physical family lost original owner T declaration")
	}
	ownerRepresentation, represented := family.representations[ownerParameter.Declaration]
	if !represented || ownerRepresentation.erasure != "java.lang.Object" || ownerRepresentation.goName != "any" {
		t.Fatal("owner T lost independent Object physical representation")
	}
	assertIdent(genericFamilyBinderGoType(ownerParameter, ownerCtx), "any")
	rawName := availableCanonicalGenericName(owner.Class.Name + "Java2goErased")
	check := func(callerName string, foo bool) {
		t.Helper()
		caller := owner.FindMethodByName(callerName, nil)
		if caller == nil {
			t.Fatal("original caller lost")
		}
		ctx := helper.Ctx.Clone()
		ctx.localScope = caller
		ctx.expectedType, ctx.expectedTypeRoot = "", nil
		invocation := findNode(caller.DeclarationNode, "method_invocation")
		if invocation == nil || invocation.ChildByFieldName("object").Content(helper.File.Source) != "box" || invocation.ChildByFieldName("arguments").NamedChildCount() != 1 || invocation.ChildByFieldName("arguments").NamedChild(0).Content(helper.File.Source) != "value" {
			t.Fatal("original invocation nodes changed")
		}
		assertLogical := func(expr ast.Expr) {
			t.Helper()
			if foo {
				p, ok := expr.(*ast.StarExpr)
				if !ok {
					t.Fatalf("Foo lost pointer: %T", expr)
				}
				assertIdent(p.X, "Foo")
				return
			}
			if len(caller.TypeParameters) != 1 || caller.TypeParameters[0].Name != "X" || caller.TypeParameters[0].Declaration == nil || caller.TypeParameters[0].Declaration == r || caller.Parameters[1].TypeParameterBindings["X"] != caller.TypeParameters[0].Declaration {
				t.Fatal("caller-owned X lost its declaration identity")
			}
			assertIdent(expr, caller.TypeParameters[0].EmittedName())
		}
		target := resolveInvocationTarget(invocation.ChildByFieldName("object"), ctx, helper.File.Source)
		resolved, selected := findBestMethodForInvocationTarget(target, "identity", invocation.ChildByFieldName("arguments"), true, false, ctx, helper.File.Source)
		if target == nil || selected == nil || resolved == nil || resolved.def != identity || resolved.owner != owner || target.classScope != owner || len(target.classTypeArgs) != 1 {
			t.Fatal("actual selected owner/method declaration lost")
		}
		assertLogical(target.classTypeArgs[0])
		bindings := genericJavaInvocationTypeBindings(resolved.def, invocation, ctx, helper.File.Source)
		if len(bindings) != 1 || bindings["R"] == "" {
			t.Fatalf("actual no-target R inference lost: %v", bindings)
		}
		assertLogical(javaTypeStringToGoTypeExpr(bindings["R"], inScopeTypeParameters(ctx), ctx))
		logical := inferMethodTypeArguments(resolved.def, invocation, ctx, helper.File.Source)
		if len(logical) != 1 {
			t.Fatal("logical R arity changed")
		}
		assertLogical(logical[0])
		plan := planGenericMethodEntryABI(owner, identity, ctx)
		if plan == nil || plan.owner != owner || plan.method != identity || len(plan.methodDeclarations) != 1 || plan.methodDeclarations[0] != r || len(plan.parameterTypes) != 1 || plan.resultJavaType != "java.lang.Object" {
			t.Fatal("actual declaration-owned erased entry lost")
		}
		assertIdent(plan.parameterTypes[0], "any")
		assertIdent(plan.resultType, "any")
		physical, javaTypes, known := genericMethodPhysicalTypeArguments(identity, ctx)
		if !known || len(physical) != 1 || len(javaTypes) != 1 || javaTypes[0] != "java.lang.Object" {
			t.Fatal("helper physical R first-bound slot changed")
		}
		assertIdent(physical[0], "any")
		verifyExpression := func(expr ast.Expr, execution string) {
			t.Helper()
			view, ok := expr.(*ast.CallExpr)
			if !ok || len(view.Args) != 2 {
				t.Fatalf("one typed consumer view required: %#v", expr)
			}
			indexed, ok := view.Fun.(*ast.IndexExpr)
			if !ok {
				t.Fatal("consumer type argument lost")
			}
			assertRuntime(indexed.X, "ObjectView")
			assertLogical(indexed.Index)
			if foo {
				if reflectCoreTypeID(t, view.Args[1]) != "gen.integration4.Foo" {
					t.Fatal("original synthetic Foo nominal descriptor changed")
				}
			} else {
				assertRuntime(view.Args[1], "ObjectTypeID")
			}
			invoke, ok := view.Args[0].(*ast.CallExpr)
			if !ok {
				t.Fatal("consumer no longer encloses one call")
			}
			selector, ok := invoke.Fun.(*ast.SelectorExpr)
			if !ok {
				t.Fatal("instance selector lost")
			}
			assertIdent(selector.X, "box")
			if execution == "" {
				if selector.Sel.Name != identity.Name || len(invoke.Args) != 1 {
					t.Fatal("original no-Execution parse context changed")
				}
				assertIdent(invoke.Args[0], "value")
			} else {
				if selector.Sel.Name != executionImplementationName(identity, owner, ctx) || len(invoke.Args) != 2 {
					t.Fatal("Execution forwarding changed")
				}
				assertIdent(invoke.Args[0], execution)
				assertIdent(invoke.Args[1], "value")
			}
		}
		verifyExpression(ParseExpr(invocation, helper.File.Source, ctx), "")
		withExecution := ctx.Clone()
		withExecution.executionContextName = "unitExecution"
		verifyExpression(ParseExpr(invocation, helper.File.Source, withExecution), "unitExecution")
	}
	check("callFoo", true)
	check("callGeneric", false)

	node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
	file, ok := node.(*ast.File)
	if !ok {
		t.Fatalf("expected complete file AST: %T", node)
	}
	var logicalAlias, physicalLayout *ast.TypeSpec
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, specification := range general.Specs {
			typ, ok := specification.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if typ.Name.Name == owner.Class.Name {
				if logicalAlias != nil {
					t.Fatal("duplicate logical owner alias")
				}
				logicalAlias = typ
			}
			if typ.Name.Name == rawName {
				if physicalLayout != nil {
					t.Fatal("duplicate physical owner layout")
				}
				physicalLayout = typ
			}
		}
	}
	if logicalAlias == nil || !logicalAlias.Assign.IsValid() || logicalAlias.TypeParams == nil || len(logicalAlias.TypeParams.List) != 1 || len(logicalAlias.TypeParams.List[0].Names) != 1 || physicalLayout == nil || physicalLayout.TypeParams != nil {
		t.Fatal("logical owner T alias and non-generic physical class layout lost")
	}
	assertIdent(logicalAlias.Type, rawName)
	assertIdent(logicalAlias.TypeParams.List[0].Type, "any")
	if logicalAlias.TypeParams.List[0].Names[0].Name != ownerParameter.EmittedName() {
		t.Fatal("logical owner T declaration alias renamed")
	}
	if _, ok := physicalLayout.Type.(*ast.StructType); !ok {
		t.Fatal("physical class layout stopped being a shared struct")
	}
	var entry, body *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "IdentityJava2goExecution" || fn.Recv == nil || len(fn.Recv.List) != 1 {
			continue
		}
		p, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			t.Fatal("instance receiver pointer lost")
		}
		switch x := p.X.(type) {
		case *ast.Ident:
			if x.Name == rawName {
				if entry != nil || fn.Type.TypeParams != nil {
					t.Fatal("non-generic physical entry duplicated or regained owner T")
				}
				entry = fn
			}
		case *ast.IndexListExpr:
			if id, ok := x.X.(*ast.Ident); ok && id.Name == identity.HelperName {
				if body != nil || len(x.Indices) != 2 {
					t.Fatal("typed body slot count changed")
				}
				assertIdent(x.Indices[0], "T")
				assertIdent(x.Indices[1], "R")
				body = fn
			}
		}
	}
	if entry == nil || body == nil || len(entry.Type.Params.List) != 2 || entry.Type.Results == nil || len(entry.Type.Results.List) != 1 || len(entry.Body.List) != 1 {
		t.Fatal("exact erased entry/typed helper body lost")
	}
	assertIdent(entry.Type.Params.List[1].Type, "any")
	assertIdent(entry.Type.Results.List[0].Type, "any")
	returned, ok := entry.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		t.Fatal("one bridge return required")
	}
	call, ok := returned.Results[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		t.Fatal("one original body execution required")
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != body.Name.Name {
		t.Fatal("bridge original body lost")
	}
	constructor, ok := selector.X.(*ast.CallExpr)
	if !ok || len(constructor.Args) != 1 {
		t.Fatal("one helper receiver required")
	}
	indexed, ok := constructor.Fun.(*ast.IndexListExpr)
	if !ok || len(indexed.Indices) != 2 {
		t.Fatal("helper T/physical-R slots lost")
	}
	assertIdent(indexed.X, "New"+identity.HelperName)
	assertIdent(indexed.Indices[0], "any")
	assertIdent(indexed.Indices[1], "any")
	assertIdent(constructor.Args[0], entry.Recv.List[0].Names[0].Name)
	assertIdent(call.Args[0], entry.Type.Params.List[0].Names[0].Name)
	assertIdent(call.Args[1], "value")
	if len(body.Type.Params.List) != 2 || body.Type.Results == nil || len(body.Type.Results.List) != 1 || len(body.Body.List) == 0 {
		t.Fatal("original typed implementation lost")
	}
	assertIdent(body.Type.Params.List[1].Type, "R")
	assertIdent(body.Type.Results.List[0].Type, "R")
	last, ok := body.Body.List[len(body.Body.List)-1].(*ast.ReturnStmt)
	if !ok || len(last.Results) != 1 {
		t.Fatal("identity return lost")
	}
	assertIdent(last.Results[0], "value")
}

func TestGenericsIntegration_InstanceGenericMethodHelper_WithBounds(t *testing.T) {
	src := `
package gen.integration4b;
public class Box<T extends Number> {
    public <R extends Comparable<T>> R identity(R value) { return value; }
    public static void call(Box<Integer> box, Integer value) { box.identity(value); }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "type BoxIdentityHelper[T stdjava.JavaNumber, R any] struct") {
		t.Fatalf("Expected helper struct with bounded type params, got:\n%s", out)
	}
	if !strings.Contains(flat, "func NewBoxIdentityHelper[T stdjava.JavaNumber, R any]") {
		t.Fatalf("Expected helper constructor to propagate bounds, got:\n%s", out)
	}
}

func TestGenericsIntegration_ExplicitTypeArgumentsOnGenericFunctionCall(t *testing.T) {
	src := `
package gen.integration5;
public class Utils {
    static <T> T id(T value) { return value; }

    public static void test() {
        Foo f = null;
        Foo g = Utils.<Foo>id(f);
    }
}
`
	h := generics472ParseWithPublicFoo(t, src)
	owner := h.File.Symbols.FindClassScope("Utils")
	if owner == nil || len(owner.TypeParameters) != 0 {
		t.Fatal("original nongeneric Utils owner lost")
	}
	identity := owner.FindMethodByName("id", nil)
	caller := owner.FindMethodByName("test", nil)
	if identity == nil || caller == nil || identity.DeclarationNode == nil || !identity.IsStatic || identity.IsPrivate || len(identity.TypeParameters) != 1 || len(identity.Parameters) != 1 || identity.OriginalType != "T" || identity.Parameters[0].OriginalType != "T" {
		t.Fatal("original generic static declaration and signature lost")
	}
	parameter := identity.TypeParameters[0]
	declaration := parameter.Declaration
	if declaration == nil || parameter.Name != "T" || len(parameter.Bounds) != 0 || definitionDeclaredJavaType(identity).TypeParameterBindings["T"] != declaration || identity.Parameters[0].TypeParameterBindings["T"] != declaration {
		t.Fatal("original return and argument must retain method-owned T")
	}
	declaring, known := genericMethodOwnerContext(identity)
	if !known || declaring.currentClass != owner || identity.DeclarationNode.ChildByFieldName("body").Content(h.File.Source) != "{ return value; }" {
		t.Fatal("original declaration owner and exact identity body lost")
	}
	ctx := h.Ctx.Clone()
	ctx.currentClass, ctx.localScope = owner, caller
	ctx.expectedType, ctx.expectedTypeRoot = "", nil
	invocation := findNode(caller.DeclarationNode, "method_invocation")
	if invocation == nil || invocation.ChildByFieldName("object") == nil || invocation.ChildByFieldName("object").Content(h.File.Source) != "Utils" || invocation.ChildByFieldName("name").Content(h.File.Source) != "id" {
		t.Fatal("original explicit class-qualified invocation lost")
	}
	explicit := invocation.ChildByFieldName("type_arguments")
	arguments := invocation.ChildByFieldName("arguments")
	if explicit == nil || explicit.NamedChildCount() != 1 || explicit.NamedChild(0).Content(h.File.Source) != "Foo" || arguments == nil || arguments.NamedChildCount() != 1 || arguments.NamedChild(0).Content(h.File.Source) != "f" {
		t.Fatal("original single explicit Foo slot and f argument lost")
	}
	if resolveClassScopeByIdentifier(ctx, h.File.Source, invocation.ChildByFieldName("object")) != owner {
		t.Fatal("qualifier must denote exact original Utils declaration")
	}
	resolved := findBestMethodInHierarchy(owner, "id", arguments, false, true, ctx, h.File.Source)
	if resolved == nil || resolved.owner != owner || resolved.def != identity {
		t.Fatal("call must select exact original static generic declaration")
	}
	bindings := genericJavaInvocationTypeBindings(identity, invocation, ctx, h.File.Source)
	if len(bindings) != 1 || bindings["T"] != "Foo" || ctx.expectedType != "" || ctx.expectedTypeRoot != nil {
		t.Fatalf("explicit logical T must be Foo without a consuming target: %v", bindings)
	}
	assertIdent := func(expr ast.Expr, name string) {
		t.Helper()
		id, ok := expr.(*ast.Ident)
		if !ok || id.Name != name {
			t.Fatalf("expected exact %s identifier, got %#v", name, expr)
		}
	}
	assertRuntime := func(expr ast.Expr, name string) {
		t.Helper()
		selector, ok := expr.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != name {
			t.Fatalf("expected runtime %s selector, got %#v", name, expr)
		}
		assertIdent(selector.X, "stdjava")
	}
	assertFoo := func(expr ast.Expr) {
		t.Helper()
		pointer, ok := expr.(*ast.StarExpr)
		if !ok {
			t.Fatalf("original logical Foo must be a pointer, got %T", expr)
		}
		assertIdent(pointer.X, "Foo")
	}
	logical := explicitTypeArgumentExprs(invocation, h.File.Source, inScopeTypeParameters(ctx), ctx)
	if len(logical) != 1 {
		t.Fatal("original explicit logical slot lost")
	}
	assertFoo(logical[0])
	inferred := inferMethodTypeArguments(identity, invocation, ctx, h.File.Source)
	if len(inferred) != 1 {
		t.Fatal("logical explicit binding lost")
	}
	assertFoo(inferred[0])
	plan := planStaticGenericMethodPhysicalABI(owner, identity, ctx)
	if plan == nil || plan.owner != owner || plan.method != identity || len(plan.methodDeclarations) != 1 || plan.methodDeclarations[0] != declaration || len(plan.parameterTypes) != 1 || len(plan.parameterJavaTypes) != 1 || plan.parameterJavaTypes[0] != "java.lang.Object" || plan.resultJavaType != "java.lang.Object" {
		t.Fatal("physical static entry must retain exact declaration-owned implicit Object slot")
	}
	assertIdent(plan.parameterTypes[0], "any")
	assertIdent(plan.resultType, "any")
	physical, javaTypes, known := genericMethodPhysicalTypeArguments(identity, ctx)
	if !known || len(physical) != 1 || len(javaTypes) != 1 || javaTypes[0] != "java.lang.Object" {
		t.Fatal("physical first-bound erasure lost")
	}
	assertIdent(physical[0], "any")
	copied := *identity
	if planStaticGenericMethodPhysicalABI(owner, &copied, ctx) != nil {
		t.Fatal("copied method must not acquire source-owned plan")
	}
	checkConsumer := func(expr ast.Expr, token string) {
		t.Helper()
		view, ok := expr.(*ast.CallExpr)
		if !ok || len(view.Args) != 2 {
			t.Fatal("expected exactly one typed result view")
		}
		indexed, ok := view.Fun.(*ast.IndexExpr)
		if !ok {
			t.Fatal("typed consumer lost one exact argument")
		}
		assertRuntime(indexed.X, "ObjectView")
		assertFoo(indexed.Index)
		// The supplemental real Foo declaration has the qualified identity
		// independently observed from Foo.class.getName() on JDK21.
		if reflectCoreTypeID(t, view.Args[1]) != "gen.integration5.Foo" {
			t.Fatal("original logical Foo descriptor lost")
		}
		call, ok := view.Args[0].(*ast.CallExpr)
		if !ok || len(call.Args) != 2 {
			t.Fatal("view must consume exactly one original invocation")
		}
		entry, ok := call.Fun.(*ast.IndexExpr)
		if !ok {
			t.Fatal("call lost single physical method slot")
		}
		assertIdent(entry.X, executionImplementationName(identity, owner, ctx))
		assertIdent(entry.Index, "any")
		assertIdent(call.Args[0], token)
		assertIdent(call.Args[1], "f")
	}
	ctx.executionContextName = "callerExecution"
	checkConsumer(ParseExpr(invocation, h.File.Source, ctx), "callerExecution")
	file, ok := ParseNode(h.File.Ast, h.File.Source, h.Ctx).(*ast.File)
	if !ok {
		t.Fatal("original source must emit one file")
	}
	var implementation, consumer *ast.FuncDecl
	for _, member := range file.Decls {
		fn, ok := member.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		if fn.Name.Name == executionImplementationName(identity, owner, h.Ctx) {
			if implementation != nil {
				t.Fatal("duplicate identity implementation")
			}
			implementation = fn
		}
		if fn.Name.Name == executionImplementationName(caller, owner, h.Ctx) {
			if consumer != nil {
				t.Fatal("duplicate consumer implementation")
			}
			consumer = fn
		}
	}
	if implementation == nil || implementation.Type.TypeParams == nil || len(implementation.Type.TypeParams.List) != 1 || len(implementation.Type.TypeParams.List[0].Names) != 1 || len(implementation.Type.Params.List) != 2 || implementation.Type.Results == nil || len(implementation.Type.Results.List) != 1 || len(implementation.Body.List) != 1 {
		t.Fatal("original generic identity signature/body lost")
	}
	assertIdent(implementation.Type.TypeParams.List[0].Names[0], parameter.EmittedName())
	assertIdent(implementation.Type.TypeParams.List[0].Type, "any")
	assertIdent(implementation.Type.Params.List[1].Type, parameter.EmittedName())
	assertIdent(implementation.Type.Results.List[0].Type, parameter.EmittedName())
	returned, ok := implementation.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		t.Fatal("original one-return body lost")
	}
	assertIdent(returned.Results[0], "value")
	if consumer == nil {
		t.Fatal("original emitted caller lost")
	}
	count := 0
	ast.Inspect(consumer.Body, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return true
		}
		lhs, ok := assign.Lhs[0].(*ast.Ident)
		if ok && lhs.Name == "g" {
			checkConsumer(assign.Rhs[0], "__java2goExecution")
			count++
		}
		return true
	})
	if count != 1 {
		t.Fatalf("expected exactly one original g consumer, got %d", count)
	}
}

func TestGenericsIntegration_StaticMethodWithBounds(t *testing.T) {
	src := `
package gen.integration5b;
public class Utils {
    static <T extends Number> T id(T value) { return value; }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "func id[T stdjava.JavaNumber]") {
		t.Fatalf("Expected bounded type parameter on static method, got:\n%s", out)
	}
}

func TestGenericsIntegration_NestedClassTypeParameters(t *testing.T) {
	src := `
package gen.integration6;
public class Outer<T> {
    public class Inner<U> {
        T t;
        U u;
    }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "type Outer[T any] struct") {
		t.Errorf("Expected Outer to be generic, got:\n%s", out)
	}
	if !strings.Contains(out, "type OuterInner[T any, U any] struct") {
		t.Errorf("Expected Inner to inherit parent type params and add its own, got:\n%s", out)
	}
}

func TestGenericsIntegration_GenericInterface(t *testing.T) {
	src := `
package gen.interfaces;
public interface Box<T> {
    T get();
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "type Box[T any] interface") {
		t.Fatalf("Expected generic interface with type parameter, got:\n%s", out)
	}
	if !strings.Contains(flat, "Get() T") {
		t.Fatalf("Expected method to use the type parameter T, got:\n%s", out)
	}
}
