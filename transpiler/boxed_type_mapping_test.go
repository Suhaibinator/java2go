package transpiler

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

func TestBoxedTypeMapping_ReferenceABI(t *testing.T) {
	helper := setupParseHelper(t, "class WrapperTypes {}")
	cases := map[string]string{
		"Boolean": "*stdjava.Boolean", "Byte": "*stdjava.Byte",
		"Short": "*stdjava.Short", "Character": "*stdjava.Character",
		"Integer": "*stdjava.Integer", "Long": "*stdjava.Long",
		"Float": "*stdjava.Float", "Double": "*stdjava.Double",
		"java.lang.Integer": "*stdjava.Integer", "Number": "stdjava.JavaNumber",
		"Comparable<Integer>": "any", "Comparable": "any",
		"int": "int32", "long": "int64", "char": "rune", "boolean": "bool",
	}
	for javaType, want := range cases {
		t.Run(javaType, func(t *testing.T) {
			var got bytes.Buffer
			if err := printer.Fprint(&got, token.NewFileSet(), javaTypeStringToGoTypeExpr(javaType, nil, helper.Ctx)); err != nil {
				t.Fatal(err)
			}
			if got.String() != want {
				t.Fatalf("mapped type = %s, want %s", got.String(), want)
			}
		})
	}
}

func TestBoxedTypeMapping_SourceDeclarationShadowsImplicitImport(t *testing.T) {
	out := normalizeSpaces(renderGoFileFromJava(t, `
class Integer {}
class Number {}
interface Comparable<T> {}
class ShadowedWrappers {
    Integer source;
    java.lang.Integer builtin;
    Number sourceNumber;
    java.lang.Number builtinNumber;
    Comparable<Integer> sourceComparable;
    java.lang.Comparable<java.lang.Integer> builtinComparable;
}`))
	for _, want := range []string{
		"source *integer", "builtin *stdjava.Integer", "sourceNumber *number",
		"builtinNumber stdjava.JavaNumber", "sourceComparable comparable[*integer]", "builtinComparable any",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in generated declarations:\n%s", want, out)
		}
	}
}

func TestBoxedTypeInference_NominalHierarchy(t *testing.T) {
	helper := setupParseHelper(t, "class WrapperInference {}")
	for _, test := range []struct {
		actual, expected string
		assignable       bool
	}{
		{"Integer", "Number", true}, {"Double", "Number", true},
		{"Character", "Number", false}, {"Boolean", "Number", false},
		{"Integer", "Object", true}, {"Number", "Object", true},
		{"Integer", "Comparable<Integer>", true}, {"Integer", "Comparable<Long>", false},
		{"Integer", "Comparable<? extends Number>", true}, {"Integer", "Comparable<? super Integer>", true},
		{"String", "Comparable<String>", true}, {"String", "Comparable<Integer>", false},
		{"Integer[]", "Number[]", true}, {"Integer[]", "int[]", false},
	} {
		if got := javaInferenceTypeAssignable(test.actual, test.expected, helper.Ctx); got != test.assignable {
			t.Errorf("assignable(%s, %s) = %v, want %v", test.actual, test.expected, got, test.assignable)
		}
	}
	if got := javaInferenceLeastUpperBound([]string{"Integer", "Long", "Double"}, helper.Ctx); got != "java.lang.Number" {
		t.Errorf("numeric wrapper LUB = %q, want java.lang.Number", got)
	}
	if got := javaInferenceLeastUpperBound([]string{"Integer", "Character"}, helper.Ctx); got != "Object" {
		t.Errorf("numeric and character LUB = %q, want Object", got)
	}
}

func TestBoxedTypeInference_PrimitiveInvocationUsesWrapperTypeArgument(t *testing.T) {
	src := `
class BoxedInference {
    static <T> T identity(T value) { return value; }
    static Integer run() { return identity(17); }
}`
	h := setupParseHelper(t, src)
	owner := h.File.Symbols.FindClassScope("BoxedInference")
	if owner == nil || len(owner.TypeParameters) != 0 {
		t.Fatal("original nongeneric owner lost")
	}
	identity := owner.FindMethodByName("identity", nil)
	caller := owner.FindMethodByName("run", nil)
	if identity == nil || caller == nil || identity.DeclarationNode == nil || !identity.IsStatic || identity.IsPrivate || len(identity.TypeParameters) != 1 || len(identity.Parameters) != 1 || identity.OriginalType != "T" || identity.Parameters[0].OriginalType != "T" {
		t.Fatal("original static identity declaration/signature lost")
	}
	typeParameter := identity.TypeParameters[0]
	declaration := typeParameter.Declaration
	if declaration == nil || typeParameter.Name != "T" || len(typeParameter.Bounds) != 0 || definitionDeclaredJavaType(identity).TypeParameterBindings["T"] != declaration || identity.Parameters[0].TypeParameterBindings["T"] != declaration {
		t.Fatal("return and parameter must bind the original method-owned T")
	}
	declaring, known := genericMethodOwnerContext(identity)
	if !known || declaring.currentClass != owner || identity.DeclarationNode.ChildByFieldName("body").Content(h.File.Source) != "{ return value; }" {
		t.Fatal("exact source owner/body changed")
	}
	ctx := h.Ctx.Clone()
	ctx.currentClass, ctx.localScope = owner, caller
	ctx.expectedType, ctx.expectedTypeRoot = "", nil
	invocation := findNode(caller.DeclarationNode, "method_invocation")
	if invocation == nil || invocation.ChildByFieldName("object") != nil || invocation.ChildByFieldName("name").Content(h.File.Source) != "identity" {
		t.Fatal("original unqualified invocation changed")
	}
	arguments := invocation.ChildByFieldName("arguments")
	if arguments == nil || arguments.NamedChildCount() != 1 || arguments.NamedChild(0).Content(h.File.Source) != "17" {
		t.Fatal("original single primitive argument changed")
	}
	resolved := findBestMethodInHierarchy(owner, "identity", arguments, false, true, ctx, h.File.Source)
	if resolved == nil || resolved.owner != owner || resolved.def != identity {
		t.Fatal("invocation must select the exact original static declaration")
	}
	actual, known := inferExprJavaType(arguments.NamedChild(0), ctx, h.File.Source)
	if !known || actual != "int" {
		t.Fatalf("original argument is not primitive int: %q", actual)
	}
	bindings := genericJavaInvocationTypeBindings(identity, invocation, ctx, h.File.Source)
	if len(bindings) != 1 || bindings["T"] != "java.lang.Integer" {
		t.Fatalf("argument-only logical T must infer the canonical boxed Integer: %v", bindings)
	}
	assertIdent := func(expr ast.Expr, name string) {
		t.Helper()
		ident, ok := expr.(*ast.Ident)
		if !ok || ident.Name != name {
			t.Fatalf("expected exact identifier %s, got %#v", name, expr)
		}
	}
	assertRuntime := func(expr ast.Expr, name string) {
		t.Helper()
		selector, ok := expr.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != name {
			t.Fatalf("expected runtime selector %s, got %#v", name, expr)
		}
		assertIdent(selector.X, "stdjava")
	}
	assertInteger := func(expr ast.Expr) {
		t.Helper()
		pointer, ok := expr.(*ast.StarExpr)
		if !ok {
			t.Fatalf("Integer must retain its object pointer, got %T", expr)
		}
		assertRuntime(pointer.X, "Integer")
	}
	logical := inferMethodTypeArguments(identity, invocation, ctx, h.File.Source)
	if len(logical) != 1 {
		t.Fatal("logical inference lost the one original method slot")
	}
	assertInteger(logical[0])
	if primitive, builtin := builtinJavaWrapperPrimitive("Integer", ctx); !builtin || primitive != "int" || resolveClassScopeByQualifiedName(ctx, "Integer") != nil {
		t.Fatal("original implicit Integer must resolve the builtin owner without a source shadow")
	}
	descriptor, known := javaTypeDescriptorExpr(bindings["T"], ctx)
	if !known || reflectCoreTypeID(t, descriptor) != "java.lang.Integer" {
		t.Fatal("logical boxed consumer lost its canonical nominal descriptor")
	}
	plan := planStaticGenericMethodPhysicalABI(owner, identity, ctx)
	if plan == nil || plan.owner != owner || plan.method != identity || len(plan.methodDeclarations) != 1 || plan.methodDeclarations[0] != declaration || len(plan.parameterJavaTypes) != 1 || plan.parameterJavaTypes[0] != "java.lang.Object" || plan.resultJavaType != "java.lang.Object" || len(plan.parameterTypes) != 1 {
		t.Fatal("static physical plan must prove the same method-owned implicit Object slot")
	}
	assertIdent(plan.parameterTypes[0], "any")
	assertIdent(plan.resultType, "any")
	physical, javaTypes, known := genericMethodPhysicalTypeArguments(identity, ctx)
	if !known || len(physical) != 1 || len(javaTypes) != 1 || javaTypes[0] != "java.lang.Object" {
		t.Fatal("physical method slot lost its declared first-bound erasure")
	}
	assertIdent(physical[0], "any")
	copied := *identity
	if planStaticGenericMethodPhysicalABI(owner, &copied, ctx) != nil {
		t.Fatal("a copied method must not acquire source-owned entry eligibility")
	}
	checkConsumer := func(expr ast.Expr, executionName string) {
		t.Helper()
		view, ok := expr.(*ast.CallExpr)
		if !ok || len(view.Args) != 2 {
			t.Fatal("expected one nominal consumer projection")
		}
		indexed, ok := view.Fun.(*ast.IndexExpr)
		if !ok {
			t.Fatal("consumer lost the exact Integer type argument")
		}
		assertRuntime(indexed.X, "ObjectView")
		assertInteger(indexed.Index)
		if reflectCoreTypeID(t, view.Args[1]) != "java.lang.Integer" {
			t.Fatal("consumer lost canonical Integer ClassID")
		}
		call, ok := view.Args[0].(*ast.CallExpr)
		if !ok || len(call.Args) != 2 {
			t.Fatal("consumer must enclose exactly one original invocation")
		}
		entry, ok := call.Fun.(*ast.IndexExpr)
		if !ok {
			t.Fatal("static invocation lost its one physical slot")
		}
		assertIdent(entry.X, executionImplementationName(identity, owner, ctx))
		assertIdent(entry.Index, "any")
		assertIdent(call.Args[0], executionName)
		box, ok := call.Args[1].(*ast.CallExpr)
		if !ok || len(box.Args) != 1 {
			t.Fatal("single argument must create one canonical boxed object")
		}
		assertRuntime(box.Fun, "BoxInteger")
		conversion, ok := box.Args[0].(*ast.CallExpr)
		if !ok || len(conversion.Args) != 1 {
			t.Fatal("boxed argument lost Java primitive width")
		}
		assertIdent(conversion.Fun, "int32")
		assertIdent(conversion.Args[0], "17")
	}
	ctx.executionContextName = "callerExecution"
	checkConsumer(ParseExpr(invocation, h.File.Source, ctx), "callerExecution")
	file, ok := ParseNode(h.File.Ast, h.File.Source, h.Ctx).(*ast.File)
	if !ok {
		t.Fatal("original Java no longer emits a Go file")
	}
	var implementation, run *ast.FuncDecl
	for _, member := range file.Decls {
		function, ok := member.(*ast.FuncDecl)
		if !ok || function.Recv != nil {
			continue
		}
		if function.Name.Name == executionImplementationName(identity, owner, h.Ctx) {
			if implementation != nil { t.Fatal("duplicate original identity implementation") }
			implementation = function
		}
		if function.Name.Name == executionImplementationName(caller, owner, h.Ctx) {
			if run != nil { t.Fatal("duplicate original typed consumer") }
			run = function
		}
	}
	if implementation == nil || implementation.Type.TypeParams == nil || len(implementation.Type.TypeParams.List) != 1 || len(implementation.Type.Params.List) != 2 || implementation.Type.Results == nil || len(implementation.Type.Results.List) != 1 || len(implementation.Body.List) != 1 {
		t.Fatal("original typed static method signature/body lost")
	}
	assertIdent(implementation.Type.TypeParams.List[0].Type, "any")
	assertIdent(implementation.Type.Params.List[1].Type, typeParameter.EmittedName())
	assertIdent(implementation.Type.Results.List[0].Type, typeParameter.EmittedName())
	body, ok := implementation.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(body.Results) != 1 { t.Fatal("identity body lost its one return") }
	assertIdent(body.Results[0], identity.Parameters[0].Name)
	if run == nil || len(run.Type.Params.List) != 1 || run.Type.Results == nil || len(run.Type.Results.List) != 1 || len(run.Body.List) != 1 || len(run.Type.Params.List[0].Names) != 1 {
		t.Fatal("typed caller signature/body changed")
	}
	assertInteger(run.Type.Results.List[0].Type)
	execution, ok := run.Type.Params.List[0].Type.(*ast.StarExpr)
	if !ok { t.Fatal("caller Execution lost pointer") }
	assertRuntime(execution.X, "Execution")
	returned, ok := run.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 { t.Fatal("typed caller lost one return") }
	checkConsumer(returned.Results[0], run.Type.Params.List[0].Names[0].Name)
}

func TestBoxedTypeInference_IdentityNullAndNumberBounds(t *testing.T) {
	assertGenericReferenceEqualityResult(t, `
public class BoxedGenericObjects {
    static <T> T identity(T value) { return value; }
    static <T extends Number> T choose(boolean first, T left, T right) {
        return first ? left : right;
    }
    static <N extends Number> Number mixedBound(N value) {
        return choose(false, 1, value);
    }
    public static int run() {
        var inferred = identity(17);
        Integer exact = inferred;
        Integer missing = identity(null);
        Integer integer = Integer.valueOf(3);
        Long longer = Long.valueOf(4);
        Number number = choose(false, integer, longer);
        int score = inferred == exact ? 1 : 0;
        if (identity(integer) == integer) score += 2;
        if (missing == null) score += 4;
        if (number == longer) score += 8;
        if (mixedBound(longer) == longer) score += 16;
        score += identity(5);
        return score;
    }
}
`, 36)
}

func TestBoxedTypeMapping_QualifiedWrapperBesideSourceInteger(t *testing.T) {
	assertGenericReferenceEqualityResult(t, `
class Integer {
    int value;
    Integer(int value) { this.value = value; }
}
public class ShadowedIntegerObjects {
    Integer source;
    java.lang.Integer builtin;
    static <T> T identity(T value) { return value; }
    public static int run() {
        ShadowedIntegerObjects holder = new ShadowedIntegerObjects();
        holder.source = new Integer(3);
        holder.builtin = identity(4);
        return holder.source.value + holder.builtin;
    }
}
`, 7)
}

func TestBoxedTypeInference_InstanceOwnerAndMethodParameters(t *testing.T) {
	assertGenericReferenceEqualityResult(t, `
public class BoxedInstanceParameters {
    static class Holder<U> {
        U value;
        void set(U value) { this.value = value; }
        <T> T pair(U owner, T value) { this.value = owner; return value; }
        <T> T identity(T value) { return value; }
    }
    static class IntegerHolder extends Holder<Integer> {}
    static <T> int call(Holder<T> holder, T owner) {
        Integer boxed = holder.pair(owner, 7);
        return boxed;
    }
    public static int run() {
        Holder<Integer> holder = new Holder<>();
        holder.set(3 * 3);
        Integer result = holder.identity(-1);
        int mixed = call(holder, holder.value);
        IntegerHolder inherited = new IntegerHolder();
        inherited.set(4);
        return holder.value + result + mixed + inherited.value;
    }
}
`, 19)
}
