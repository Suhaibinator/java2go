package transpiler

import (
	"go/ast"
	"reflect"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

type deferredToCollectionProof struct {
	result, element inferredJavaTypeOrigin
	factory         *sitter.Node
	resultJava      string
}

// Recompute declaration facts from this actual attached occurrence. This is
// neither a nominal Collector object nor a retained per-graph pointer registry.
func deferredToCollectionCertificate(collector *sitter.Node, ctx Ctx, source []byte) (deferredToCollectionProof, bool) {
	var empty deferredToCollectionProof
	if !sourceArrayListNativeSourceContext(collector, ctx, source) || invocationArgumentCount(collector) != 1 {
		return empty, false
	}
	if name, canonical := deferredToCollectionFactoryName(collector, source, ctx); !canonical || name != "toCollection" {
		return empty, false
	}
	arguments := collector.Parent()
	if arguments == nil || arguments.Type() != "argument_list" || arguments.NamedChildCount() != 1 || !arguments.NamedChild(0).Equal(collector) {
		return empty, false
	}
	collect := arguments.Parent()
	if collect == nil || collect.Type() != "method_invocation" || collect.ChildByFieldName("name") == nil || collect.ChildByFieldName("name").Content(source) != "collect" || invocationArgumentCount(collect) != 1 {
		return empty, false
	}
	// Downstream typed-R composition needs its own physical result plan. The
	// initial erased supplied-collection result belongs to a direct terminal.
	receiver := collect.ChildByFieldName("object")
	if selected, known := intrinsicMethodReceiverTypeName(receiver, "collect", ctx, source); !known || selected != "Stream" || instanceNodeIntrinsics[intrinsicKey{"Stream", "collect"}] == nil {
		return empty, false
	}
	element, known := deferredToCollectionReceiverElement(receiver, ctx, source)
	if !known {
		return empty, false
	}
	factory := invocationArgumentNode(collector, 0)
	result, known := deferredToCollectionSupplierResult(factory, ctx, source)
	if !known {
		return empty, false
	}
	slot, known := deferredToCollectionElement(result, ctx)
	if !known || !sameJavaTypeOrigin(slot, element) {
		return empty, false
	}
	java, bindings, known := sourceDiamondOriginEdge(result)
	if !known {
		return empty, false
	}
	for _, declaration := range bindings {
		found := false
		for _, parameter := range visibleTypeParameterDeclarations(ctx) {
			found = found || parameter.Declaration == declaration
		}
		if !found {
			return empty, false
		}
	}
	// SAM parsing receives a spelling only after its independently retained
	// origin agrees in this lexical namespace. A caller homonym cannot rebind C.
	roundTrip := declaredJavaTypeOrigin(symbol.JavaType{Original: java, TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}, ctx)
	if !sameJavaTypeOrigin(result, roundTrip) {
		return empty, false
	}
	return deferredToCollectionProof{result: result, element: element, factory: factory, resultJava: java}, true
}

func deferredToCollectionReceiverElement(receiver *sitter.Node, ctx Ctx, source []byte) (inferredJavaTypeOrigin, bool) {
	receiver = unwrapParenthesizedExpressionNode(receiver)
	if receiver == nil {
		return inferredJavaTypeOrigin{}, false
	}
	// The same owned written element drives Stream.of's physical Go T and
	// its receiver origin; operand LUB and an outer target cannot replace it.
	if element, _, known := explicitStreamFactoryElement(receiver, ctx, source); known {
		return element, true
	}

	if !sourceArrayListNativeScalarExpression(receiver, ctx, source) {
		return inferredJavaTypeOrigin{}, false
	}
	var origin inferredJavaTypeOrigin
	_, known := inferExprJavaType(receiver, ctx, source, &origin)
	if !known || origin.unresolved || origin.wildcard || origin.arrayRank != 0 || origin.parameter != nil || origin.nominalScope != nil || origin.externalName != "java.util.stream.Stream" || len(origin.arguments) != 1 {
		return inferredJavaTypeOrigin{}, false
	}
	return origin.arguments[0], sourceArrayListNativeScalarOrigin(origin.arguments[0])
}

func deferredToCollectionSupplierResult(factory *sitter.Node, ctx Ctx, source []byte) (inferredJavaTypeOrigin, bool) {
	factory = unwrapParenthesizedExpressionNode(factory)
	if factory == nil {
		return inferredJavaTypeOrigin{}, false
	}
	if factory.Type() == "method_reference" {
		if factory.NamedChildCount() != 1 || !strings.HasSuffix(strings.TrimSpace(factory.Content(source)), "::new") {
			return inferredJavaTypeOrigin{}, false
		}
		target := factory.NamedChild(0)
		origin := declaredJavaTypeOrigin(symbol.JavaType{Original: target.Content(source), TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}, ctx)
		if origin.externalName == "java.util.List" {
			return inferredJavaTypeOrigin{}, false
		}
		_, known := deferredToCollectionElement(origin, ctx)
		return origin, known
	}
	if factory.Type() == "lambda_expression" {
		parameters := factory.ChildByFieldName("parameters")
		if parameters == nil || parameters.NamedChildCount() != 0 {
			return inferredJavaTypeOrigin{}, false
		}
		body := factory.ChildByFieldName("body")
		if body != nil && body.Type() == "block" {
			body = singleReturnedExpression(body)
		}
		if body == nil || body.Type() != "object_creation_expression" || objectCreationClassBody(body) != nil {
			return inferredJavaTypeOrigin{}, false
		}
		var origin inferredJavaTypeOrigin
		_, known := inferExprJavaType(body, ctx, source, &origin)
		if origin.externalName == "java.util.List" {
			return inferredJavaTypeOrigin{}, false
		}
		_, admitted := deferredToCollectionElement(origin, ctx)
		return origin, known && admitted
	}
	if !sourceArrayListNativeScalarExpression(factory, ctx, source) {
		return inferredJavaTypeOrigin{}, false
	}
	var supplier inferredJavaTypeOrigin
	_, known := inferExprJavaType(factory, ctx, source, &supplier)
	if !known || supplier.unresolved || supplier.wildcard || supplier.arrayRank != 0 || supplier.parameter != nil {
		return inferredJavaTypeOrigin{}, false
	}
	if supplier.nominalScope == nil && supplier.externalName == "java.util.function.Supplier" && len(supplier.arguments) == 1 {
		return supplier.arguments[0], true
	}
	if supplier.nominalScope == nil {
		return inferredJavaTypeOrigin{}, false
	}
	for _, contract := range sourceNativeFunctionalContracts(supplier.nominalScope, ctx) {
		if contract.family != "Supplier" {
			continue
		}
		method, owner, _ := resolveSourceNativeSAM(supplier.nominalScope, contract, ctx)
		if method == nil || !supplierSourceSelectedMethodOwned(supplier.nominalScope, owner, method, contract, ctx) {
			continue
		}
		declaring, attached := supplierSourceDeclarationCtx(supplier.nominalScope, ctx)
		if !attached {
			return inferredJavaTypeOrigin{}, false
		}
		origin := sourceNativeFunctionalResultOrigin(supplier.nominalScope, contract.result, declaring)
		bindings := sourceClassOriginBindings(supplier.nominalScope, supplier.arguments, supplier.nominalScope, ctx)
		return substituteJavaTypeOrigin(origin, bindings), true
	}
	return inferredJavaTypeOrigin{}, false
}

func deferredToCollectionElement(result inferredJavaTypeOrigin, ctx Ctx) (inferredJavaTypeOrigin, bool) {
	if sourceArrayListNativeRole(result) {
		if len(result.arguments) == 1 {
			return result.arguments[0], sourceArrayListNativeScalarOrigin(result.arguments[0])
		}
		return inferredJavaTypeOrigin{externalName: "java.lang.Object", javaType: "java.lang.Object"}, true
	}
	if result.nominalScope == nil || result.nominalScope.IsAbstract || result.nominalScope.IsInterface {
		return inferredJavaTypeOrigin{}, false
	}
	return sourceArrayListNativeElement(result, ctx)
}

func deferredToCollectionResultOrigin(collector *sitter.Node, ctx Ctx, source []byte) (inferredJavaTypeOrigin, string, bool) {
	proof, known := deferredToCollectionCertificate(collector, ctx, source)
	return proof.result, proof.resultJava, known
}

func lowerDeferredToCollectionPlan(collector *sitter.Node, ctx Ctx, source []byte) (ast.Expr, string) {
	proof, known := deferredToCollectionCertificate(collector, ctx, source)
	if !known {
		return nil, ""
	}
	element, known := sourceArrayListNativeElementType(proof.element, ctx)
	if !known {
		return nil, ""
	}
	factory := parseDeferredCallable(proof.factory, "java.util.function.Supplier<"+proof.resultJava+">", nil, proof.resultJava, ctx, source)
	if factory == nil {
		return nil, ""
	}
	view := stdjavaCall(ctx, "SupplierReferenceView", factory)
	return stdjavaGenericCall(ctx, "CollectorToCollectionPlan", []ast.Expr{element}, []ast.Expr{view}), proof.resultJava
}

func deferredToCollectionResultType(origin inferredJavaTypeOrigin, ctx Ctx) (ast.Expr, bool) {
	if sourceArrayListObjectOrigin(origin) {
		return ast.NewIdent("any"), true
	}
	element, known := deferredToCollectionElement(origin, ctx)
	if !known {
		return nil, false
	}
	physical, known := sourceArrayListNativeElementType(element, ctx)
	if !known {
		return nil, false
	}
	if sourceArrayListNativeRole(origin) {
		return &ast.StarExpr{X: applyTypeArguments(stdjavaQualifiedExpr("List", ctx), []ast.Expr{physical})}, true
	}
	owner := origin.nominalScope
	if owner == nil || owner.Class == nil || len(owner.TypeParameters) != len(origin.arguments) {
		return nil, false
	}
	var args []ast.Expr
	for _, argument := range origin.arguments {
		item, known := sourceArrayListNativeElementType(argument, ctx)
		if !known {
			return nil, false
		}
		args = append(args, item)
	}
	return &ast.StarExpr{X: applyTypeArguments(qualifiedNameExpr(owner.Class.Name, findJavaPackageForClassScope(owner), ctx), args)}, true
}

// Match the actual runtime ABI, not a same-spelled call or an arbitrary any
// closure. The Java occurrence, selected recipe, T and operation E all agree.
func deferredToCollectionRawResult(value ast.Expr, proof deferredToCollectionProof, ctx Ctx) bool {
	call, ok := value.(*ast.CallExpr)
	if !ok || call.Ellipsis.IsValid() || len(call.Args) != 3 || !reflect.DeepEqual(call.Fun, stdjavaQualifiedExpr("StreamCollectPlanExecution", ctx)) || !reflect.DeepEqual(call.Args[0], intrinsicExecutionExpr(ctx)) {
		return false
	}
	plan, ok := call.Args[2].(*ast.CallExpr)
	if !ok || plan.Ellipsis.IsValid() || len(plan.Args) != 1 {
		return false
	}
	typed, ok := plan.Fun.(*ast.IndexExpr)
	element, known := sourceArrayListNativeElementType(proof.element, ctx)
	if !ok || !known || !reflect.DeepEqual(typed.X, stdjavaQualifiedExpr("CollectorToCollectionPlan", ctx)) || !reflect.DeepEqual(typed.Index, element) {
		return false
	}
	supplier, ok := plan.Args[0].(*ast.CallExpr)
	return ok && !supplier.Ellipsis.IsValid() && len(supplier.Args) == 1 && reflect.DeepEqual(supplier.Fun, stdjavaQualifiedExpr("SupplierReferenceView", ctx))
}

func deferredToCollectionBoundary(value ast.Expr, node *sitter.Node, expected string, ctx Ctx, source []byte) (ast.Expr, bool) {
	if node == nil || node.Type() != "method_invocation" || strings.TrimSpace(expected) == "" || invocationArgumentCount(node) != 1 {
		return nil, false
	}
	proof, known := deferredToCollectionCertificate(invocationArgumentNode(node, 0), ctx, source)
	if !known || !deferredToCollectionRawResult(value, proof, ctx) {
		return nil, false
	}
	target, known := sourceArrayListNativeBoundaryOrigin(node, ctx, source)
	if !known {
		return nil, false
	}
	if sourceArrayListObjectOrigin(target) {
		return value, true
	}
	assignable := sameJavaTypeOrigin(proof.result, target)
	if sourceArrayListNativeRole(target) {
		actual, actualKnown := deferredToCollectionElement(proof.result, ctx)
		wanted, targetKnown := deferredToCollectionElement(target, ctx)
		assignable = actualKnown && targetKnown && sameJavaTypeOrigin(actual, wanted)
	}
	if !assignable {
		return nil, false
	}
	physical, known := deferredToCollectionResultType(target, ctx)
	descriptor, described := declaredOriginDescriptor(target, ctx, map[*symbol.TypeParamDeclaration]bool{})
	if !known || !described {
		return nil, false
	}
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{physical}, []ast.Expr{value, descriptor}), true
}

// Only an actual direct canonical element-list factory may become a pipeline
// source here. Existing pipelines and legacy stream values keep their behavior.
func deferredToCollectionFactorySource(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) (ast.Expr, bool) {
	if !sourceArrayListNativeSourceContext(invocation, ctx, source) || invocation.Type() != "method_invocation" || invocationArgumentCount(invocation) != 1 || !sourceArrayListConstructorFileNamespace(ctx.currentFile) {
		return nil, false
	}
	proofCtx := ctx.Clone()
	proofCtx.memberTypeHeaderOwner = nil
	proofCtx.localClasses = nil
	proofCtx.anonymousClasses = nil
	proof, known := deferredToCollectionCertificate(invocationArgumentNode(invocation, 0), proofCtx, source)
	if !known {
		return nil, false
	}
	receiver := invocation.ChildByFieldName("object")
	element, _, known := explicitStreamFactoryElement(receiver, proofCtx, source)
	count := invocationArgumentCount(receiver)
	if !known || count == 1 || !sameJavaTypeOrigin(element, proof.element) {
		return nil, false
	}
	physical, known := sourceArrayListNativeElementType(element, ctx)
	if !known {
		return nil, false
	}
	replace := func(value ast.Expr) (ast.Expr, bool) {
		call, ok := value.(*ast.CallExpr)
		if !ok || call.Ellipsis.IsValid() || len(call.Args) != count {
			return nil, false
		}
		typed, ok := call.Fun.(*ast.IndexExpr)
		if !ok || !reflect.DeepEqual(typed.X, stdjavaQualifiedExpr("NewStream", ctx)) || !reflect.DeepEqual(typed.Index, physical) {
			return nil, false
		}
		replacement := *call
		replacement.Fun = &ast.IndexExpr{X: stdjavaQualifiedExpr("StreamFromValues", ctx), Index: typed.Index}
		return &replacement, true
	}
	if replacement, ok := replace(recv); ok {
		return replacement, true
	}
	return nil, false
}
