package transpiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/astutil"
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
	sitter "github.com/smacker/go-tree-sitter"
)

// extractTypeArgsFromString extracts type arguments from a string like "List<Integer>"
// or nested generics like "Map<String, List<Integer>>".
// Returns ["Integer"] or ["String", "List<Integer>"] respectively, or nil if no type arguments found
// or if the input has unbalanced angle brackets.
func extractTypeArgsFromString(typeStr string) []string {
	start := strings.Index(typeStr, "<")
	end := strings.LastIndex(typeStr, ">")
	if start == -1 || end == -1 || end <= start {
		return nil
	}
	argsStr := typeStr[start+1 : end]

	// Split by commas, but only at the top level (not inside nested angle brackets)
	var result []string
	var current strings.Builder
	depth := 0

	for _, ch := range argsStr {
		switch ch {
		case '<':
			depth++
			current.WriteRune(ch)
		case '>':
			depth--
			if depth < 0 {
				log.WithField("typeStr", typeStr).Warn("Unbalanced angle brackets in type string: too many '>'")
				return nil
			}
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				// Top-level comma - split here
				trimmed := strings.TrimSpace(current.String())
				if trimmed != "" {
					result = append(result, trimmed)
				}
				current.Reset()
			} else {
				// Comma inside nested generics - keep it
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	// Validate that all angle brackets were closed
	if depth != 0 {
		log.WithField("typeStr", typeStr).Warn("Unbalanced angle brackets in type string: unclosed '<'")
		return nil
	}

	// Don't forget the last argument
	trimmed := strings.TrimSpace(current.String())
	if trimmed != "" {
		result = append(result, trimmed)
	}

	return result
}

// ParseExpr parses an expression type
func ParseExpr(node *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	switch node.Type() {
	case "ERROR":
		log.WithFields(log.Fields{
			"parsed":    node.Content(source),
			"className": ctx.className,
		}).Warn("Expression parse error")
		return &ast.BadExpr{}
	case "comment":
		return &ast.BadExpr{}
	case "update_expression":
		// Go has no ++/-- in expression position (e.g. `println(counter++)` or
		// `arr[i++]`), so route through stdjava helpers that take a pointer to the
		// operand, mutate it, and return the appropriate (pre or post) value.
		//
		// A post expression has the operand first (`i++`); a pre expression has the
		// operator first (`++i`).
		var operandNode *sitter.Node
		var post bool
		if node.Child(0).IsNamed() {
			operandNode = node.Child(0)
			post = true
		} else {
			operandNode = node.Child(1)
		}

		increment := strings.Contains(node.Content(source), "++")
		var helper string
		switch {
		case post && increment:
			helper = "PostIncrement"
		case post && !increment:
			helper = "PostDecrement"
		case !post && increment:
			helper = "PreIncrement"
		default:
			helper = "PreDecrement"
		}

		return stdjavaCall(ctx, helper, &ast.UnaryExpr{
			Op: token.AND,
			X:  ParseExpr(operandNode, source, ctx),
		})
	case "class_literal":
		// Class literals refer to the class directly, such as
		// Object.class
		return &ast.BadExpr{}
	case "assignment_expression":
		return &ast.CallExpr{
			Fun: &ast.Ident{Name: "AssignmentExpression"},
			Args: []ast.Expr{
				ParseExpr(node.Child(0), source, ctx),
				&ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%s\"", node.Child(1).Content(source))},
				ParseExpr(node.Child(2), source, ctx),
			},
		}
	case "super":
		return superSelectorExpr(ctx)
	case "lambda_expression":
		// Lambdas can either be called with a list of expressions
		// (ex: (n1, n1) -> {}), or with a single expression
		// (ex: n1 -> {})

		var lambdaBody *ast.BlockStmt

		var lambdaParameters *ast.FieldList
		samMethod, samTypeBindings := resolveFunctionalInterfaceMethod(ctx, ctx.expectedType)
		var lambdaResults *ast.FieldList

		bodyNode := node.ChildByFieldName("body")

		if samMethod != nil && strings.TrimSpace(samMethod.OriginalType) != "" && strings.TrimSpace(samMethod.OriginalType) != "void" {
			resultType := substituteJavaTypeParams(samMethod.OriginalType, samTypeBindings)
			lambdaResults = &ast.FieldList{
				List: []*ast.Field{
					{Type: javaTypeStringToGoTypeExpr(resultType, inScopeTypeParameters(ctx), ctx)},
				},
			}
		}

		switch bodyNode.Type() {
		case "block":
			lambdaBody = ParseStmt(bodyNode, source, ctx).(*ast.BlockStmt)
		default:
			// Lambdas can be called inline without a block expression
			inlineExpr := ParseExpr(bodyNode, source, ctx)
			inlineStmt := ast.Stmt(&ast.ExprStmt{X: inlineExpr})
			if lambdaResults != nil && len(lambdaResults.List) > 0 {
				inlineStmt = &ast.ReturnStmt{Results: []ast.Expr{inlineExpr}}
			}
			lambdaBody = &ast.BlockStmt{
				List: []ast.Stmt{
					inlineStmt,
				},
			}
		}

		paramNode := node.ChildByFieldName("parameters")
		paramCount := 0
		if paramNode != nil {
			paramCount = int(paramNode.NamedChildCount())
			if paramNode.Type() != "inferred_parameters" && paramNode.Type() != "formal_parameters" {
				paramCount = 1
			}
		}
		inferredParamTypes := inferLambdaParameterTypeExprs(ctx, paramCount)

		switch paramNode.Type() {
		case "formal_parameters":
			lambdaParameters = ParseNode(paramNode, source, ctx).(*ast.FieldList)
		case "inferred_parameters":
			lambdaParameters = &ast.FieldList{}
			for ind, param := range nodeutil.NamedChildrenOf(paramNode) {
				paramType := ast.Expr(&ast.Ident{Name: "any"})
				if ind < len(inferredParamTypes) && inferredParamTypes[ind] != nil {
					paramType = inferredParamTypes[ind]
				}
				lambdaParameters.List = append(lambdaParameters.List, &ast.Field{
					Names: []*ast.Ident{identFromNode(param, source)},
					Type:  paramType,
				})
			}
		default:
			// If we can't identify the types of the parameters, then just set their
			// types to any
			paramType := ast.Expr(&ast.Ident{Name: "any"})
			if len(inferredParamTypes) > 0 && inferredParamTypes[0] != nil {
				paramType = inferredParamTypes[0]
			}
			lambdaParameters = &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Names: []*ast.Ident{identFromNode(paramNode, source)},
						Type:  paramType,
					},
				},
			}
		}

		lambdaFunc := &ast.FuncLit{
			Type: &ast.FuncType{
				Params:  lambdaParameters,
				Results: lambdaResults,
			},
			Body: lambdaBody,
		}

		if adapted := wrapLambdaWithFunctionalInterfaceAdapter(lambdaFunc, ctx.expectedType, ctx); adapted != nil {
			return adapted
		}

		return lambdaFunc
	case "method_reference":
		// This refers to manually selecting a function from a specific class and
		// passing it in as an argument in the `func(className::methodName)` style

		// For class constructors such as `Class::new`, you only get one node
		if node.NamedChildCount() < 2 {
			constructorRef := ast.Expr(&ast.SelectorExpr{
				X:   ParseExpr(node.NamedChild(0), source, ctx),
				Sel: &ast.Ident{Name: "new"},
			})
			if adapted := wrapLambdaWithFunctionalInterfaceAdapter(constructorRef, ctx.expectedType, ctx); adapted != nil {
				return adapted
			}
			return constructorRef
		}

		targetNode := node.NamedChild(0)
		methodName := node.NamedChild(1).Content(source)

		methodExpr := ast.Expr(&ast.SelectorExpr{
			X:   ParseExpr(targetNode, source, ctx),
			Sel: identFromNode(node.NamedChild(1), source),
		})

		if classScope := resolveClassScopeByIdentifier(ctx, source, targetNode); classScope != nil {
			if staticDef := findStaticMethodByName(classScope, methodName); staticDef != nil {
				staticPkg := resolveJavaPackageForType(ctx, targetNode.Content(source), classScope)
				methodExpr = qualifiedNameExpr(staticDef.Name, staticPkg, ctx)
			}
		}

		if adapted := wrapLambdaWithFunctionalInterfaceAdapter(methodExpr, ctx.expectedType, ctx); adapted != nil {
			return adapted
		}

		return methodExpr
	case "array_initializer":
		// A literal that initilzes an array, such as `{1, 2, 3}`
		items := []ast.Expr{}
		for _, c := range nodeutil.NamedChildrenOf(node) {
			items = append(items, ParseExpr(c, source, ctx))
		}

		// If there wasn't a type for the array specified, then use the one that has been defined
		if _, ok := ctx.lastType.(*ast.ArrayType); ctx.lastType != nil && ok {
			return &ast.CompositeLit{
				Type: ctx.lastType.(*ast.ArrayType),
				Elts: items,
			}
		}
		return &ast.CompositeLit{
			Elts: items,
		}
	case "method_invocation":
		// Methods with a selector are called as X.Sel(Args)
		// Otherwise, they are called as Fun(Args)
		if node.ChildByFieldName("object") != nil {
			objectNode := node.ChildByFieldName("object")
			methodName := node.ChildByFieldName("name").Content(source)
			methodIdent := identFromNode(node.ChildByFieldName("name"), source)

			if isSystemOutSelector(objectNode, source) && (methodName == "println" || methodName == "print") {
				argListNode := node.ChildByFieldName("arguments")
				args := parseArgumentListWithExpectedTypes(argListNode, source, ctx, nil)
				// Java prints a char as its glyph, but a transpiled char is a Go
				// rune (int32), which fmt prints as a number. Wrap char-typed
				// arguments in string(...) so println(c) prints the character.
				if argListNode != nil {
					for ind, argNode := range nodeutil.NamedChildrenOf(argListNode) {
						if ind < len(args) && isCharTypedExprNode(argNode, ctx, source) {
							args[ind] = &ast.CallExpr{Fun: &ast.Ident{Name: "string"}, Args: []ast.Expr{args[ind]}}
						}
					}
				}
				funName := "Println"
				if methodName == "print" {
					funName = "Print"
				}
				return &ast.CallExpr{
					Fun:  qualifiedNameExpr(funName, "fmt", ctx),
					Args: args,
				}
			}

			// Standard-library intrinsics (String, StringBuilder, Math, boxed
			// types, ...) are rewritten via a data-driven table. Instance
			// intrinsics dispatch on the receiver's Java type; static intrinsics
			// dispatch on a class-name receiver.
			if rewritten, ok := tryInstanceIntrinsic(objectNode, methodName, source, ctx); ok {
				return rewritten
			}
			if rewritten, ok := tryStaticIntrinsic(objectNode, methodName, source, ctx); ok {
				return rewritten
			}

			argListNode := node.ChildByFieldName("arguments")
			argCount := 0
			if argListNode != nil {
				argCount = int(argListNode.NamedChildCount())
			}

			// Throwable.getMessage()/printStackTrace() on a caught exception are
			// routed through the stdjava runtime, which understands both the
			// built-in exception types and user-defined ones.
			if argCount == 0 && (methodName == "getMessage" || methodName == "printStackTrace") {
				if javaType, ok := inferExprJavaType(objectNode, ctx, source); ok && isExceptionJavaType(ctx, javaType) {
					runtimeFn := "GetMessage"
					if methodName == "printStackTrace" {
						runtimeFn = "PrintStackTrace"
					}
					return &ast.CallExpr{
						Fun:  stdjavaQualifiedExpr(runtimeFn, ctx),
						Args: []ast.Expr{ParseExpr(objectNode, source, ctx)},
					}
				}
			}

			// Check if this is an enum values() call
			// Transform EnumName.values() to EnumNameValues()
			if objectNode.Type() == "identifier" && methodName == "values" {
				if enumScope := resolveClassScopeByIdentifier(ctx, source, objectNode); enumScope != nil && enumScope.IsEnum {
					enumPkg := resolveJavaPackageForType(ctx, objectNode.Content(source), enumScope)
					return &ast.CallExpr{
						Fun:  qualifiedNameExpr(enumScope.Class.Name+"Values", enumPkg, ctx),
						Args: []ast.Expr{},
					}
				}
			}

			objectExpr := ParseExpr(objectNode, source, ctx)
			var expectedArgTypes []string
			classScope := resolveClassScopeByIdentifier(ctx, source, objectNode)
			staticDef := findStaticMethodByNameAndArgCount(classScope, methodName, argCount)
			if staticDef != nil {
				expectedArgTypes = definitionParameterOriginalTypes(staticDef)
			}
			target := resolveInvocationTarget(objectNode, ctx, source)
			instanceResolution := (*methodResolution)(nil)
			staticResolution := (*methodResolution)(nil)
			if target != nil {
				instanceResolution = findInstanceMethodInHierarchy(target.classScope, methodName, argCount, ctx)
				if instanceResolution != nil {
					expectedArgTypes = definitionParameterOriginalTypes(instanceResolution.def)
				} else {
					staticResolution = findStaticMethodInHierarchy(target.classScope, methodName, argCount, ctx)
					if staticResolution != nil {
						expectedArgTypes = definitionParameterOriginalTypes(staticResolution.def)
					}
				}
			}
			args := parseArgumentListWithExpectedTypes(argListNode, source, ctx, expectedArgTypes)
			typeArgs := explicitTypeArgumentExprs(node, source, inScopeTypeParameters(ctx), ctx)

			// If this is a static call on a class name (e.g., Utils.<T>id(...)),
			// rewrite it to a plain function call to match how static methods are emitted.
			if staticDef != nil {
				staticPkg := resolveJavaPackageForType(ctx, objectNode.Content(source), classScope)
				fun := qualifiedNameExpr(staticDef.Name, staticPkg, ctx)
				fun = applyTypeArguments(fun, typeArgs)
				return &ast.CallExpr{Fun: fun, Args: args}
			}

			if rewritten := maybeRewriteInstanceGenericMethodInvocationWithTarget(target, objectExpr, methodName, args, node, ctx, source); rewritten != nil {
				return rewritten
			}

			if target != nil {
				if instanceResolution != nil {
					methodIdent = &ast.Ident{Name: instanceResolution.def.Name}
				} else if staticResolution != nil {
					// Java permits calling static methods via an instance expression; rewrite
					// to a plain function call to match codegen.
					fun := qualifiedNameExpr(staticResolution.def.Name, findJavaPackageForClassScope(staticResolution.owner), ctx)
					fun = applyTypeArguments(fun, typeArgs)
					return &ast.CallExpr{Fun: fun, Args: args}
				}
			}

			return &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   objectExpr,
					Sel: methodIdent,
				},
				Args: args,
			}
		}
		methodName := node.ChildByFieldName("name").Content(source)
		argListNode := node.ChildByFieldName("arguments")
		argCount := 0
		if argListNode != nil {
			argCount = int(argListNode.NamedChildCount())
		}
		var expectedArgTypes []string
		implicitInstanceResolution := (*methodResolution)(nil)
		implicitStaticResolution := (*methodResolution)(nil)
		if ctx.currentClass != nil {
			if ctx.localScope != nil && ctx.localScope.OriginalName != "" && !ctx.localScope.IsStatic {
				implicitInstanceResolution = findInstanceMethodInHierarchy(ctx.currentClass, methodName, argCount, ctx)
				if implicitInstanceResolution != nil {
					expectedArgTypes = definitionParameterOriginalTypes(implicitInstanceResolution.def)
				}
			}
			if len(expectedArgTypes) == 0 {
				implicitStaticResolution = findStaticMethodInHierarchy(ctx.currentClass, methodName, argCount, ctx)
				if implicitStaticResolution != nil {
					expectedArgTypes = definitionParameterOriginalTypes(implicitStaticResolution.def)
				}
			}
		}
		args := parseArgumentListWithExpectedTypes(argListNode, source, ctx, expectedArgTypes)
		typeArgs := explicitTypeArgumentExprs(node, source, inScopeTypeParameters(ctx), ctx)

		// Unqualified invocation in Java is typically an implicit receiver call.
		// Only do this in a non-static method/constructor body where the receiver
		// variable exists.
		if ctx.currentClass != nil && ctx.localScope != nil && ctx.localScope.OriginalName != "" && !ctx.localScope.IsStatic {
			recv := &ast.Ident{Name: ShortName(ctx.className)}
			target := &invocationTargetInfo{
				classScope:    ctx.currentClass,
				classTypeArgs: typeParamExprs(ctx.currentClass.TypeParameterNames()),
			}
			if rewritten := maybeRewriteInstanceGenericMethodInvocationWithTarget(target, recv, methodName, args, node, ctx, source); rewritten != nil {
				return rewritten
			}
			if implicitInstanceResolution != nil {
				return &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   recv,
						Sel: &ast.Ident{Name: implicitInstanceResolution.def.Name},
					},
					Args: args,
				}
			}
			// Unqualified call to an enclosing class's instance method from inside
			// an inner class: route it through the enclosing-instance field, e.g.
			// foo() -> or.outer.Foo().
			if enclSel := enclosingMemberMethodSelector(methodName, argCount, ctx); enclSel != nil {
				return &ast.CallExpr{Fun: enclSel, Args: args}
			}
		}

		// Otherwise, treat as a plain function call (static methods are emitted as
		// functions).
		if ctx.currentClass != nil {
			if implicitStaticResolution != nil {
				fun := qualifiedNameExpr(implicitStaticResolution.def.Name, findJavaPackageForClassScope(implicitStaticResolution.owner), ctx)
				fun = applyTypeArguments(fun, typeArgs)
				return &ast.CallExpr{Fun: fun, Args: args}
			}
		}

		fun := ast.Expr(identFromNode(node.ChildByFieldName("name"), source))
		fun = applyTypeArguments(fun, typeArgs)
		return &ast.CallExpr{Fun: fun, Args: args}
	case "object_creation_expression":
		// This is called when anything is created with a constructor

		objectType := node.ChildByFieldName("type")

		// An anonymous class declaration carries a class_body. When the supertype
		// is a single-abstract-method interface, lower it to the functional
		// interface adapter with a closure (mirroring lambda lowering).
		if classBody := objectCreationClassBody(node); classBody != nil {
			if lowered := lowerAnonymousClass(node, objectType, classBody, source, ctx); lowered != nil {
				return lowered
			}
		}

		// The `outer.new Inner()` / `this.new Inner()` qualifier form is handled
		// below by threading the leading expression as the enclosing instance.

		// Get all the arguments, and look up their types
		objectArguments := node.ChildByFieldName("arguments")
		arguments := make([]ast.Expr, objectArguments.NamedChildCount())
		argumentTypes := make([]string, objectArguments.NamedChildCount())
		for ind, argument := range nodeutil.NamedChildrenOf(objectArguments) {
			arguments[ind] = ParseExpr(argument, source, ctx)

			// Look up each argument and find its type
			if argument.Type() != "identifier" {
				argumentTypes[ind] = symbol.TypeOfLiteral(argument, source)
			} else {
				if localDef := ctx.localScope.FindVariable(argument.Content(source)); localDef != nil {
					argumentTypes[ind] = localDef.OriginalType
					// Otherwise, a variable may exist as a global variable
				} else if def := ctx.currentFile.FindField().ByOriginalName(argument.Content(source)); len(def) > 0 {
					argumentTypes[ind] = def[0].OriginalType
				}
			}
		}

		// Extract base class name and type arguments
		var className string
		var typeArgs []string
		isDiamond := false
		if objectType.Type() == "generic_type" {
			className = objectType.NamedChild(0).Content(source)
			typeArgs = astutil.ExtractTypeArguments(objectType, source)
			// Diamond operator: generic_type with no type arguments and explicit "<>" in source
			if len(typeArgs) == 0 {
				content := objectType.Content(source)
				// Look for "<>" after the class name (allowing for whitespace)
				afterClass := strings.TrimSpace(content[len(className):])
				isDiamond = strings.HasPrefix(afterClass, "<>")
			}
		} else {
			className = objectType.Content(source)
		}

		// Built-in exception types (java.lang/java.io) are modelled by the stdjava
		// runtime, so `new IllegalArgumentException("msg")` becomes a call to the
		// corresponding stdjava constructor, preserving the detail message.
		if isBuiltinExceptionType(className) && resolveClassScopeByQualifiedName(ctx, className) == nil {
			return builtinExceptionConstructorExpr(className, arguments, ctx)
		}

		// Standard-library constructors (StringBuilder, ...) are handled by the
		// intrinsics table, which maps them onto stdjava runtime constructors.
		if rewritten, ok := tryConstructorIntrinsic(className, arguments, ctx); ok {
			return rewritten
		}

		// Find the respective constructor (if we have symbol info for that class).
		var constructor *symbol.Definition
		targetScope := resolveClassScopeByQualifiedName(ctx, className)
		constructor = findMatchingConstructor(targetScope, stripJavaQualifier(className), argumentTypes)
		targetPkg := resolveJavaPackageForType(ctx, className, targetScope)

		// Inner-class instantiation captures an enclosing instance. For the
		// explicit `outer.new Inner()` form, the qualifier expression precedes the
		// type; for the implicit `new Inner()` form inside the enclosing class, the
		// current receiver is captured. The captured value is threaded as the
		// constructor's leading argument.
		if targetScope != nil && targetScope.IsInner {
			if enclArg := enclosingInstanceArgument(node, objectType, source, ctx); enclArg != nil {
				arguments = append([]ast.Expr{enclArg}, arguments...)
			}
		}

		// Helper function to add type arguments to a function expression
		addTypeArgs := func(funExpr ast.Expr, args []string) ast.Expr {
			if len(args) == 0 {
				return funExpr
			}
			scopeTypeParams := inScopeTypeParameters(ctx)
			typeArgExprs := make([]ast.Expr, 0, len(args))
			for _, ta := range args {
				typeArgExprs = append(typeArgExprs, javaTypeStringToGoTypeExpr(ta, scopeTypeParams, ctx))
			}
			return applyTypeArguments(funExpr, typeArgExprs)
		}

		// Determine effective type arguments:
		// 1. If explicit type arguments provided, use them
		// 2. If diamond operator, try to infer from expectedType
		// 3. For inner class constructors (non-diamond), use parent class type params
		effectiveTypeArgs := typeArgs
		if len(effectiveTypeArgs) == 0 {
			// For diamond operator, try to infer from expectedType
			if isDiamond && ctx.expectedType != "" {
				effectiveTypeArgs = extractTypeArgsFromString(ctx.expectedType)
			}

			// For inner class constructors (not diamond), use parent class type parameters
			// This handles cases like `new Node(element)` inside a generic class
			if len(effectiveTypeArgs) == 0 && !isDiamond && len(ctx.currentClass.TypeParameters) > 0 {
				// Check if className is a nested class of the current class
				for _, sub := range ctx.currentClass.Subclasses {
					if sub.Class.OriginalName == className {
						effectiveTypeArgs = ctx.currentClass.TypeParameterNames()
						break
					}
				}
			}
		}

		if constructor != nil {
			funExpr := addTypeArgs(qualifiedNameExpr(constructor.Name, targetPkg, ctx), effectiveTypeArgs)
			return &ast.CallExpr{
				Fun:  funExpr,
				Args: arguments,
			}
		}

		// No explicit constructor matched. If we resolved the target class within
		// our own symbols, it was given a synthesized default constructor named
		// New<ResolvedGoName>; use the resolved Go name so nested classes (renamed
		// e.g. Outer -> OuterInner) bind correctly.
		if targetScope != nil && targetScope.Class != nil && targetScope.Class.Name != "" {
			funExpr := addTypeArgs(qualifiedNameExpr("New"+targetScope.Class.Name, targetPkg, ctx), effectiveTypeArgs)
			return &ast.CallExpr{
				Fun:  funExpr,
				Args: arguments,
			}
		}

		// Otherwise the constructor is genuinely unresolved (external type with no
		// symbol info), so fall back to <Type> + "Construct".
		funExpr := addTypeArgs(qualifiedNameExpr("Construct"+stripJavaQualifier(className), targetPkg, ctx), effectiveTypeArgs)
		return &ast.CallExpr{
			Fun:  funExpr,
			Args: arguments,
		}
	case "array_creation_expression":
		dimensions := []ast.Expr{}
		// The "type" field is the element type (e.g. `int` in `new int[]{...}`),
		// not the array type, so wrap it so the initializer can emit a typed
		// composite literal like `[]int32{...}` instead of a bare `{...}`.
		elementType := astutil.ParseType(node.ChildByFieldName("type"), source)
		var initializer ast.Expr

		for _, child := range nodeutil.NamedChildrenOf(node) {
			if child.Type() == "dimensions_expr" {
				dimensions = append(dimensions, ParseExpr(child, source, ctx))
			} else if child.Type() == "array_initializer" {
				initCtx := ctx.Clone()
				initCtx.lastType = &ast.ArrayType{Elt: elementType}
				initializer = ParseExpr(child, source, initCtx)
			}
		}

		if initializer != nil {
			return initializer
		}

		if len(dimensions) == 0 {
			panic("Array had zero dimensions")
		}

		return GenMultiDimArray(symbol.NodeToStr(elementType), dimensions)
	case "instanceof_expression":
		left := node.ChildByFieldName("left")
		if left == nil && node.NamedChildCount() > 0 {
			left = node.NamedChild(0)
		}
		right := node.ChildByFieldName("right")
		if right == nil && node.NamedChildCount() > 1 {
			right = node.NamedChild(1)
		}
		if left == nil || right == nil {
			return &ast.BadExpr{}
		}

		assertType := instanceofAssertTypeExpr(right.Content(source), ctx)
		if assertType == nil {
			return &ast.BadExpr{}
		}

		return &ast.CallExpr{
			Fun: &ast.FuncLit{
				Type: &ast.FuncType{
					Results: &ast.FieldList{
						List: []*ast.Field{
							{Type: &ast.Ident{Name: "bool"}},
						},
					},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.AssignStmt{
							Lhs: []ast.Expr{
								&ast.Ident{Name: "_"},
								&ast.Ident{Name: "ok"},
							},
							Tok: token.DEFINE,
							Rhs: []ast.Expr{
								&ast.TypeAssertExpr{
									X: &ast.CallExpr{
										Fun:  &ast.Ident{Name: "any"},
										Args: []ast.Expr{ParseExpr(left, source, ctx)},
									},
									Type: assertType,
								},
							},
						},
						&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "ok"}}},
					},
				},
			},
		}
	case "dimensions_expr":
		return ParseExpr(node.NamedChild(0), source, ctx)
	case "binary_expression":
		operator := node.Child(1).Content(source)
		if operator == ">>>" {
			return stdjavaCall(ctx, "UnsignedRightShift",
				ParseExpr(node.Child(0), source, ctx),
				maskedShiftAmount(node.Child(2), source, ctx),
			)
		}
		leftNode := node.Child(0)
		rightNode := node.Child(2)
		leftExpr := ParseExpr(leftNode, source, ctx)
		rightExpr := ParseExpr(rightNode, source, ctx)
		if operator == "+" && (isStringLikeExprNode(leftNode, ctx, source) || isStringLikeExprNode(rightNode, ctx, source) || isFmtSprintfCall(leftExpr)) {
			return mergeFmtSprintCall(leftExpr, rightExpr, ctx)
		}
		// Java masks shift counts (int: low 5 bits, long: low 6 bits) before
		// shifting, whereas Go applies the full count. Mask constant shift amounts
		// at transpile time so e.g. `1 << 32` stays 1.
		if operator == "<<" || operator == ">>" {
			rightExpr = maskedShiftAmount(rightNode, source, ctx)
		}
		return &ast.BinaryExpr{
			X:  leftExpr,
			Op: StrToToken(operator),
			Y:  rightExpr,
		}
	case "unary_expression":
		return &ast.UnaryExpr{
			Op: StrToToken(node.Child(0).Content(source)),
			X:  ParseExpr(node.Child(1), source, ctx),
		}
	case "parenthesized_expression":
		return &ast.ParenExpr{
			X: ParseExpr(node.NamedChild(0), source, ctx),
		}
	case "ternary_expression":
		// Ternary expressions are replaced with a function that takes in the
		// condition, and returns one of the two values, depending on the condition

		args := []ast.Expr{}
		for _, c := range nodeutil.NamedChildrenOf(node) {
			args = append(args, ParseExpr(c, source, ctx))
		}
		return stdjavaCall(ctx, "Ternary", args...)
	case "cast_expression":
		targetJavaType := node.NamedChild(0).Content(source)
		targetType := javaTypeStringToGoTypeExpr(targetJavaType, inScopeTypeParameters(ctx), ctx)
		valueExpr := ParseExpr(node.NamedChild(1), source, ctx)

		if isPrimitiveCastTarget(targetType) {
			return &ast.CallExpr{
				Fun:  targetType,
				Args: []ast.Expr{valueExpr},
			}
		}

		return &ast.TypeAssertExpr{
			X: &ast.CallExpr{
				Fun:  &ast.Ident{Name: "any"},
				Args: []ast.Expr{valueExpr},
			},
			Type: targetType,
		}
	case "field_access":
		// X.Sel
		obj := node.ChildByFieldName("object")

		// Standard-library constant access (Integer.MAX_VALUE, Math.PI, ...).
		if fieldNode := node.ChildByFieldName("field"); fieldNode != nil {
			if rewritten, ok := tryStaticFieldIntrinsic(obj, fieldNode.Content(source), source, ctx); ok {
				return rewritten
			}

			// Java's array.length is a field; Go uses len(). Only lower when the
			// receiver is known to be an array, so a user class field named
			// "length" is left untouched.
			if fieldNode.Content(source) == "length" && isArrayTypedExprNode(obj, ctx, source) {
				return &ast.CallExpr{
					Fun:  &ast.Ident{Name: "int32"},
					Args: []ast.Expr{&ast.CallExpr{Fun: &ast.Ident{Name: "len"}, Args: []ast.Expr{ParseExpr(obj, source, ctx)}}},
				}
			}
		}

		// Qualified enum constant access (Day.WED, Planet.EARTH). Enum constants are
		// generated as package-level vars named after the constant, so `Day.WED`
		// lowers to `WED` (qualified with the enum's Go package when it lives
		// elsewhere) rather than an invalid `Day.WED` selector.
		if fieldNode := node.ChildByFieldName("field"); fieldNode != nil {
			if enumScope := resolveClassScopeByIdentifier(ctx, source, obj); enumScope != nil && enumScope.IsEnum {
				constName := fieldNode.Content(source)
				enumPkg := resolveJavaPackageForType(ctx, obj.Content(source), enumScope)
				return qualifiedNameExpr(constName, enumPkg, ctx)
			}
		}

		if obj.Type() == "this" {
			fieldName := node.ChildByFieldName("field").Content(source)
			def := findFieldInHierarchy(ctx.currentClass, fieldName, ctx)
			selName := fieldName
			if def != nil && def.Name != "" {
				selName = def.Name
			}

			return &ast.SelectorExpr{
				X:   ParseExpr(node.ChildByFieldName("object"), source, ctx),
				Sel: &ast.Ident{Name: selName},
			}
		}
		return &ast.SelectorExpr{
			X:   ParseExpr(obj, source, ctx),
			Sel: identFromNode(node.ChildByFieldName("field"), source),
		}
	case "array_access":
		return &ast.IndexExpr{
			X:     ParseExpr(node.NamedChild(0), source, ctx),
			Index: ParseExpr(node.NamedChild(1), source, ctx),
		}
	case "scoped_identifier":
		return ParseExpr(node.NamedChild(0), source, ctx)
	case "this":
		return &ast.Ident{Name: ShortName(ctx.className)}
	case "identifier":
		identName := node.Content(source)
		if ctx.localScope != nil {
			if param := ctx.localScope.ParameterByName(identName); param != nil {
				return &ast.Ident{Name: param.Name}
			}
			if local := ctx.localScope.FindVariable(identName); local != nil {
				return &ast.Ident{Name: local.Name}
			}
		}
		if ctx.currentClass != nil {
			if field := findFieldInHierarchy(ctx.currentClass, identName, ctx); field != nil {
				if field.IsStatic {
					return &ast.Ident{Name: field.Name}
				}
				if ctx.localScope != nil && ctx.localScope.IsStatic {
					return &ast.Ident{Name: field.Name}
				}
				recvName := ctx.className
				if recvName == "" && ctx.currentClass.Class != nil {
					recvName = ctx.currentClass.Class.Name
				}
				if recvName != "" {
					return &ast.SelectorExpr{
						X:   &ast.Ident{Name: ShortName(recvName)},
						Sel: &ast.Ident{Name: field.Name},
					}
				}
			}
			// Unqualified access to an enclosing class's instance field from inside
			// an inner class goes through the synthesized enclosing-instance field:
			// `base` -> `or.outer.base`.
			if enclAccess := enclosingMemberFieldAccess(identName, ctx); enclAccess != nil {
				return enclAccess
			}
		}
		if classScope := resolveClassScopeByQualifiedName(ctx, identName); classScope != nil {
			if alias := markJavaPackageUsage(ctx, resolveJavaPackageForType(ctx, identName, classScope)); alias != "" {
				return &ast.Ident{Name: alias}
			}
		}
		return &ast.Ident{Name: identName}
	case "type_identifier": // Any reference type
		switch node.Content(source) {
		// Special case for strings, because in Go, these are primitive types
		case "String":
			return &ast.Ident{Name: "string"}
		}

		if ctx.currentFile != nil {
			// Look for the class locally first
			if localClass := ctx.currentFile.FindClass(node.Content(source)); localClass != nil {
				return &ast.StarExpr{
					X: &ast.Ident{Name: localClass.Name},
				}
			}
		}

		return &ast.StarExpr{
			X: &ast.Ident{Name: node.Content(source)},
		}
	case "null_literal":
		return &ast.Ident{Name: "nil"}
	case "decimal_integer_literal":
		literal := node.Content(source)
		switch literal[len(literal)-1] {
		case 'L':
			return &ast.CallExpr{Fun: &ast.Ident{Name: "int64"}, Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: literal[:len(literal)-1]}}}
		}
		return &ast.Ident{Name: literal}
	case "hex_integer_literal":
		return &ast.Ident{Name: node.Content(source)}
	case "decimal_floating_point_literal":
		// This is something like 1.3D or 1.3F
		literal := node.Content(source)
		switch literal[len(literal)-1] {
		case 'D':
			return &ast.CallExpr{Fun: &ast.Ident{Name: "float64"}, Args: []ast.Expr{&ast.BasicLit{Kind: token.FLOAT, Value: literal[:len(literal)-1]}}}
		case 'F':
			return &ast.CallExpr{Fun: &ast.Ident{Name: "float32"}, Args: []ast.Expr{&ast.BasicLit{Kind: token.FLOAT, Value: literal[:len(literal)-1]}}}
		}
		return &ast.Ident{Name: literal}
	case "string_literal":
		return &ast.Ident{Name: node.Content(source)}
	case "character_literal":
		return &ast.Ident{Name: node.Content(source)}
	case "true", "false":
		return &ast.Ident{Name: node.Content(source)}
	}

	diag := reportUnsupported("expression", node, source, ctx)
	// Emit a placeholder expression that still compiles, so the rest of the file
	// can be converted. The panic call preserves the diagnostic at runtime.
	return &ast.CallExpr{
		Fun: &ast.Ident{Name: "panic"},
		Args: []ast.Expr{
			&ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", strings.TrimPrefix(unsupportedComment(diag), "// "))},
		},
	}
}

func isSystemOutSelector(node *sitter.Node, source []byte) bool {
	if node == nil {
		return false
	}
	if strings.TrimSpace(node.Content(source)) == "System.out" {
		return true
	}
	if node.Type() != "field_access" {
		return false
	}
	obj := node.ChildByFieldName("object")
	field := node.ChildByFieldName("field")
	if obj == nil || field == nil {
		return false
	}
	return obj.Type() == "identifier" && obj.Content(source) == "System" && field.Content(source) == "out"
}

func isStringLikeExprNode(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil {
		return false
	}

	switch node.Type() {
	case "string_literal":
		return true
	case "binary_expression":
		if node.Child(1) != nil && node.Child(1).Content(source) == "+" {
			return isStringLikeExprNode(node.Child(0), ctx, source) || isStringLikeExprNode(node.Child(2), ctx, source)
		}
	}

	if javaType, ok := inferExprJavaType(node, ctx, source); ok {
		baseType, _ := parseJavaTypeString(javaType)
		return stripJavaQualifier(baseType) == "String"
	}

	return false
}

func isFmtSprintfCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok || call == nil {
		return false
	}
	if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
		base, ok := fun.X.(*ast.Ident)
		return ok && base.Name == "fmt" && fun.Sel != nil && fun.Sel.Name == "Sprintf"
	}
	return false
}

func mergeFmtSprintCall(leftExpr, rightExpr ast.Expr, ctx Ctx) ast.Expr {
	if call, ok := leftExpr.(*ast.CallExpr); ok && isFmtSprintfCall(call) {
		// Append %v to existing format string and add argument
		formatLit := call.Args[0].(*ast.BasicLit)
		formatLit.Value = formatLit.Value[:len(formatLit.Value)-1] + "%v\""
		call.Args = append(call.Args, rightExpr)
		return call
	}
	return &ast.CallExpr{
		Fun: qualifiedNameExpr("Sprintf", "fmt", ctx),
		Args: []ast.Expr{
			&ast.BasicLit{Kind: token.STRING, Value: "\"%v%v\""},
			leftExpr, rightExpr,
		},
	}
}

// abstractClassToInterface post-processes a Go type expression that was
// generated for a method parameter.  If the underlying Java type resolves to an
// abstract class, the pointer-to-struct type (*ClassName) is replaced with the
// companion interface (ClassNameI) so that callers can pass any concrete
// subclass and instanceof checks keep working.
func abstractClassToInterface(expr ast.Expr, javaType string, ctx Ctx) ast.Expr {
	javaType = strings.TrimSpace(javaType)
	if javaType == "" || ctx.currentFile == nil {
		return expr
	}

	base, _ := parseJavaTypeString(javaType)
	scope := resolveClassScopeByQualifiedName(ctx, base)
	if scope == nil || !scope.IsAbstract || scope.IsInterface || scope.Class == nil {
		return expr
	}

	// Unwrap *pkg.ClassName  →  pkg.ClassNameI  (no pointer for interfaces).
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return expr
	}
	switch inner := star.X.(type) {
	case *ast.SelectorExpr:
		// Cross-package: *pkg.Task → pkg.TaskI
		inner.Sel = &ast.Ident{Name: inner.Sel.Name + "I"}
		return inner
	case *ast.Ident:
		// Same package: *Task → TaskI
		inner.Name = inner.Name + "I"
		return inner
	}
	return expr
}

// enclosingMemberFieldAccess resolves an unqualified identifier that names an
// instance field of an enclosing class, reached from inside an inner class. It
// walks the chain of enclosing classes, building a selector that hops through
// each synthesized enclosing-instance field, e.g. recv.outer.base. Returns nil
// if the identifier does not resolve to an enclosing instance field, or if the
// current scope is static.
func enclosingMemberFieldAccess(identName string, ctx Ctx) ast.Expr {
	scope := ctx.currentClass
	if scope == nil || !scope.IsInner {
		return nil
	}
	if ctx.localScope != nil && ctx.localScope.IsStatic {
		return nil
	}
	recvName := ctx.className
	if recvName == "" && scope.Class != nil {
		recvName = scope.Class.Name
	}
	if recvName == "" {
		return nil
	}

	var expr ast.Expr = &ast.Ident{Name: ShortName(recvName)}
	for cur := scope; cur != nil && cur.IsInner; cur = cur.Enclosing {
		expr = &ast.SelectorExpr{X: expr, Sel: &ast.Ident{Name: cur.EnclosingFieldName()}}
		encl := cur.Enclosing
		if encl == nil {
			break
		}
		if field := encl.FindFieldByName(identName); field != nil && !field.IsStatic {
			return &ast.SelectorExpr{X: expr, Sel: &ast.Ident{Name: field.Name}}
		}
	}
	return nil
}

// enclosingMemberMethodSelector resolves an unqualified method call that targets
// an enclosing class's instance method from inside an inner class, returning the
// selector to invoke (e.g. or.outer.Foo). Returns nil if no enclosing method
// matches or the current scope is static.
func enclosingMemberMethodSelector(methodName string, argCount int, ctx Ctx) ast.Expr {
	scope := ctx.currentClass
	if scope == nil || !scope.IsInner {
		return nil
	}
	if ctx.localScope != nil && ctx.localScope.IsStatic {
		return nil
	}
	recvName := ctx.className
	if recvName == "" && scope.Class != nil {
		recvName = scope.Class.Name
	}
	if recvName == "" {
		return nil
	}

	var expr ast.Expr = &ast.Ident{Name: ShortName(recvName)}
	for cur := scope; cur != nil && cur.IsInner; cur = cur.Enclosing {
		expr = &ast.SelectorExpr{X: expr, Sel: &ast.Ident{Name: cur.EnclosingFieldName()}}
		encl := cur.Enclosing
		if encl == nil {
			break
		}
		if resolved := findInstanceMethodInHierarchy(encl, methodName, argCount, ctx); resolved != nil && resolved.def != nil {
			return &ast.SelectorExpr{X: expr, Sel: &ast.Ident{Name: resolved.def.Name}}
		}
	}
	return nil
}

// enclosingInstanceArgument computes the enclosing-instance expression passed to
// an inner class's constructor. If the object creation is written as
// `qualifier.new Inner(...)`, the qualifier expression is used; otherwise the
// current method receiver (the implicit enclosing instance) is used.
func enclosingInstanceArgument(node, objectType *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	if node != nil && node.NamedChildCount() > 0 {
		first := node.NamedChild(0)
		// A leading named child that is not the type node is the explicit
		// qualifier in `qualifier.new Inner()`.
		if first != nil && objectType != nil && first.StartByte() != objectType.StartByte() {
			if _, isType := javaTypeNodeKinds[first.Type()]; !isType {
				return ParseExpr(first, source, ctx)
			}
		}
	}

	// Implicit form: use the current receiver as the enclosing instance.
	if ctx.className != "" {
		return &ast.Ident{Name: ShortName(ctx.className)}
	}
	return nil
}

func resolveClassScopeByQualifiedName(ctx Ctx, name string) *symbol.ClassScope {
	if ctx.currentFile == nil {
		return nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	// Try fully-qualified lookup first: "pkg.path.Class".
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		pkgPath := name[:idx]
		className := name[idx+1:]
		if pkg := symbol.GlobalScope.FindPackage(pkgPath); pkg != nil {
			if scope := pkg.FindClassScope(className); scope != nil {
				return scope
			}
		}
		// Fall back to unqualified lookup.
		name = className
	}

	// Current file.
	if scope := ctx.currentFile.FindClassScope(name); scope != nil {
		return scope
	}

	// Current package (other files).
	if pkg := symbol.GlobalScope.FindPackage(ctx.currentFile.Package); pkg != nil {
		if scope := pkg.FindClassScope(name); scope != nil {
			return scope
		}
	}

	// Imported package.
	if pkgPath, ok := ctx.currentFile.Imports[name]; ok {
		if pkg := symbol.GlobalScope.FindPackage(pkgPath); pkg != nil {
			if scope := pkg.FindClassScope(name); scope != nil {
				return scope
			}
		}
	}

	return nil
}

func resolveClassScopeByIdentifier(ctx Ctx, source []byte, objectNode *sitter.Node) *symbol.ClassScope {
	if objectNode == nil || objectNode.Type() != "identifier" {
		return nil
	}
	return resolveClassScopeByQualifiedName(ctx, objectNode.Content(source))
}

func resolveSuperclassScope(ctx Ctx, scope *symbol.ClassScope) *symbol.ClassScope {
	if scope == nil || strings.TrimSpace(scope.Superclass) == "" {
		return nil
	}
	base, _ := parseJavaTypeString(scope.Superclass)
	return resolveClassScopeByQualifiedName(ctx, base)
}

type methodResolution struct {
	def   *symbol.Definition
	owner *symbol.ClassScope
}

func findInstanceMethodInHierarchy(start *symbol.ClassScope, methodName string, argCount int, ctx Ctx) *methodResolution {
	seen := map[*symbol.ClassScope]struct{}{}
	for scope := start; scope != nil; scope = resolveSuperclassScope(ctx, scope) {
		if _, ok := seen[scope]; ok {
			return nil
		}
		seen[scope] = struct{}{}
		for _, def := range scope.Methods {
			if def == nil || def.IsStatic {
				continue
			}
			if def.OriginalName != methodName {
				continue
			}
			if len(def.Parameters) != argCount {
				continue
			}
			return &methodResolution{def: def, owner: scope}
		}
	}
	return nil
}

func findStaticMethodInHierarchy(start *symbol.ClassScope, methodName string, argCount int, ctx Ctx) *methodResolution {
	seen := map[*symbol.ClassScope]struct{}{}
	for scope := start; scope != nil; scope = resolveSuperclassScope(ctx, scope) {
		if _, ok := seen[scope]; ok {
			return nil
		}
		seen[scope] = struct{}{}
		for _, def := range scope.Methods {
			if def == nil || !def.IsStatic {
				continue
			}
			if def.OriginalName != methodName {
				continue
			}
			if len(def.Parameters) != argCount {
				continue
			}
			return &methodResolution{def: def, owner: scope}
		}
	}
	return nil
}

func findFieldInHierarchy(start *symbol.ClassScope, fieldName string, ctx Ctx) *symbol.Definition {
	seen := map[*symbol.ClassScope]struct{}{}
	for scope := start; scope != nil; scope = resolveSuperclassScope(ctx, scope) {
		if _, ok := seen[scope]; ok {
			return nil
		}
		seen[scope] = struct{}{}
		if field := scope.FindFieldByName(fieldName); field != nil {
			return field
		}
	}
	return nil
}

func mapClassTypeArgsToAncestor(child *symbol.ClassScope, childTypeArgs []ast.Expr, ancestor *symbol.ClassScope, ctx Ctx) []ast.Expr {
	if child == nil || ancestor == nil {
		return nil
	}
	if child == ancestor {
		return childTypeArgs
	}

	currentScope := child
	currentArgs := childTypeArgs
	seen := map[*symbol.ClassScope]struct{}{}

	for currentScope != nil && currentScope != ancestor {
		if _, ok := seen[currentScope]; ok {
			return nil
		}
		seen[currentScope] = struct{}{}

		superType := strings.TrimSpace(currentScope.Superclass)
		if superType == "" {
			return nil
		}

		base, superArgStrs := parseJavaTypeString(superType)
		parentScope := resolveClassScopeByQualifiedName(ctx, base)
		if parentScope == nil {
			return nil
		}

		// Map child's type parameters to its actual type arguments.
		paramNames := currentScope.TypeParameterNames()
		paramMap := make(map[string]ast.Expr, len(paramNames))
		for i, p := range paramNames {
			if i < len(currentArgs) {
				paramMap[p] = currentArgs[i]
			}
		}

		scopeTypeParams := append(inScopeTypeParameters(ctx), paramNames...)
		parentArgs := make([]ast.Expr, 0, len(superArgStrs))
		for _, a := range superArgStrs {
			a = strings.TrimSpace(stripJavaQualifier(a))
			if expr, ok := paramMap[a]; ok {
				parentArgs = append(parentArgs, expr)
				continue
			}
			parentArgs = append(parentArgs, javaTypeStringToGoTypeExpr(a, scopeTypeParams, ctx))
		}

		currentScope = parentScope
		currentArgs = parentArgs
	}

	if currentScope == ancestor {
		return currentArgs
	}
	return nil
}

func typeParamNameSet(typeParams []string) map[string]struct{} {
	if len(typeParams) == 0 {
		return nil
	}
	m := make(map[string]struct{}, len(typeParams))
	for _, tp := range typeParams {
		m[tp] = struct{}{}
	}
	return m
}

func findMatchingConstructor(scope *symbol.ClassScope, className string, argumentTypes []string) *symbol.Definition {
	if scope == nil {
		return nil
	}

	for _, def := range scope.Methods {
		if !def.Constructor {
			continue
		}
		if def.OriginalName != className {
			continue
		}
		if len(def.Parameters) != len(argumentTypes) {
			continue
		}

		// Allow type parameter positions (class or constructor type params) to match
		// any argument type, since the constructor can be instantiated accordingly.
		acceptedTypeParams := append([]string{}, scope.TypeParameterNames()...)
		acceptedTypeParams = append(acceptedTypeParams, def.TypeParameterNames()...)
		tpSet := typeParamNameSet(acceptedTypeParams)

		matches := true
		for i, param := range def.Parameters {
			argType := argumentTypes[i]
			if argType == "" {
				continue
			}
			if param.OriginalType == argType {
				continue
			}
			if tpSet != nil {
				if _, ok := tpSet[param.OriginalType]; ok {
					continue
				}
			}
			matches = false
			break
		}
		if matches {
			return def
		}
	}

	return nil
}

func findStaticMethodByNameAndArgCount(scope *symbol.ClassScope, methodName string, argCount int) *symbol.Definition {
	if scope == nil {
		return nil
	}
	for _, def := range scope.Methods {
		if !def.IsStatic {
			continue
		}
		if def.OriginalName != methodName {
			continue
		}
		if len(def.Parameters) != argCount {
			continue
		}
		return def
	}
	return nil
}

func findStaticMethodByName(scope *symbol.ClassScope, methodName string) *symbol.Definition {
	if scope == nil {
		return nil
	}
	for _, def := range scope.Methods {
		if def == nil || !def.IsStatic {
			continue
		}
		if def.OriginalName == methodName {
			return def
		}
	}
	return nil
}

func definitionParameterOriginalTypes(def *symbol.Definition) []string {
	if def == nil || len(def.Parameters) == 0 {
		return nil
	}
	types := make([]string, len(def.Parameters))
	for ind, param := range def.Parameters {
		if param == nil {
			continue
		}
		types[ind] = param.OriginalType
	}
	return types
}

func parseArgumentListWithExpectedTypes(argsNode *sitter.Node, source []byte, ctx Ctx, expectedTypes []string) []ast.Expr {
	if argsNode == nil {
		return nil
	}
	args := make([]ast.Expr, 0, argsNode.NamedChildCount())
	for ind, argNode := range nodeutil.NamedChildrenOf(argsNode) {
		argCtx := ctx.Clone()
		expectedType := ""
		if ind < len(expectedTypes) && strings.TrimSpace(expectedTypes[ind]) != "" {
			expectedType = expectedTypes[ind]
			argCtx.expectedType = expectedType
		}
		parsed := ParseExpr(argNode, source, argCtx)
		args = append(args, coerceArgumentToExpectedType(parsed, argNode, expectedType, ctx, source))
	}
	return args
}

func classInheritsFrom(child *symbol.ClassScope, expected *symbol.ClassScope, ctx Ctx) bool {
	if child == nil || expected == nil || child == expected {
		return false
	}
	seen := map[*symbol.ClassScope]struct{}{}
	for scope := child; scope != nil; scope = resolveSuperclassScope(ctx, scope) {
		if _, ok := seen[scope]; ok {
			return false
		}
		seen[scope] = struct{}{}
		if scope == expected {
			return true
		}
	}
	return false
}

func coerceArgumentToExpectedType(argExpr ast.Expr, argNode *sitter.Node, expectedType string, ctx Ctx, source []byte) ast.Expr {
	expectedType = strings.TrimSpace(expectedType)
	if expectedType == "" || argNode == nil || ctx.currentFile == nil {
		return argExpr
	}

	expectedBase, _ := parseJavaTypeString(expectedType)
	expectedScope := resolveClassScopeByQualifiedName(ctx, expectedBase)
	if expectedScope == nil || expectedScope.IsInterface || expectedScope.IsAbstract || expectedScope.Class == nil {
		return argExpr
	}

	actualType, ok := inferExprJavaType(argNode, ctx, source)
	if !ok || strings.TrimSpace(actualType) == "" {
		return argExpr
	}
	actualBase, _ := parseJavaTypeString(actualType)
	actualScope := resolveClassScopeByQualifiedName(ctx, actualBase)
	if actualScope == nil || actualScope == expectedScope {
		return argExpr
	}
	if !classInheritsFrom(actualScope, expectedScope, ctx) {
		return argExpr
	}

	return &ast.SelectorExpr{
		X:   argExpr,
		Sel: &ast.Ident{Name: expectedScope.Class.Name},
	}
}

func resolveFunctionalInterfaceMethod(ctx Ctx, expectedType string) (*symbol.Definition, map[string]string) {
	expectedType = strings.TrimSpace(expectedType)
	if expectedType == "" {
		return nil, nil
	}

	baseType, typeArgs := parseJavaTypeString(expectedType)
	if baseType == "" {
		return nil, nil
	}

	scope := resolveClassScopeByQualifiedName(ctx, baseType)
	if scope == nil {
		return nil, nil
	}
	if !scope.IsInterface {
		return nil, nil
	}
	if len(scope.ImplementedInterfaces) > 0 || strings.TrimSpace(scope.Superclass) != "" {
		return nil, nil
	}

	candidates := []*symbol.Definition{}
	for _, def := range scope.Methods {
		if def == nil || def.IsStatic || def.Constructor {
			continue
		}
		candidates = append(candidates, def)
	}

	// A lambda target must map to a single abstract method.
	if len(candidates) != 1 {
		return nil, nil
	}

	typeBindings := map[string]string{}
	for ind, typeParam := range scope.TypeParameters {
		if ind >= len(typeArgs) {
			break
		}
		bound := strings.TrimSpace(typeArgs[ind])
		if bound != "" {
			typeBindings[typeParam.Name] = bound
		}
	}

	return candidates[0], typeBindings
}

func substituteJavaTypeParams(typeStr string, bindings map[string]string) string {
	typeStr = strings.TrimSpace(typeStr)
	if typeStr == "" || len(bindings) == 0 {
		return typeStr
	}

	arraySuffix := ""
	for strings.HasSuffix(typeStr, "[]") {
		arraySuffix += "[]"
		typeStr = strings.TrimSpace(typeStr[:len(typeStr)-2])
	}

	if strings.HasPrefix(typeStr, "?") {
		rest := strings.TrimSpace(strings.TrimPrefix(typeStr, "?"))
		if rest == "" {
			return "?" + arraySuffix
		}
		if strings.HasPrefix(rest, "extends") {
			bound := strings.TrimSpace(strings.TrimPrefix(rest, "extends"))
			return "? extends " + substituteJavaTypeParams(bound, bindings) + arraySuffix
		}
		if strings.HasPrefix(rest, "super") {
			bound := strings.TrimSpace(strings.TrimPrefix(rest, "super"))
			return "? super " + substituteJavaTypeParams(bound, bindings) + arraySuffix
		}
		return typeStr + arraySuffix
	}

	baseType, typeArgs := parseJavaTypeString(typeStr)
	if len(typeArgs) == 0 {
		if replacement, exists := bindings[baseType]; exists {
			return replacement + arraySuffix
		}
		return typeStr + arraySuffix
	}

	mappedArgs := make([]string, len(typeArgs))
	for ind, arg := range typeArgs {
		mappedArgs[ind] = substituteJavaTypeParams(arg, bindings)
	}

	return fmt.Sprintf("%s<%s>%s", baseType, strings.Join(mappedArgs, ", "), arraySuffix)
}

func inferLambdaParameterTypeExprs(ctx Ctx, parameterCount int) []ast.Expr {
	if parameterCount <= 0 {
		return nil
	}

	method, typeBindings := resolveFunctionalInterfaceMethod(ctx, ctx.expectedType)
	if method == nil || len(method.Parameters) != parameterCount {
		return nil
	}

	inScopeParams := inScopeTypeParameters(ctx)
	types := make([]ast.Expr, len(method.Parameters))
	for ind, param := range method.Parameters {
		if param == nil {
			types[ind] = &ast.Ident{Name: "any"}
			continue
		}
		javaType := substituteJavaTypeParams(param.OriginalType, typeBindings)
		types[ind] = javaTypeStringToGoTypeExpr(javaType, inScopeParams, ctx)
	}
	return types
}

// objectCreationClassBody returns the class_body child of an object creation
// expression (present only for anonymous classes), or nil.
func objectCreationClassBody(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	for _, child := range nodeutil.NamedChildrenOf(node) {
		if child.Type() == "class_body" {
			return child
		}
	}
	return nil
}

// anonymousClassMethods returns the method declarations directly declared in an
// anonymous class body.
func anonymousClassMethods(classBody *sitter.Node) []*sitter.Node {
	var methods []*sitter.Node
	for _, child := range nodeutil.NamedChildrenOf(classBody) {
		if child.Type() == "method_declaration" {
			methods = append(methods, child)
		}
	}
	return methods
}

// lowerAnonymousClass lowers an anonymous class instantiation. When the
// supertype is a functional (single-abstract-method) interface and the body
// declares exactly that method, it is lowered to the interface's func adapter
// invoked with a closure that captures the surrounding locals. Returns nil when
// the anonymous class is not a SAM implementation (handled by Milestone 4).
func lowerAnonymousClass(node, objectType, classBody *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	if objectType == nil {
		return nil
	}
	supertype := objectType.Content(source)
	baseType, _ := parseJavaTypeString(supertype)

	interfaceScope := resolveClassScopeByQualifiedName(ctx, baseType)
	if interfaceScope == nil || !interfaceScope.IsInterface {
		return nil
	}

	// Identify the interface's single abstract method.
	method, _ := resolveFunctionalInterfaceMethod(ctx, supertype)
	if method == nil {
		return nil
	}

	// The anonymous class must implement exactly that one method for the SAM
	// lowering to be sound. Anything else is left for the synthesized-struct path.
	bodyMethods := anonymousClassMethods(classBody)
	if len(bodyMethods) != 1 {
		return nil
	}
	implMethod := bodyMethods[0]
	if implMethod.ChildByFieldName("name").Content(source) != method.OriginalName {
		return nil
	}

	// Build a function literal from the implementing method's signature and body.
	// The body is parsed in a scope that knows the method's parameters so captured
	// locals from the enclosing method resolve naturally as closure captures.
	implScope := scopeForAnonymousMethod(implMethod, method, source)
	methodCtx := ctx.Clone()
	methodCtx.localScope = implScope

	params := ParseNode(implMethod.ChildByFieldName("parameters"), source, methodCtx).(*ast.FieldList)

	var results *ast.FieldList
	if strings.TrimSpace(method.OriginalType) != "" && strings.TrimSpace(method.OriginalType) != "void" {
		results = &ast.FieldList{
			List: []*ast.Field{
				{Type: javaTypeStringToGoTypeExpr(method.OriginalType, inScopeTypeParameters(ctx), ctx)},
			},
		}
	}

	bodyNode := implMethod.ChildByFieldName("body")
	if bodyNode == nil {
		return nil
	}
	body := ParseStmt(bodyNode, source, methodCtx).(*ast.BlockStmt)

	funcLit := &ast.FuncLit{
		Type: &ast.FuncType{Params: params, Results: results},
		Body: body,
	}

	return wrapLambdaWithFunctionalInterfaceAdapter(funcLit, supertype, ctx)
}

// scopeForAnonymousMethod builds a local scope describing the parameters of an
// anonymous class's implementing method so its body parses with the right names
// and types.
func scopeForAnonymousMethod(implMethod *sitter.Node, samDef *symbol.Definition, source []byte) *symbol.Definition {
	scope := &symbol.Definition{
		OriginalName: samDef.OriginalName,
		Name:         samDef.Name,
		OriginalType: samDef.OriginalType,
	}
	paramsNode := implMethod.ChildByFieldName("parameters")
	for _, param := range nodeutil.NamedChildrenOf(paramsNode) {
		var nameNode, typeNode *sitter.Node
		if param.Type() == "spread_parameter" {
			nameNode = param.NamedChild(1).ChildByFieldName("name")
			typeNode = param.NamedChild(0)
		} else {
			nameNode = param.ChildByFieldName("name")
			typeNode = param.ChildByFieldName("type")
		}
		if nameNode == nil || typeNode == nil {
			continue
		}
		scope.Parameters = append(scope.Parameters, &symbol.Definition{
			OriginalName: nameNode.Content(source),
			Name:         nameNode.Content(source),
			OriginalType: typeNode.Content(source),
		})
	}
	return scope
}

func wrapLambdaWithFunctionalInterfaceAdapter(lambdaExpr ast.Expr, expectedType string, ctx Ctx) ast.Expr {
	method, _ := resolveFunctionalInterfaceMethod(ctx, expectedType)
	if method == nil {
		return nil
	}

	baseType, typeArgs := parseJavaTypeString(expectedType)
	interfaceScope := resolveClassScopeByQualifiedName(ctx, baseType)
	if interfaceScope == nil || interfaceScope.Class == nil || interfaceScope.Class.Name == "" {
		return nil
	}

	constructor := qualifiedNameExpr("New"+interfaceScope.Class.Name+"FuncAdapter", findJavaPackageForClassScope(interfaceScope), ctx)
	if len(typeArgs) > 0 {
		typeArgExprs := make([]ast.Expr, 0, len(typeArgs))
		for _, arg := range typeArgs {
			typeArgExprs = append(typeArgExprs, javaTypeStringToGoTypeExpr(arg, inScopeTypeParameters(ctx), ctx))
		}
		constructor = applyTypeArguments(constructor, typeArgExprs)
	}

	return &ast.CallExpr{
		Fun:  constructor,
		Args: []ast.Expr{lambdaExpr},
	}
}

func isPrimitiveCastTarget(target ast.Expr) bool {
	ident, ok := target.(*ast.Ident)
	if !ok || ident == nil {
		return false
	}

	switch ident.Name {
	case "bool", "int", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "byte", "rune", "float32", "float64":
		return true
	default:
		return false
	}
}

func instanceofAssertTypeExpr(javaType string, ctx Ctx) ast.Expr {
	javaType = strings.TrimSpace(javaType)
	if javaType == "" {
		return nil
	}

	baseType, typeArgs := parseJavaTypeString(javaType)
	inScopeParams := inScopeTypeParameters(ctx)

	if scope := resolveClassScopeByQualifiedName(ctx, baseType); scope != nil && scope.Class != nil {
		typeName := scope.Class.Name
		if typeName == "" {
			typeName = stripJavaQualifier(baseType)
		}

		baseExpr := qualifiedNameExpr(typeName, resolveJavaPackageForType(ctx, baseType, scope), ctx)
		if len(typeArgs) > 0 {
			typeArgExprs := make([]ast.Expr, 0, len(typeArgs))
			for _, arg := range typeArgs {
				typeArgExprs = append(typeArgExprs, javaTypeStringToGoTypeExpr(arg, inScopeParams, ctx))
			}
			baseExpr = applyTypeArguments(baseExpr, typeArgExprs)
		}

		if scope.IsInterface {
			return baseExpr
		}
		return &ast.StarExpr{X: baseExpr}
	}

	return javaTypeStringToGoTypeExpr(javaType, inScopeParams, ctx)
}

// maskedShiftAmount returns the Go expression for a Java shift count. Java masks
// the count to the low 5 bits for int shifts and the low 6 bits for long shifts
// before shifting, while Go applies the full count. For a constant decimal count
// we fold the mask at transpile time (e.g. `1 << 32` becomes `1 << 0`). Variable
// counts are left as-is: masking them would change the type of the shift and the
// common case (counts already in range) is unaffected.
func maskedShiftAmount(rightNode *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	if rightNode != nil && rightNode.Type() == "decimal_integer_literal" {
		literal := rightNode.Content(source)
		// Drop a trailing long suffix if present; the count itself is always int.
		trimmed := strings.TrimRight(literal, "lL")
		if value, ok := parseDecimalUint(trimmed); ok {
			masked := value & 31
			if masked != value {
				return &ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", masked)}
			}
		}
	}
	return ParseExpr(rightNode, source, ctx)
}

// parseDecimalUint parses a non-negative decimal integer string, ignoring Java
// digit separators ('_'). It reports false on any non-digit input.
func parseDecimalUint(s string) (uint64, bool) {
	if s == "" {
		return 0, false
	}
	var value uint64
	for _, r := range s {
		if r == '_' {
			continue
		}
		if r < '0' || r > '9' {
			return 0, false
		}
		value = value*10 + uint64(r-'0')
	}
	return value, true
}

func parseJavaTypeString(typeStr string) (string, []string) {
	typeStr = strings.TrimSpace(typeStr)
	if typeStr == "" {
		return "", nil
	}
	base := typeStr
	if idx := strings.Index(typeStr, "<"); idx >= 0 {
		base = strings.TrimSpace(typeStr[:idx])
	}
	return base, extractTypeArgsFromString(typeStr)
}

func stripJavaQualifier(typeName string) string {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return ""
	}
	// Tree-sitter (and symbol.OriginalType) can include package qualifiers like
	// "java.util.List<String>". The generator doesn't model Java packages as Go
	// packages, so drop the qualifier and keep the leaf type name.
	if idx := strings.LastIndex(typeName, "."); idx >= 0 {
		return typeName[idx+1:]
	}
	return typeName
}

func inScopeTypeParameters(ctx Ctx) []string {
	var params []string
	if ctx.currentClass != nil {
		params = append(params, ctx.currentClass.TypeParameterNames()...)
	}
	if ctx.localScope != nil {
		params = append(params, ctx.localScope.TypeParameterNames()...)
	}
	return params
}

// javaTypeStringToGoTypeExpr converts a Java type string (as it appears in
// symbol.OriginalType) into a Go AST expression suitable for use as a type
// argument in an IndexExpr/IndexListExpr. It mirrors astutil.ParseTypeWithTypeParams
// behavior for pointer-wrapping reference types, but operates on strings to support
// type inference paths.
func javaTypeStringToGoTypeExpr(typeStr string, typeParams []string, ctx Ctx) ast.Expr {
	typeStr = strings.TrimSpace(typeStr)
	if typeStr == "" {
		return &ast.Ident{Name: "any"}
	}

	// Arrays like Foo[][].
	arrayDims := 0
	for strings.HasSuffix(typeStr, "[]") {
		arrayDims++
		typeStr = strings.TrimSpace(typeStr[:len(typeStr)-2])
	}

	// Wildcards like ?, ? extends Foo, ? super Foo.
	if strings.HasPrefix(typeStr, "?") {
		rest := strings.TrimSpace(strings.TrimPrefix(typeStr, "?"))
		if rest == "" {
			return &ast.Ident{Name: "any"}
		}
		if strings.HasPrefix(rest, "extends") {
			bound := strings.TrimSpace(strings.TrimPrefix(rest, "extends"))
			if bound == "" {
				return &ast.Ident{Name: "any"}
			}
			return javaTypeStringToGoTypeExpr(bound, typeParams, ctx)
		}
		// ? super ... is hard to model faithfully in Go; fall back to any.
		return &ast.Ident{Name: "any"}
	}

	// Normalize qualifiers.
	base, typeArgs := parseJavaTypeString(typeStr)
	isInterface := false
	baseName := stripJavaQualifier(base)
	targetPkg := ""
	if resolvedScope := resolveClassScopeByQualifiedName(ctx, base); resolvedScope != nil && resolvedScope.Class != nil {
		isInterface = resolvedScope.IsInterface
		if resolvedScope.Class.Name != "" {
			baseName = resolvedScope.Class.Name
		}
		targetPkg = resolveJavaPackageForType(ctx, base, resolvedScope)
	} else if ctx.currentFile != nil {
		// If this type name maps to an import whose package we parsed in the same conversion run,
		// emit a qualified Go selector and add the corresponding import.
		if importedPkg, ok := ctx.currentFile.Imports[baseName]; ok && symbol.GlobalScope.FindPackage(importedPkg) != nil {
			targetPkg = importedPkg
		}
	}

	isTypeParam := func(name string) bool {
		for _, tp := range typeParams {
			if tp == name {
				return true
			}
		}
		return false
	}

	primitive := func(name string) (ast.Expr, bool) {
		switch name {
		case "String":
			return &ast.Ident{Name: "string"}, true
		// Boxed wrapper types map to their Go primitive. Java distinguishes a
		// boxed wrapper (nullable) from a primitive, but transpiled code uses the
		// value form; nullability is not modelled here. This makes boxed type
		// arguments (e.g. Box<Integer>, List<Integer>) and boxed declared types
		// resolve to a real Go type instead of an undefined *Integer.
		case "Integer":
			return &ast.Ident{Name: "int32"}, true
		case "Long":
			return &ast.Ident{Name: "int64"}, true
		case "Short":
			return &ast.Ident{Name: "int16"}, true
		case "Byte":
			return &ast.Ident{Name: "byte"}, true
		case "Character":
			return &ast.Ident{Name: "rune"}, true
		case "Float":
			return &ast.Ident{Name: "float32"}, true
		case "Double":
			return &ast.Ident{Name: "float64"}, true
		case "Boolean":
			return &ast.Ident{Name: "bool"}, true
		case "AutoCloseable":
			return &ast.InterfaceType{
				Methods: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{{Name: "Close"}},
							Type: &ast.FuncType{
								Params: &ast.FieldList{},
							},
						},
					},
				},
			}, true
		case "boolean":
			return &ast.Ident{Name: "bool"}, true
		case "int":
			return &ast.Ident{Name: "int32"}, true
		case "short":
			return &ast.Ident{Name: "int16"}, true
		case "long":
			return &ast.Ident{Name: "int64"}, true
		case "char":
			return &ast.Ident{Name: "rune"}, true
		case "byte":
			return &ast.Ident{Name: "byte"}, true
		case "float":
			return &ast.Ident{Name: "float32"}, true
		case "double":
			return &ast.Ident{Name: "float64"}, true
		}
		return nil, false
	}

	var expr ast.Expr
	if prim, ok := primitive(baseName); ok {
		expr = prim
	} else if isTypeParam(baseName) {
		expr = &ast.Ident{Name: baseName}
	} else {
		baseIdent := qualifiedNameExpr(baseName, targetPkg, ctx)
		if len(typeArgs) > 0 {
			argExprs := make([]ast.Expr, 0, len(typeArgs))
			for _, arg := range typeArgs {
				argExprs = append(argExprs, javaTypeStringToGoTypeExpr(arg, typeParams, ctx))
			}
			indexed := applyTypeArguments(baseIdent, argExprs)
			if isInterface {
				expr = indexed
			} else {
				expr = &ast.StarExpr{X: indexed}
			}
		} else {
			if isInterface {
				expr = baseIdent
			} else {
				expr = &ast.StarExpr{X: baseIdent}
			}
		}
	}

	for i := 0; i < arrayDims; i++ {
		expr = &ast.ArrayType{Elt: expr}
	}
	return expr
}

func inferIdentifierJavaType(name string, ctx Ctx) (string, bool) {
	if ctx.localScope != nil {
		if param := ctx.localScope.ParameterByName(name); param != nil && param.OriginalType != "" {
			return param.OriginalType, true
		}
		if local := ctx.localScope.FindVariable(name); local != nil && local.OriginalType != "" {
			return local.OriginalType, true
		}
	}
	if ctx.currentClass != nil {
		if field := findFieldInHierarchy(ctx.currentClass, name, ctx); field != nil && field.OriginalType != "" {
			return field.OriginalType, true
		}
	}
	return "", false
}

func inferExprJavaType(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	switch node.Type() {
	case "identifier":
		return inferIdentifierJavaType(node.Content(source), ctx)
	case "this":
		if ctx.currentClass == nil {
			return "", false
		}
		base := ctx.currentClass.Class.OriginalName
		if len(ctx.currentClass.TypeParameters) == 0 {
			return base, true
		}
		return fmt.Sprintf("%s<%s>", base, strings.Join(ctx.currentClass.TypeParameterNames(), ", ")), true
	case "object_creation_expression":
		typeNode := node.ChildByFieldName("type")
		if typeNode == nil {
			return "", false
		}
		return typeNode.Content(source), true
	case "string_literal":
		// A string literal is a java.lang.String, so chained calls on a literal
		// (e.g. "  x  ".trim()) resolve as String intrinsics.
		return "String", true
	case "parenthesized_expression":
		if inner := node.NamedChild(0); inner != nil {
			return inferExprJavaType(inner, ctx, source)
		}
	case "method_invocation":
		// Chained String intrinsics: if the inner call is itself a String method
		// that returns a String, the result type is String so the outer call also
		// resolves (e.g. s.trim().toUpperCase()).
		if objectNode := node.ChildByFieldName("object"); objectNode != nil {
			nameNode := node.ChildByFieldName("name")
			if nameNode != nil {
				methodName := nameNode.Content(source)
				// Character.toUpperCase/toLowerCase(char) return char.
				if objectNode.Type() == "identifier" && objectNode.Content(source) == "Character" {
					switch methodName {
					case "toUpperCase", "toLowerCase":
						return "char", true
					}
				}
				if recvType, ok := inferExprJavaType(objectNode, ctx, source); ok {
					base, _ := parseJavaTypeString(recvType)
					switch stripJavaQualifier(base) {
					case "String":
						switch {
						case methodName == "charAt":
							// String.charAt returns char.
							return "char", true
						case stringReturningStringMethods[methodName]:
							return "String", true
						case methodName == "split":
							// String.split returns String[].
							return "String[]", true
						}
					case "StringBuilder", "StringBuffer":
						if methodName == "charAt" {
							return "char", true
						}
					}
				}
			}
		}
	}
	return "", false
}

// isCharTypedExprNode reports whether the expression node evaluates to a Java
// char. Used so that printing a char emits its glyph rather than its code point.
func isCharTypedExprNode(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil {
		return false
	}
	if node.Type() == "character_literal" {
		return true
	}
	if javaType, ok := inferExprJavaType(node, ctx, source); ok {
		base, _ := parseJavaTypeString(javaType)
		switch stripJavaQualifier(base) {
		case "char", "Character":
			return true
		}
	}
	return false
}

// isArrayTypedExprNode reports whether the expression node is known to evaluate
// to a Java array, so that `expr.length` can be lowered to len().
func isArrayTypedExprNode(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil {
		return false
	}
	if node.Type() == "array_creation_expression" {
		return true
	}
	if javaType, ok := inferExprJavaType(node, ctx, source); ok {
		return strings.HasSuffix(strings.TrimSpace(javaType), "[]")
	}
	return false
}

// stringReturningStringMethods lists the String instance methods whose result is
// itself a String, so chained intrinsic calls infer their receiver type.
var stringReturningStringMethods = map[string]bool{
	"substring":   true,
	"toUpperCase": true,
	"toLowerCase": true,
	"trim":        true,
	"strip":       true,
	"replace":     true,
	"concat":      true,
}

func superSelectorExpr(ctx Ctx) ast.Expr {
	if ctx.currentClass == nil {
		return &ast.BadExpr{}
	}
	superType := strings.TrimSpace(ctx.currentClass.Superclass)
	if superType == "" {
		return &ast.BadExpr{}
	}
	base, _ := parseJavaTypeString(superType)
	if base == "" {
		return &ast.BadExpr{}
	}
	superName := stripJavaQualifier(base)
	if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil && scope.Class != nil && scope.Class.Name != "" {
		superName = scope.Class.Name
	}
	recvName := ctx.className
	if recvName == "" && ctx.currentClass.Class != nil {
		recvName = ctx.currentClass.Class.Name
	}
	if recvName == "" {
		return &ast.BadExpr{}
	}
	return &ast.SelectorExpr{
		X:   &ast.Ident{Name: ShortName(recvName)},
		Sel: &ast.Ident{Name: superName},
	}
}

func applyTypeArguments(fun ast.Expr, args []ast.Expr) ast.Expr {
	if len(args) == 0 {
		return fun
	}
	if len(args) == 1 {
		return &ast.IndexExpr{X: fun, Index: args[0]}
	}
	return &ast.IndexListExpr{X: fun, Indices: args}
}

type invocationTargetInfo struct {
	classScope    *symbol.ClassScope
	classTypeArgs []ast.Expr
}

func resolveInvocationTarget(objectNode *sitter.Node, ctx Ctx, source []byte) *invocationTargetInfo {
	if ctx.currentFile == nil {
		return nil
	}

	scopeTypeParams := inScopeTypeParameters(ctx)

	var className string
	var classTypeArgs []string
	switch objectNode.Type() {
	case "this":
		if ctx.currentClass == nil {
			return nil
		}
		className = ctx.currentClass.Class.OriginalName
		classTypeArgs = ctx.currentClass.TypeParameterNames()
	case "super":
		if ctx.currentClass == nil {
			return nil
		}
		superType := strings.TrimSpace(ctx.currentClass.Superclass)
		if superType == "" {
			return nil
		}
		className, classTypeArgs = parseJavaTypeString(superType)
	case "identifier":
		javaType, ok := inferIdentifierJavaType(objectNode.Content(source), ctx)
		if !ok {
			return nil
		}
		className, classTypeArgs = parseJavaTypeString(javaType)
	default:
		javaType, ok := inferExprJavaType(objectNode, ctx, source)
		if !ok {
			return nil
		}
		className, classTypeArgs = parseJavaTypeString(javaType)
	}

	classScope := resolveClassScopeByQualifiedName(ctx, className)
	if classScope == nil {
		return nil
	}

	classTypeArgExprs := make([]ast.Expr, 0, len(classTypeArgs))
	for _, arg := range classTypeArgs {
		classTypeArgExprs = append(classTypeArgExprs, javaTypeStringToGoTypeExpr(arg, scopeTypeParams, ctx))
	}

	return &invocationTargetInfo{
		classScope:    classScope,
		classTypeArgs: classTypeArgExprs,
	}
}

func explicitTypeArgumentExprs(node *sitter.Node, source []byte, typeParams []string, ctx Ctx) []ast.Expr {
	typeArgsNode := node.ChildByFieldName("type_arguments")
	if typeArgsNode == nil {
		return nil
	}
	var exprs []ast.Expr
	for _, arg := range nodeutil.NamedChildrenOf(typeArgsNode) {
		exprs = append(exprs, javaTypeStringToGoTypeExpr(arg.Content(source), typeParams, ctx))
	}
	return exprs
}

func inferMethodTypeArguments(def *symbol.Definition, invocationNode *sitter.Node, ctx Ctx, source []byte) []ast.Expr {
	if len(def.TypeParameters) == 0 {
		return nil
	}

	if explicit := explicitTypeArgumentExprs(invocationNode, source, inScopeTypeParameters(ctx), ctx); len(explicit) == len(def.TypeParameters) && len(explicit) > 0 {
		return explicit
	}

	argsNode := invocationNode.ChildByFieldName("arguments")
	if argsNode == nil {
		return nil
	}

	resolved := make(map[string]ast.Expr)
	argNodes := nodeutil.NamedChildrenOf(argsNode)
	for idx, param := range def.Parameters {
		for _, tp := range def.TypeParameters {
			if param.OriginalType == tp.Name && idx < len(argNodes) {
				if javaType, ok := inferExprJavaType(argNodes[idx], ctx, source); ok {
					resolved[tp.Name] = javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx)
				}
			}
		}
	}

	result := make([]ast.Expr, len(def.TypeParameters))
	for i, tp := range def.TypeParameters {
		if expr, ok := resolved[tp.Name]; ok {
			result[i] = expr
		} else {
			result[i] = &ast.Ident{Name: "any"}
		}
	}
	return result
}

func maybeRewriteInstanceGenericMethodInvocationWithTarget(target *invocationTargetInfo, objectExpr ast.Expr, methodName string, args []ast.Expr, invocationNode *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	if target == nil {
		return nil
	}

	resolved := findInstanceMethodInHierarchy(target.classScope, methodName, len(args), ctx)
	if resolved == nil || resolved.def == nil || !resolved.def.RequiresHelper {
		return nil
	}
	helperDef := resolved.def
	ownerScope := resolved.owner

	receiverExpr := objectExpr
	classTypeArgs := target.classTypeArgs
	if ownerScope != nil && ownerScope != target.classScope {
		receiverExpr = &ast.SelectorExpr{X: objectExpr, Sel: &ast.Ident{Name: ownerScope.Class.Name}}
		if mapped := mapClassTypeArgsToAncestor(target.classScope, target.classTypeArgs, ownerScope, ctx); mapped != nil {
			classTypeArgs = mapped
		}
	}

	methodTypeArgs := inferMethodTypeArguments(helperDef, invocationNode, ctx, source)
	helperTypeArgs := append(classTypeArgs, methodTypeArgs...)

	helperPkg := findJavaPackageForClassScope(ownerScope)
	if helperPkg == "" {
		helperPkg = findJavaPackageForClassScope(target.classScope)
	}
	constructorExpr := qualifiedNameExpr("New"+helperDef.HelperName, helperPkg, ctx)
	helperConstructor := applyTypeArguments(constructorExpr, helperTypeArgs)
	helperCall := &ast.CallExpr{
		Fun:  helperConstructor,
		Args: []ast.Expr{receiverExpr},
	}

	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   helperCall,
			Sel: &ast.Ident{Name: helperDef.Name},
		},
		Args: args,
	}
}
