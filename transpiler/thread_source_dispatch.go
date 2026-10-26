package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/token"
)

type threadSourceInvocationPlan struct {
	receiverScope  *symbol.ClassScope
	dispatchScope  *symbol.ClassScope
	selected       *methodResolution
	super          bool
	classQualifier bool
	name           string
}

func threadSourceStaticMethod(name string) bool {
	return name == "interrupted" || name == "currentThread" || name == "onSpinWait"
}
func threadSourceClassQualifier(object *sitter.Node, ctx Ctx, source []byte) *symbol.ClassScope {
	if object == nil {
		return nil
	}
	root := object
	for root.Type() == "field_access" || root.Type() == "scoped_identifier" {
		next := root.ChildByFieldName("object")
		if next == nil {
			next = root.ChildByFieldName("scope")
		}
		if next == nil && root.NamedChildCount() > 0 {
			next = root.NamedChild(0)
		}
		if next == nil {
			break
		}
		root = next
	}
	if root.Type() != "identifier" {
		return nil
	}
	if _, bound := resolveReferenceTypeParameter(symbol.JavaType{Original: root.Content(source)}, ctx); bound {
		return nil
	}
	if _, value := inferIdentifierJavaType(root.Content(source), ctx); value {
		return nil
	}
	scope := resolveClassScopeByQualifiedName(ctx, object.Content(source))
	if threadSourceRoot(scope, ctx) != nil {
		return scope
	}
	return nil
}

// Selection is pure: it never parses/emits the receiver, evaluates arguments or
// borrows expectedType. Metadata and emission consume the same Java declaration.
func threadSourceSelectInvocation(node *sitter.Node, source []byte, ctx Ctx) *threadSourceInvocationPlan {
	if node == nil || node.Type() != "method_invocation" {
		return nil
	}
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := nameNode.Content(source)
	switch name {
	case "interrupt", "isInterrupted", "run", "start", "join", "isAlive", "getName", "interrupted", "currentThread", "onSpinWait":
	default:
		return nil
	}
	object := node.ChildByFieldName("object")
	args := node.ChildByFieldName("arguments")
	count := nodeutil.SemanticNamedChildCount(args)
	if name != "join" && count != 0 || name == "join" && count > 2 {
		return nil
	}
	p := &threadSourceInvocationPlan{name: name, super: object != nil && object.Type() == "super"}
	if object == nil || p.super {
		if ctx.currentClass == nil || ctx.localScope == nil || !extendsBuiltinThread(ctx) || ctx.localScope.IsStatic && !threadSourceStaticMethod(name) {
			return nil
		}
		p.receiverScope = ctx.currentClass
		p.dispatchScope = ctx.currentClass
		if p.super {
			p.dispatchScope = resolveSuperclassScopeInDeclaringContext(ctx, ctx.currentClass)
		}
	} else if scope := threadSourceClassQualifier(object, ctx, source); scope != nil {
		if !threadSourceStaticMethod(name) {
			return nil
		}
		p.receiverScope = scope
		p.dispatchScope = scope
		p.classQualifier = true
	} else {
		inference := ctx.Clone()
		inference.expectedType = ""
		inference.expectedTypeRoot = nil
		var origin inferredJavaTypeOrigin
		typ, known := inferExprJavaType(object, inference, source, &origin)
		_, rank := javaArrayTypeParts(typ)
		if !known || rank != 0 || origin.unresolved || origin.parameter != nil || origin.nominalScope == nil || threadSourceRoot(origin.nominalScope, ctx) == nil {
			return nil
		}
		p.receiverScope = origin.nominalScope
		p.dispatchScope = p.receiverScope
	}
	if p.dispatchScope != nil {
		p.selected = findBestMethodInHierarchy(p.dispatchScope, name, args, !p.classQualifier, true, ctx, source)
	}
	return p
}

func threadSourceInvocation(node *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	p := threadSourceSelectInvocation(node, source, ctx)
	if p == nil {
		return nil
	}
	object := node.ChildByFieldName("object")
	argsNode := node.ChildByFieldName("arguments")
	// Existing ordinary source lowering owns normal source declarations.
	if p.selected != nil && p.selected.def != nil && p.selected.def.DeclarationNode != nil {
		if !p.super {
			return nil
		}
		receiver := ParseExpr(object, source, ctx)
		args, expand := parseResolvedInvocationArguments(p.selected, argsNode, source, ctx, nil, nil, node)
		return markDirectVarargsExpansion(methodCall(receiver, executionImplementationName(p.selected.def, p.selected.owner, ctx), prependExecutionMethodArgument(ctx, p.selected.def, args)...), expand)
	}
	if threadSourceStaticMethod(p.name) {
		var call ast.Expr
		var results *ast.FieldList
		switch p.name {
		case "interrupted":
			call = stdjavaCall(ctx, "ThreadInterruptedExecution", intrinsicExecutionExpr(ctx))
			results = &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}}
		case "currentThread":
			call = stdjavaCall(ctx, "ThreadCurrentThread", intrinsicExecutionExpr(ctx))
			results = &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("Thread", ctx)}}}}
		case "onSpinWait":
			call = stdjavaCall(ctx, "ThreadOnSpinWaitExecution", intrinsicExecutionExpr(ctx))
		}
		if object == nil || p.super || p.classQualifier {
			return call
		}
		return stageStaticQualifierWithResults(ParseExpr(object, source, ctx), call, results)
	}
	var receiver ast.Expr
	if object == nil || p.super && p.dispatchScope == nil {
		receiver = ast.NewIdent(sourceInstanceReceiverName(ctx))
	} else {
		receiver = ParseExpr(object, source, ctx)
	}
	view := threadSourceProjection(receiver, ctx)
	if p.super && p.dispatchScope == nil {
		view = threadSourceNativeViewExpr(p.receiverScope, receiver, ctx)
	}
	function := map[string]string{"interrupt": "ThreadInterrupt", "isInterrupted": "ThreadIsInterrupted", "run": "ThreadRun", "start": "ThreadStart"}[p.name]
	if function != "" {
		if p.super {
			function += "Default"
		}
		return stdjavaCall(ctx, function+"Execution", intrinsicExecutionExpr(ctx), view)
	}
	return threadSourceFinalNativeInvocation(node, view, p.name, ctx, source)
}

// Staging checks null only after arguments/unboxing, retaining Java call order.
func threadSourceFinalNativeInvocation(node *sitter.Node, view ast.Expr, name string, ctx Ctx, source []byte) ast.Expr {
	used := affineLoopUsedNames(node, source, ctx)
	receiverName := synchronizedUniqueLocalName("__java2goThreadReceiver", used)
	body := []ast.Stmt{stagedInvocationLocal(receiverName, view)}
	argsNode := node.ChildByFieldName("arguments")
	args := parseArgumentListWithExpectedTypes(argsNode, source, ctx, []string{"long", "int"})
	staged := make([]ast.Expr, len(args))
	for i, value := range args {
		local := synchronizedUniqueLocalName("__java2goThreadArgument", used)
		typ := ast.Expr(ast.NewIdent("int64"))
		if i == 1 {
			typ = ast.NewIdent("int32")
		}
		body = append(body, &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{ast.NewIdent(local)}, Type: typ, Values: []ast.Expr{value}}}}})
		staged[i] = ast.NewIdent(local)
	}
	goName := map[string]string{"join": "Join", "isAlive": "IsAlive", "getName": "GetNameReference"}[name]
	if name == "join" && len(args) > 0 {
		goName = "JoinTimed"
	}
	call := methodCall(stdjavaCall(ctx, "ReferenceRequireNonNull", ast.NewIdent(receiverName)), goName, staged...)
	var result *ast.FieldList
	switch name {
	case "isAlive":
		result = &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}}
	case "getName":
		result = &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("JavaString", ctx)}}}}
	}
	if result == nil {
		body = append(body, &ast.ExprStmt{X: call})
	} else {
		body = append(body, &ast.ReturnStmt{Results: []ast.Expr{call}})
	}
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: result}, Body: &ast.BlockStmt{List: body}}}
}

func threadSourceInheritedResultType(node *sitter.Node, ctx Ctx, source []byte, origins ...*inferredJavaTypeOrigin) (string, bool) {
	p := threadSourceSelectInvocation(node, source, ctx)
	if p == nil || p.selected != nil && p.selected.def != nil && p.selected.def.DeclarationNode != nil {
		return "", false
	}
	switch p.name {
	case "isInterrupted", "interrupted", "isAlive":
		return "boolean", true
	case "getName":
		setInferredJavaTypeOrigin(origins, intrinsicJavaTypeOrigin("java.lang.String"))
		return "java.lang.String", true
	case "currentThread":
		setInferredJavaTypeOrigin(origins, intrinsicJavaTypeOrigin("java.lang.Thread"))
		return "java.lang.Thread", true
	case "interrupt", "run", "start", "join", "onSpinWait":
		return "void", true
	}
	return "", false
}
