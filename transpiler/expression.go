package transpiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
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
		if lowered, ok := lowerStaticFieldUpdate(node, operandNode, helper, source, ctx); ok {
			return lowered
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
		return lowerAssignmentExpression(node, source, ctx)
	case "super":
		return superSelectorExpr(ctx)
	case "switch_expression":
		// A switch used as a value (Java 14+) is lowered to an immediately-invoked
		// function literal whose arms `return` the produced value. yield X and
		// arrow-form `case X -> expr` both become `return X`.
		return buildSwitchExpressionIIFE(node, source, ctx)
	case "lambda_expression":
		// Lambdas can either be called with a list of expressions
		// (ex: (n1, n1) -> {}), or with a single expression
		// (ex: n1 -> {})
		samMethod, samTypeBindings := resolveFunctionalInterfaceMethod(ctx, ctx.expectedType)
		var lambdaResults *ast.FieldList
		lambdaReturnType := "void"

		if samMethod != nil && strings.TrimSpace(samMethod.OriginalType) != "" && strings.TrimSpace(samMethod.OriginalType) != "void" {
			resultType := substituteJavaTypeParams(samMethod.OriginalType, samTypeBindings)
			lambdaReturnType = resultType
			lambdaResults = &ast.FieldList{
				List: []*ast.Field{
					{Type: javaTypeStringToGoTypeExpr(resultType, inScopeTypeParameters(ctx), ctx)},
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
		inferredParamJavaTypes := inferLambdaParameterJavaTypes(ctx, paramCount)
		inferredParamTypes := inferLambdaParameterTypeExprs(ctx, paramCount)
		lambdaCtx := contextWithLambdaParameters(ctx, paramNode, inferredParamJavaTypes, lambdaReturnType, source)
		expectedBase, _ := parseJavaTypeString(ctx.expectedType)
		expectedScope := resolveClassScopeByQualifiedName(ctx, expectedBase)
		executionAwareSAM := samMethod != nil && expectedScope != nil && expectedScope.IsInterface
		executionAwareRunnable := samMethod == nil && expectedScope == nil && stripJavaQualifier(expectedBase) == "Runnable"

		var lambdaParameters *ast.FieldList
		switch paramNode.Type() {
		case "formal_parameters":
			lambdaParameters = ParseNode(paramNode, source, lambdaCtx).(*ast.FieldList)
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
		if executionAwareSAM || executionAwareRunnable {
			executionName := executionParameterName(node, source, lambdaCtx)
			lambdaCtx.executionContextName = executionName
			lambdaParameters.List = append(
				[]*ast.Field{executionParameterField(executionName, ctx)},
				lambdaParameters.List...,
			)
		}

		// Parse the body only after the inferred SAM parameters have been added to
		// the local symbol context. The generated signature alone is not enough:
		// method and intrinsic resolution operates on the Java tree while parsing.
		var lambdaBody *ast.BlockStmt
		bodyNode := node.ChildByFieldName("body")
		switch bodyNode.Type() {
		case "block":
			lambdaBody = ParseStmt(bodyNode, source, lambdaCtx).(*ast.BlockStmt)
		default:
			// Lambdas can be called inline without a block expression
			inlineExpr := ParseExpr(bodyNode, source, lambdaCtx)
			inlineStmt := ast.Stmt(&ast.ExprStmt{X: inlineExpr})
			if lambdaResults != nil && len(lambdaResults.List) > 0 {
				inlineStmt = &ast.ReturnStmt{Results: []ast.Expr{inlineExpr}}
			}
			lambdaBody = &ast.BlockStmt{List: []ast.Stmt{inlineStmt}}
		}
		if lambdaResults != nil && len(lambdaResults.List) > 0 && bodyNeedsFallbackReturn(lambdaBody) {
			lambdaBody.List = append(lambdaBody.List, &ast.ReturnStmt{
				Results: []ast.Expr{zeroValueForType(lambdaResults.List[0].Type)},
			})
		}

		lambdaFunc := &ast.FuncLit{
			Type: &ast.FuncType{
				Params:  lambdaParameters,
				Results: lambdaResults,
			},
			Body: lambdaBody,
		}

		if adapted := wrapLambdaWithFunctionalInterfaceAdapter(lambdaFunc, ctx.expectedType, executionAwareSAM, ctx); adapted != nil {
			return adapted
		}

		return lambdaFunc
	case "method_reference":
		// This refers to manually selecting a function from a specific class and
		// passing it in as an argument in the `func(className::methodName)` style

		samMethod, samBindings := resolveFunctionalInterfaceMethod(ctx, ctx.expectedType)
		if samMethod == nil && isExternalRunnableType(ctx.expectedType, ctx) {
			// java.lang.Runnable is supplied by stdjava rather than the parsed
			// symbol graph. Model its zero-argument void SAM here so bound method
			// references select the execution-aware implementation just like
			// source-declared functional interfaces do.
			samMethod = &symbol.Definition{OriginalName: "run", Name: "Run", OriginalType: "void"}
		}
		samType, samExecutionName := executionAwareSAMFuncType(node, samMethod, samBindings, source, ctx)

		// For class constructors such as `Class::new`, you only get one node.
		if node.NamedChildCount() < 2 {
			targetNode := node.NamedChild(0)
			if targetNode != nil && samMethod != nil {
				if targetScope := resolveClassScopeByQualifiedName(ctx, targetNode.Content(source)); targetScope != nil && targetScope.Class != nil {
					constructorName := constructorFuncName(targetScope)
					var constructor *symbol.Definition
					for _, candidate := range targetScope.Methods {
						if candidate != nil && candidate.Constructor && len(candidate.Parameters) == len(samMethod.Parameters) {
							constructor = candidate
							constructorName = candidate.Name
							break
						}
					}
					if constructorName == "" {
						constructorName = defaultConstructorName(targetScope.Class.Name)
					}
					// A missing constructor definition selects the synthesized default
					// constructor, which is execution-aware. A present synthetic
					// definition (notably a record canonical constructor) is public-only.
					executionAware := constructorHasExecutionImplementation(constructor, targetScope)
					if executionAware {
						constructorName = executionConstructorImplementationName(constructorName, targetScope)
					}
					constructorRef := qualifiedNameExpr(constructorName, findJavaPackageForClassScope(targetScope), ctx)
					if adapted := wrapLambdaWithFunctionalInterfaceAdapter(constructorRef, ctx.expectedType, executionAware, ctx); adapted != nil {
						return adapted
					}
				}
			}
			constructorRef := ParseExpr(targetNode, source, ctx)
			if adapted := wrapLambdaWithFunctionalInterfaceAdapter(constructorRef, ctx.expectedType, false, ctx); adapted != nil {
				return adapted
			}
			return constructorRef
		}

		targetNode := node.NamedChild(0)
		methodName := node.NamedChild(1).Content(source)

		if classScope := resolveClassScopeByIdentifier(ctx, source, targetNode); classScope != nil {
			if staticDef := findStaticMethodByName(classScope, methodName); staticDef != nil {
				staticPkg := resolveJavaPackageForType(ctx, targetNode.Content(source), classScope)
				methodName := staticDef.Name
				executionAware := staticDef.DeclarationNode != nil
				if executionAware {
					methodName = executionImplementationName(staticDef, classScope)
				}
				methodExpr := qualifiedNameExpr(methodName, staticPkg, ctx)
				if adapted := wrapLambdaWithFunctionalInterfaceAdapter(methodExpr, ctx.expectedType, executionAware, ctx); adapted != nil {
					return adapted
				}
				return methodExpr
			}
			if samMethod != nil && samType != nil && len(samMethod.Parameters) > 0 {
				if resolution := findInstanceMethodInHierarchy(classScope, methodName, len(samMethod.Parameters)-1, ctx); resolution != nil && resolution.def != nil && resolution.def.DeclarationNode != nil {
					methodExpr := executionAwareMethodReferenceForwarder(nil, resolution, &invocationTargetInfo{classScope: classScope}, samType, samExecutionName, true, node, source, ctx)
					if adapted := wrapLambdaWithFunctionalInterfaceAdapter(methodExpr, ctx.expectedType, true, ctx); adapted != nil {
						return adapted
					}
				}
			}
		}

		methodExpr := ast.Expr(&ast.SelectorExpr{X: ParseExpr(targetNode, source, ctx), Sel: identFromNode(node.NamedChild(1), source)})
		if samMethod != nil && samType != nil {
			if target := resolveInvocationTarget(targetNode, ctx, source); target != nil && target.classScope != nil {
				if resolution := findInstanceMethodInHierarchy(target.classScope, methodName, len(samMethod.Parameters), ctx); resolution != nil && resolution.def != nil && resolution.def.DeclarationNode != nil {
					receiver := ParseExpr(targetNode, source, ctx)
					if target.classScope.IsInterface || target.classScope.IsAbstract {
						methodExpr = executionAwareMethodReferenceForwarder(receiver, resolution, target, samType, samExecutionName, false, node, source, ctx)
					} else {
						selectorBase := receiver
						if classNeedsVirtualDispatch(resolution.owner, ctx) {
							selectorBase = &ast.SelectorExpr{X: receiver, Sel: &ast.Ident{Name: classDispatchFieldName(resolution.owner)}}
						}
						methodExpr = &ast.SelectorExpr{X: selectorBase, Sel: &ast.Ident{Name: executionImplementationName(resolution.def, resolution.owner)}}
					}
					if adapted := wrapLambdaWithFunctionalInterfaceAdapter(methodExpr, ctx.expectedType, true, ctx); adapted != nil {
						return adapted
					}
				}
			}
		}

		if adapted := wrapLambdaWithFunctionalInterfaceAdapter(methodExpr, ctx.expectedType, false, ctx); adapted != nil {
			return adapted
		}

		return methodExpr
	case "array_initializer":
		// A literal that initilzes an array, such as `{1, 2, 3}`
		items := []ast.Expr{}
		reifiedComponent, reifiedComponentType, reifiedComponentID, reified := reifiedReferenceArrayComponentInfo(ctx.expectedType, ctx)
		primitiveComponent, primitiveArray := javaPrimitiveArrayComponent(ctx.expectedType)
		var primitiveComponentType ast.Expr
		var primitiveComponentID ast.Expr
		if primitiveArray {
			primitiveComponentType = javaTypeStringToGoTypeExpr(primitiveComponent, inScopeTypeParameters(ctx), ctx)
			primitiveComponentID, _ = javaPrimitiveTypeIDExpr(primitiveComponent, ctx)
		}
		arrayType, hasArrayType := ctx.lastType.(*ast.ArrayType)
		if !hasArrayType && strings.HasSuffix(strings.TrimSpace(ctx.expectedType), "[]") {
			inferredType := javaTypeStringToGoTypeExpr(ctx.expectedType, inScopeTypeParameters(ctx), ctx)
			arrayType, hasArrayType = inferredType.(*ast.ArrayType)
		}
		expectedElementType := strings.TrimSpace(ctx.expectedType)
		if strings.HasSuffix(expectedElementType, "[]") {
			expectedElementType = strings.TrimSpace(strings.TrimSuffix(expectedElementType, "[]"))
		}
		if reified {
			expectedElementType = reifiedComponent
		} else if primitiveArray {
			expectedElementType = primitiveComponent
		}
		for _, c := range nodeutil.NamedChildrenOf(node) {
			itemCtx := ctx.Clone()
			if hasArrayType {
				// Nested Java initializers omit their inner type. Carry the outer
				// component type down so each row is allocated through ArrayLiteral
				// with the correct generic element type.
				itemCtx.lastType = arrayType.Elt
			}
			itemCtx.expectedType = expectedElementType
			itemCtx.expectedTypeRoot = c
			item := ParseExpr(c, source, itemCtx)
			item = coerceArgumentToExpectedType(item, c, expectedElementType, ctx, source)
			item = boxPrimitiveForObject(item, c, expectedElementType, ctx, source)
			items = append(items, item)
		}

		// A source reference array retains its runtime component independently of
		// any covariant target type. Values remain ordinary generated object views;
		// ReferenceArray performs the nominal store checks.
		if reified {
			args := append([]ast.Expr{reifiedComponentID}, items...)
			return stdjavaGenericCall(ctx, "ReferenceArrayLiteralOf", []ast.Expr{reifiedComponentType}, args)
		}
		if primitiveArray {
			args := append([]ast.Expr{primitiveComponentID}, items...)
			return stdjavaGenericCall(ctx, "PrimitiveArrayLiteral", []ast.Expr{primitiveComponentType}, args)
		}

		// ArrayLiteral retains allocation identity for an empty initializer while
		// preserving left-to-right item evaluation and the exact inferred type.
		if hasArrayType {
			return stdjavaGenericCall(ctx, "ArrayLiteral", []ast.Expr{arrayType.Elt}, items)
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
				// PrintStream performs Java text conversion before writing. Route
				// every non-char value through the shared runtime bridge so null and
				// floating-point values retain their Java spellings; chars need their
				// static type here because Go represents both char and int as int32.
				if argListNode != nil {
					for ind, argNode := range nodeutil.NamedChildrenOf(argListNode) {
						if ind < len(args) {
							args[ind] = javaStringConversionExpr(argNode, args[ind], ctx, source)
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

			// Throwable.getCause()/getMessage()/getSuppressed()/printStackTrace() on a caught exception are
			// routed through the stdjava runtime, which understands both the
			// built-in exception types and user-defined ones.
			if argCount == 0 && (methodName == "getCause" || methodName == "getMessage" || methodName == "getSuppressed" || methodName == "printStackTrace") {
				if javaType, ok := inferExprJavaType(objectNode, ctx, source); ok && isExceptionJavaType(ctx, javaType) {
					receiver := ParseExpr(objectNode, source, ctx)
					if methodName == "getSuppressed" {
						return stdjavaCall(ctx, "SuppressedArray", stdjavaCall(ctx, "GetSuppressed", receiver))
					}
					runtimeFn := "GetMessage"
					switch methodName {
					case "getCause":
						runtimeFn = "GetCause"
					case "printStackTrace":
						runtimeFn = "PrintStackTrace"
					}
					return &ast.CallExpr{
						Fun:  stdjavaQualifiedExpr(runtimeFn, ctx),
						Args: []ast.Expr{receiver},
					}
				}
			}

			// Thread methods on a `class X extends Thread` subclass dispatch to the
			// embedded *stdjava.Thread, whose methods use Go's exported casing.
			if argCount == 0 {
				if goMethod, ok := threadSubclassMethod(objectNode, methodName, ctx, source); ok {
					return &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ParseExpr(objectNode, source, ctx),
							Sel: &ast.Ident{Name: goMethod},
						},
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
			target := resolveInvocationTarget(objectNode, ctx, source)
			instanceResolution := (*methodResolution)(nil)
			staticResolution := (*methodResolution)(nil)

			if classScope != nil {
				// A class-qualified invocation (Utility.parse(...)) can only target a
				// static overload. Select it using the arguments' Java types rather than
				// whichever declaration happened to appear first in the source file.
				staticResolution = findBestMethodInHierarchy(classScope, methodName, argListNode, false, true, ctx, source)
			} else if target != nil {
				// Java permits a static method to be selected through an expression as
				// well as normal instance dispatch, so consider the complete overload set.
				selected := findBestMethodInHierarchy(target.classScope, methodName, argListNode, true, true, ctx, source)
				if selected != nil && selected.def != nil && selected.def.IsStatic {
					staticResolution = selected
				} else {
					instanceResolution = selected
				}
			}
			if instanceResolution != nil {
				expectedArgTypes = definitionParameterOriginalTypes(instanceResolution.def)
				if genericArrayFormalNeedsExplicitTypeArguments(instanceResolution.def) {
					expectedArgTypes = genericArrayInvocationExpectedTypes(instanceResolution.def, node, ctx, source)
				}
			} else if staticResolution != nil {
				expectedArgTypes = definitionParameterOriginalTypes(staticResolution.def)
				if genericArrayFormalNeedsExplicitTypeArguments(staticResolution.def) {
					expectedArgTypes = genericArrayInvocationExpectedTypes(staticResolution.def, node, ctx, source)
				}
			}
			selectedResolution := instanceResolution
			if selectedResolution == nil {
				selectedResolution = staticResolution
			}
			args, expandVarargsArray := parseResolvedInvocationArguments(selectedResolution, argListNode, source, ctx, expectedArgTypes)
			typeArgs := explicitTypeArgumentExprs(node, source, inScopeTypeParameters(ctx), ctx)
			if len(typeArgs) == 0 && staticResolution != nil && genericArrayFormalNeedsExplicitTypeArguments(staticResolution.def) {
				typeArgs = inferMethodTypeArguments(staticResolution.def, node, ctx, source)
			}

			// If this is a static call on a class name (e.g., Utils.<T>id(...)),
			// rewrite it to a plain function call to match how static methods are emitted.
			if classScope != nil && staticResolution != nil {
				staticPkg := findJavaPackageForClassScope(staticResolution.owner)
				fun := qualifiedNameExpr(executionMethodCallName(staticResolution.def, staticResolution.owner, ctx), staticPkg, ctx)
				fun = applyTypeArguments(fun, typeArgs)
				return markDirectVarargsExpansion(
					&ast.CallExpr{Fun: fun, Args: prependExecutionMethodArgument(ctx, staticResolution.def, args)},
					expandVarargsArray,
				)
			}

			if !expandVarargsArray {
				if rewritten := rewriteAffineArrayAccessorInvocation(node, objectNode, objectExpr, target, instanceResolution, args, ctx, source); rewritten != nil {
					return rewritten
				}
			}

			if rewritten := maybeRewriteInstanceGenericMethodInvocationWithTarget(target, instanceResolution, objectExpr, methodName, args, node, ctx, source); rewritten != nil {
				markDirectVarargsExpansionExpr(rewritten, expandVarargsArray)
				return rewritten
			}

			if target != nil {
				if instanceResolution != nil {
					methodIdent = &ast.Ident{Name: executionMethodCallName(instanceResolution.def, instanceResolution.owner, ctx)}
				} else if staticResolution != nil {
					// Java permits calling static methods via an instance expression; rewrite
					// to a plain function call to match codegen. The qualifying expression is
					// still evaluated before the arguments, even though its value is ignored.
					fun := qualifiedNameExpr(executionMethodCallName(staticResolution.def, staticResolution.owner, ctx), findJavaPackageForClassScope(staticResolution.owner), ctx)
					fun = applyTypeArguments(fun, typeArgs)
					call := markDirectVarargsExpansion(
						&ast.CallExpr{Fun: fun, Args: prependExecutionMethodArgument(ctx, staticResolution.def, args)},
						expandVarargsArray,
					)
					if staged := stageStaticInvocationQualifier(node, objectExpr, staticResolution, call, ctx, source); staged != nil {
						return staged
					}
					return call
				}
			}

			// Abstract Java reference types are emitted as companion Go interfaces.
			// Calling through that interface already performs dynamic dispatch and it
			// has no concrete class dispatch field to select.
			abstractInterfaceReceiver := target != nil && target.classScope != nil && target.classScope.IsAbstract
			companionInterfaceReceiver := target != nil && target.classScope != nil &&
				(target.classScope.IsInterface || target.classScope.IsAbstract)
			if companionInterfaceReceiver && instanceResolution != nil && executionExpr(ctx) != nil {
				if dispatched := executionCompanionDispatchInvocation(
					node, objectNode, objectExpr, target, instanceResolution, args, expandVarargsArray, ctx, source,
				); dispatched != nil {
					return dispatched
				}
			}
			if objectNode.Type() != "super" && !abstractInterfaceReceiver {
				if dispatched := virtualDispatchMethodCall(objectExpr, instanceResolution, args, expandVarargsArray, ctx); dispatched != nil {
					buildDispatch := func(receiver ast.Expr, callArgs []ast.Expr) ast.Expr {
						return virtualDispatchMethodCall(receiver, instanceResolution, callArgs, expandVarargsArray, ctx)
					}
					if staged := stageVirtualDispatchInvocation(node, objectNode, objectExpr, instanceResolution, args, buildDispatch, ctx, source); staged != nil {
						return staged
					}
					return dispatched
				}
			}

			callArgs := args
			if instanceResolution != nil && !companionInterfaceReceiver {
				callArgs = prependExecutionMethodArgument(ctx, instanceResolution.def, callArgs)
			}
			if instanceResolution != nil && companionInterfaceReceiver {
				methodIdent = &ast.Ident{Name: instanceResolution.def.Name}
			}
			return markDirectVarargsExpansion(&ast.CallExpr{
				Fun:  &ast.SelectorExpr{X: objectExpr, Sel: methodIdent},
				Args: callArgs,
			}, expandVarargsArray)
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
			allowInstance := ctx.localScope != nil && ctx.localScope.OriginalName != "" && !ctx.localScope.IsStatic
			selected := findBestMethodInHierarchy(ctx.currentClass, methodName, argListNode, allowInstance, true, ctx, source)
			if selected != nil && selected.def != nil && selected.def.IsStatic {
				implicitStaticResolution = selected
			} else {
				implicitInstanceResolution = selected
			}
			if selected != nil {
				expectedArgTypes = definitionParameterOriginalTypes(selected.def)
			}

			// A nested class may invoke a static member of any lexically enclosing
			// class without qualification. Static nested classes do not carry an
			// enclosing-instance field, but the lexical lookup still applies. Resolve
			// that owner here so execution-aware calls can forward the current token
			// instead of falling through to the public fresh-token wrapper.
			if implicitInstanceResolution == nil && implicitStaticResolution == nil {
				for enclosing := ctx.currentClass.Enclosing; enclosing != nil; enclosing = enclosing.Enclosing {
					selected = findBestMethodInHierarchy(enclosing, methodName, argListNode, false, true, ctx, source)
					if selected != nil && selected.def != nil && selected.def.IsStatic {
						implicitStaticResolution = selected
						expectedArgTypes = definitionParameterOriginalTypes(selected.def)
						break
					}
				}
			}
		}
		if implicitInstanceResolution != nil && genericArrayFormalNeedsExplicitTypeArguments(implicitInstanceResolution.def) {
			expectedArgTypes = genericArrayInvocationExpectedTypes(implicitInstanceResolution.def, node, ctx, source)
		} else if implicitStaticResolution != nil && genericArrayFormalNeedsExplicitTypeArguments(implicitStaticResolution.def) {
			expectedArgTypes = genericArrayInvocationExpectedTypes(implicitStaticResolution.def, node, ctx, source)
		}
		selectedResolution := implicitInstanceResolution
		if selectedResolution == nil {
			selectedResolution = implicitStaticResolution
		}
		args, expandVarargsArray := parseResolvedInvocationArguments(selectedResolution, argListNode, source, ctx, expectedArgTypes)
		typeArgs := explicitTypeArgumentExprs(node, source, inScopeTypeParameters(ctx), ctx)
		if len(typeArgs) == 0 && implicitStaticResolution != nil && genericArrayFormalNeedsExplicitTypeArguments(implicitStaticResolution.def) {
			typeArgs = inferMethodTypeArguments(implicitStaticResolution.def, node, ctx, source)
		}

		// Unqualified invocation in Java is typically an implicit receiver call.
		// Only do this in a non-static method/constructor body where the receiver
		// variable exists.
		if ctx.currentClass != nil && ctx.localScope != nil && ctx.localScope.OriginalName != "" && !ctx.localScope.IsStatic {
			recv := &ast.Ident{Name: ShortName(ctx.className)}
			target := &invocationTargetInfo{
				classScope:    ctx.currentClass,
				classTypeArgs: typeParamExprs(ctx.currentClass.TypeParameterNames()),
			}
			if rewritten := maybeRewriteInstanceGenericMethodInvocationWithTarget(target, implicitInstanceResolution, recv, methodName, args, node, ctx, source); rewritten != nil {
				markDirectVarargsExpansionExpr(rewritten, expandVarargsArray)
				return rewritten
			}
			if implicitInstanceResolution != nil {
				if dispatched := virtualDispatchMethodCall(recv, implicitInstanceResolution, args, expandVarargsArray, ctx); dispatched != nil {
					return dispatched
				}
				return markDirectVarargsExpansion(&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   recv,
						Sel: &ast.Ident{Name: executionMethodCallName(implicitInstanceResolution.def, implicitInstanceResolution.owner, ctx)},
					},
					Args: prependExecutionMethodArgument(ctx, implicitInstanceResolution.def, args),
				}, expandVarargsArray)
			}
			// Unqualified call to an enclosing class's instance method from inside
			// an inner class: route it through the enclosing-instance field, e.g.
			// foo() -> or.outer.Foo().
			if enclSel, resolution := enclosingMemberMethodSelector(methodName, argCount, ctx); enclSel != nil {
				return &ast.CallExpr{Fun: enclSel, Args: prependExecutionMethodArgument(ctx, resolution.def, args)}
			}
		}

		// Otherwise, treat as a plain function call (static methods are emitted as
		// functions).
		if ctx.currentClass != nil {
			if implicitStaticResolution != nil {
				fun := qualifiedNameExpr(executionMethodCallName(implicitStaticResolution.def, implicitStaticResolution.owner, ctx), findJavaPackageForClassScope(implicitStaticResolution.owner), ctx)
				fun = applyTypeArguments(fun, typeArgs)
				return markDirectVarargsExpansion(
					&ast.CallExpr{Fun: fun, Args: prependExecutionMethodArgument(ctx, implicitStaticResolution.def, args)},
					expandVarargsArray,
				)
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
		// interface adapter with a closure (mirroring lambda lowering). Otherwise
		// (multiple methods, fields, or extending a class) synthesize a uniquely
		// named struct hoisted to file scope, capturing referenced enclosing
		// locals as fields.
		if classBody := objectCreationClassBody(node); classBody != nil {
			if lowered := lowerAnonymousClass(node, objectType, classBody, source, ctx); lowered != nil {
				return lowered
			}
			if lowered := lowerAnonymousClassToStruct(node, objectType, classBody, source, ctx); lowered != nil {
				return lowered
			}
		}

		// The `outer.new Inner()` / `this.new Inner()` qualifier form is handled
		// below by threading the leading expression as the enclosing instance.

		// Keep the source argument nodes intact until constructor selection. Lambdas,
		// method references, null, numeric widening, and poly conditionals all need
		// Java invocation-conversion scoring before their target parameter type is
		// known.
		objectArguments := node.ChildByFieldName("arguments")

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
			arguments := parseArgumentListWithExpectedTypes(objectArguments, source, ctx, nil)
			return builtinExceptionConstructorExpr(className, arguments, ctx)
		}

		// Find the respective constructor (if we have symbol info for that class).
		var constructor *symbol.Definition
		var constructorResolution *methodResolution
		targetScope := resolveClassScopeByQualifiedName(ctx, className)
		if resolution := findBestConstructor(targetScope, objectArguments, ctx, source); resolution != nil {
			constructor = resolution.def
			constructorResolution = resolution
		}
		targetPkg := resolveJavaPackageForType(ctx, className, targetScope)
		var expectedArgumentTypes []string
		if constructor != nil {
			expectedArgumentTypes = definitionParameterOriginalTypes(constructor)
		} else if stripJavaQualifier(className) == "Thread" && resolveClassScopeByQualifiedName(ctx, className) == nil {
			expectedArgumentTypes = []string{"Runnable"}
		}
		arguments, expandVarargsArray := parseResolvedInvocationArguments(constructorResolution, objectArguments, source, ctx, expectedArgumentTypes)
		localInfo := ctx.localClasses[className]
		if localInfo != nil {
			captureArgs := make([]ast.Expr, 0, len(localInfo.captured))
			for _, capture := range localInfo.captured {
				captureValue := ast.Expr(&ast.Ident{Name: capture.name})
				// Recursive allocation from a hoisted instance method, constructor,
				// or field initializer forwards the value already stored on this
				// instance. At the enclosing call site the lexical local remains the
				// correct source value.
				if localInfo.scope != nil && ctx.currentClass == localInfo.scope && ctx.localScope != nil && !ctx.localScope.IsStatic {
					captureValue = &ast.SelectorExpr{
						X:   &ast.Ident{Name: ShortName(localInfo.structName)},
						Sel: &ast.Ident{Name: capture.name},
					}
				}
				captureArgs = append(captureArgs, captureValue)
			}
			arguments = append(captureArgs, arguments...)
		}

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

		// Standard-library constructors (StringBuilder, ArrayList, HashMap, ...) are
		// handled by the intrinsics table, which maps them onto stdjava runtime
		// constructors. This runs after type arguments are resolved so a collection
		// constructor can carry its element type (e.g. stdjava.NewList[string]()).
		if _, isIntrinsic := constructorIntrinsics[stripJavaQualifier(className)]; isIntrinsic {
			scopeTypeParams := inScopeTypeParameters(ctx)
			typeArgExprs := make([]ast.Expr, 0, len(effectiveTypeArgs))
			for _, ta := range effectiveTypeArgs {
				typeArgExprs = append(typeArgExprs, javaTypeStringToGoTypeExpr(ta, scopeTypeParams, ctx))
			}
			if rewritten, ok := tryConstructorIntrinsic(className, typeArgExprs, arguments, ctx); ok {
				return rewritten
			}
		}

		if constructor != nil {
			constructorName := constructor.Name
			callArgs := arguments
			if executionExpr(ctx) != nil && constructorHasExecutionImplementation(constructor, targetScope) {
				constructorName = executionConstructorImplementationName(constructorName, targetScope)
				callArgs = prependExecutionArgument(ctx, callArgs)
			}
			funExpr := addTypeArgs(qualifiedNameExpr(constructorName, targetPkg, ctx), effectiveTypeArgs)
			call := markDirectVarargsExpansion(&ast.CallExpr{
				Fun:  funExpr,
				Args: callArgs,
			}, expandVarargsArray)
			if targetScope == nil || targetScope.Class == nil {
				return call
			}
			resultType := &ast.StarExpr{X: addTypeArgs(
				qualifiedNameExpr(targetScope.Class.Name, targetPkg, ctx),
				effectiveTypeArgs,
			)}
			if localInfo != nil {
				return call
			}
			return guardClassInitializationBeforeExpr(targetScope, call, resultType, ctx)
		}

		// No explicit constructor matched by argument types. If we resolved the
		// target class within our own symbols, use its actual generated constructor
		// name. Prefer the constructor symbol's Name (so a package-private class
		// binds to `newRectangle`, not the miscased `Newrectangle`); fall back to
		// an export-status-aware New<Name> for a synthesized default constructor.
		if targetScope != nil && targetScope.Class != nil && targetScope.Class.Name != "" {
			ctorName := constructorFuncName(targetScope)
			if ctorName == "" {
				ctorName = defaultConstructorName(targetScope.Class.Name)
			}
			callArgs := arguments
			if executionExpr(ctx) != nil && constructorHasExecutionImplementation(nil, targetScope) {
				ctorName = executionConstructorImplementationName(ctorName, targetScope)
				callArgs = prependExecutionArgument(ctx, callArgs)
			}
			funExpr := addTypeArgs(qualifiedNameExpr(ctorName, targetPkg, ctx), effectiveTypeArgs)
			call := &ast.CallExpr{
				Fun:  funExpr,
				Args: callArgs,
			}
			resultType := &ast.StarExpr{X: addTypeArgs(
				qualifiedNameExpr(targetScope.Class.Name, targetPkg, ctx),
				effectiveTypeArgs,
			)}
			if localInfo != nil {
				return call
			}
			return guardClassInitializationBeforeExpr(targetScope, call, resultType, ctx)
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
		typeNode := node.ChildByFieldName("type")
		elementJavaType := ""
		if typeNode != nil {
			elementJavaType = typeNode.Content(source)
		}
		arrayJavaType, arrayDimensions := javaArrayCreationJavaType(node, source)
		_, _, reifiedSourceArray := reifiedSourceReferenceArrayComponent(arrayJavaType, ctx)
		_, primitiveLeafArray := javaPrimitiveArrayComponent(arrayJavaType)
		if !reifiedSourceArray && !primitiveLeafArray && expectedTypeTargetsExpression(ctx, node) && arrayDimensions > 0 &&
			javaTernaryAssignmentCompatible(arrayJavaType, ctx.expectedType, ctx) {
			targetElement := strings.TrimSpace(ctx.expectedType)
			for dimension := 0; dimension < arrayDimensions && strings.HasSuffix(targetElement, "[]"); dimension++ {
				targetElement = strings.TrimSpace(strings.TrimSuffix(targetElement, "[]"))
			}
			if targetElement != "" && !strings.HasSuffix(targetElement, "[]") {
				elementJavaType = targetElement
			}
		}
		elementType := astutil.ParseType(typeNode, source)
		// Use the symbol-aware converter for the element type so imported generated
		// classes are package-qualified (`new Cohort[n]` ->
		// `make([]*model.Cohort, n)`) and package-private/interface casing remains
		// consistent with declarations and constructor calls.
		if typeNode != nil {
			elementType = javaTypeStringToGoTypeExpr(
				elementJavaType,
				inScopeTypeParameters(ctx),
				ctx,
			)
			// A stdjava-backed runtime element type (e.g. Thread in `new Thread[n]`)
			// must resolve to its stdjava Go type (*stdjava.Thread), not a bare and
			// undefined *Thread.
			if rt, ok := stdjavaRuntimeTypeExpr(stripJavaQualifier(elementJavaType), nil, inScopeTypeParameters(ctx), ctx); ok {
				elementType = rt
			}
		}
		var initializer ast.Expr

		for _, child := range nodeutil.NamedChildrenOf(node) {
			if child.Type() == "dimensions_expr" {
				dimensions = append(dimensions, ParseExpr(child, source, ctx))
			} else if child.Type() == "array_initializer" {
				initCtx := ctx.Clone()
				if reifiedSourceArray || primitiveLeafArray {
					initCtx.lastType = reifiedReferenceArrayTypeExpr(ctx)
					initCtx.expectedType = arrayJavaType
					initCtx.expectedTypeRoot = child
				} else {
					initCtx.lastType = genArrayType(elementType, arrayDimensions)
				}
				if typeNode != nil && !reifiedSourceArray && !primitiveLeafArray {
					initCtx.expectedType = elementJavaType
					initCtx.expectedTypeRoot = child
				}
				initializer = ParseExpr(child, source, initCtx)
			}
		}

		if initializer != nil {
			return initializer
		}

		if len(dimensions) == 0 {
			panic("Array had zero dimensions")
		}
		if reifiedSourceArray {
			_, componentType, componentID, ok := reifiedReferenceArrayComponentInfo(arrayJavaType, ctx)
			if ok && arrayDimensions == 1 {
				return stdjavaGenericCall(ctx, "NewReferenceArrayOf", []ast.Expr{componentType}, []ast.Expr{dimensions[0], componentID})
			}
		}
		if primitiveLeafArray && arrayDimensions == 1 {
			component, _ := javaPrimitiveArrayComponent(arrayJavaType)
			componentType := javaTypeStringToGoTypeExpr(component, inScopeTypeParameters(ctx), ctx)
			componentID, _ := javaPrimitiveTypeIDExpr(component, ctx)
			return stdjavaGenericCall(ctx, "NewPrimitiveArray", []ast.Expr{componentType}, []ast.Expr{dimensions[0], componentID})
		}
		if arrayDimensions > 1 {
			baseComponent, _ := javaArrayTypeParts(arrayJavaType)
			baseComponentType := javaTypeStringToGoTypeExpr(baseComponent, inScopeTypeParameters(ctx), ctx)
			baseComponentID, descriptorOK := javaTypeDescriptorExpr(baseComponent, ctx)
			if descriptorOK {
				args := []ast.Expr{baseComponentID, &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(arrayDimensions)}}
				for _, dimension := range dimensions {
					args = append(args, &ast.CallExpr{Fun: &ast.Ident{Name: "int32"}, Args: []ast.Expr{dimension}})
				}
				return stdjavaGenericCall(ctx, "NewMultiArrayOf", []ast.Expr{baseComponentType}, args)
			}
		}

		return GenMultiDimArray(elementType, dimensions, arrayDimensions, ctx)
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

		rightJavaType := right.Content(source)
		if _, rank := javaArrayTypeParts(rightJavaType); rank > 0 {
			if descriptor, ok := javaTypeDescriptorExpr(rightJavaType, ctx); ok {
				return stdjavaCall(ctx, "JavaArrayInstanceOf", ParseExpr(left, source, ctx), descriptor)
			}
		}

		assertType := instanceofAssertTypeExpr(rightJavaType, ctx)
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
										Args: []ast.Expr{instanceofSubjectExpr(left, right.Content(source), source, ctx)},
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
			leftNode := node.Child(0)
			leftExpr := ParseExpr(leftNode, source, ctx)
			if javaType, ok := inferExprJavaType(leftNode, ctx, source); ok {
				targetType := "int"
				if canonical, numeric := canonicalJavaNumericType(javaType); numeric && canonical == "long" {
					targetType = "long"
				}
				if conversion := goPrimitiveConversionName(targetType); conversion != "" {
					leftExpr = &ast.CallExpr{Fun: &ast.Ident{Name: conversion}, Args: []ast.Expr{leftExpr}}
				}
			}
			return stdjavaCall(ctx, "UnsignedRightShift",
				leftExpr,
				maskedShiftAmount(leftNode, node.Child(2), source, ctx),
			)
		}
		leftNode := node.Child(0)
		rightNode := node.Child(2)
		leftNull := isStaticallyNullReference(leftNode)
		rightNull := isStaticallyNullReference(rightNode)
		if (operator == "==" || operator == "!=") && leftNull && rightNull {
			result := "false"
			if operator == "==" {
				result = "true"
			}
			return &ast.Ident{Name: result}
		}
		if (operator == "==" || operator == "!=") && (leftNull || rightNull) {
			otherNode := rightNode
			if rightNull {
				otherNode = leftNode
			}
			if javaType, ok := inferExprJavaType(otherNode, ctx, source); ok && isJavaStringType(javaType) {
				comparison := ast.Expr(stdjavaCall(ctx, "StringIsNull", ParseExpr(otherNode, source, ctx)))
				if operator == "!=" {
					comparison = &ast.UnaryExpr{Op: token.NOT, X: comparison}
				}
				return comparison
			}
		}
		leftExpr := ParseExpr(leftNode, source, ctx)
		rightExpr := ParseExpr(rightNode, source, ctx)
		if operator == "+" && (isStringLikeExprNode(leftNode, ctx, source) || isStringLikeExprNode(rightNode, ctx, source) || isFmtSprintfCall(leftExpr)) {
			leftExpr = javaStringConversionExpr(leftNode, leftExpr, ctx, source)
			rightExpr = javaStringConversionExpr(rightNode, rightExpr, ctx, source)
			return mergeFmtSprintCall(leftExpr, rightExpr, ctx)
		}
		// Java masks shift counts (int: low 5 bits, long: low 6 bits) before
		// shifting, whereas Go applies the full count. Mask constant shift amounts
		// at transpile time so e.g. `1 << 32` stays 1.
		if operator == "<<" || operator == ">>" {
			rightExpr = maskedShiftAmount(leftNode, rightNode, source, ctx)
			leftExpr = promoteJavaUnaryNumericOperand(leftNode, leftExpr, ctx, source)
		}
		leftExpr, rightExpr = promoteJavaBinaryNumericOperands(
			operator, leftNode, rightNode, leftExpr, rightExpr, source, ctx,
		)
		return &ast.BinaryExpr{
			X:  leftExpr,
			Op: StrToToken(operator),
			Y:  rightExpr,
		}
	case "unary_expression":
		operator := node.Child(0).Content(source)
		operandNode := node.Child(1)
		operand := ParseExpr(operandNode, source, ctx)
		if operator == "+" || operator == "-" || operator == "~" {
			operand = promoteJavaUnaryNumericOperand(operandNode, operand, ctx, source)
		}
		return &ast.UnaryExpr{
			Op: StrToToken(operator),
			X:  operand,
		}
	case "parenthesized_expression":
		return &ast.ParenExpr{
			X: ParseExpr(node.NamedChild(0), source, ctx),
		}
	case "ternary_expression":
		return buildTernaryExpressionIIFE(node, source, ctx)
	case "cast_expression":
		targetJavaType := node.NamedChild(0).Content(source)
		targetType := javaTypeStringToGoTypeExpr(targetJavaType, inScopeTypeParameters(ctx), ctx)
		valueNode := node.NamedChild(1)
		if isJavaStringType(targetJavaType) && isStaticallyNullReference(valueNode) {
			return javaNullStringExpr()
		}
		valueExpr := ParseExpr(valueNode, source, ctx)
		if _, rank := javaArrayTypeParts(targetJavaType); rank > 0 {
			if descriptor, ok := javaTypeDescriptorExpr(targetJavaType, ctx); ok {
				return stdjavaGenericCall(ctx, "JavaArrayCast", []ast.Expr{targetType}, []ast.Expr{valueExpr, descriptor})
			}
		}
		typeAssert := func() ast.Expr {
			return &ast.TypeAssertExpr{
				X: &ast.CallExpr{
					Fun:  &ast.Ident{Name: "any"},
					Args: []ast.Expr{valueExpr},
				},
				Type: targetType,
			}
		}

		if isPrimitiveCastTarget(targetType) {
			// Boxed Java casts are reference casts followed by unboxing. When the
			// operand is Object, a raw generic result, or a type parameter, a Go
			// numeric conversion would either reject generic T at compile time or
			// silently convert the wrong dynamic type. Preserve the runtime check via
			// an assertion; known numeric operands still use Java's numeric cast.
			if isBoxedPrimitiveJavaType(targetJavaType) {
				valueJavaType, known := inferExprJavaType(valueNode, ctx, source)
				if _, numeric := canonicalJavaNumericType(valueJavaType); !known || !numeric {
					return typeAssert()
				}
			}
			return &ast.CallExpr{
				Fun:  targetType,
				Args: []ast.Expr{valueExpr},
			}
		}

		return typeAssert()
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
				if _, _, _, reified := expressionUsesReifiedReferenceArray(obj, ctx, source); reified {
					return stdjavaCall(ctx, "ReferenceArrayLength", ParseExpr(obj, source, ctx))
				}
				if _, _, _, primitive := expressionUsesPrimitiveArray(obj, ctx, source); primitive {
					return stdjavaCall(ctx, "PrimitiveArrayLength", ParseExpr(obj, source, ctx))
				}
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
		if access, ok := resolveStaticFieldAccess(node, source, ctx); ok {
			return lowerStaticFieldRead(access, source, ctx)
		}

		fieldName := node.ChildByFieldName("field").Content(source)
		var owner *symbol.ClassScope
		switch obj.Type() {
		case "this":
			owner = ctx.currentClass
		case "super":
			owner = resolveSuperclassScope(ctx, ctx.currentClass)
		default:
			if ownerType, ok := inferExprJavaType(obj, ctx, source); ok {
				base, _ := parseJavaTypeString(ownerType)
				owner = resolveClassScopeByQualifiedName(ctx, base)
			}
			if owner == nil && obj.Type() == "identifier" {
				owner = resolveClassScopeByIdentifier(ctx, source, obj)
			}
		}
		selName := sanitizeGoIdent(fieldName)
		if def := findFieldInHierarchy(owner, fieldName, ctx); def != nil && def.Name != "" {
			// Resolve against the symbol's final display name. Reserved identifiers
			// can be renamed further than lexical sanitization (`map` -> `map0`) to
			// avoid collisions, and every receiver form must select that same field.
			selName = def.Name
		}
		return &ast.SelectorExpr{
			X:   ParseExpr(obj, source, ctx),
			Sel: &ast.Ident{Name: selName},
		}
	case "array_access":
		arrayNode := node.ChildByFieldName("array")
		indexNode := node.ChildByFieldName("index")
		if arrayNode == nil && node.NamedChildCount() > 0 {
			arrayNode = node.NamedChild(0)
		}
		if indexNode == nil && node.NamedChildCount() > 1 {
			indexNode = node.NamedChild(1)
		}
		if arrayNode != nil && indexNode != nil {
			if _, componentType, componentID, reified := expressionUsesReifiedReferenceArray(arrayNode, ctx, source); reified {
				return stdjavaGenericCall(ctx, "ReferenceArrayGet", []ast.Expr{componentType}, []ast.Expr{
					ParseExpr(arrayNode, source, ctx),
					ParseExpr(indexNode, source, ctx),
					componentID,
				})
			}
			if _, _, _, primitive := expressionUsesPrimitiveArray(arrayNode, ctx, source); primitive {
				return &ast.IndexExpr{
					X: &ast.SelectorExpr{
						X:   ParseExpr(arrayNode, source, ctx),
						Sel: &ast.Ident{Name: "Elements"},
					},
					Index: goIndexExpr(indexNode, source, ctx),
				}
			}
		}
		return &ast.IndexExpr{
			X: ParseExpr(node.NamedChild(0), source, ctx),
			// Java index expressions are int32 now that int locals are pinned, but
			// Go requires an `int` index, so coerce. Plain integer literals are
			// untyped constants and need no cast.
			Index: goIndexExpr(node.NamedChild(1), source, ctx),
		}
	case "scoped_identifier":
		return ParseExpr(node.NamedChild(0), source, ctx)
	case "this":
		return &ast.Ident{Name: ShortName(ctx.className)}
	case "identifier":
		identName := node.Content(source)
		if ctx.localScope != nil {
			if param := ctx.localScope.ParameterByName(identName); param != nil {
				return &ast.Ident{Name: sanitizeGoIdent(param.Name)}
			}
			if local := ctx.localScope.FindVariable(identName); local != nil {
				return &ast.Ident{Name: sanitizeGoIdent(local.Name)}
			}
		}
		if access, ok := resolveStaticFieldAccess(node, source, ctx); ok {
			return lowerStaticFieldRead(access, source, ctx)
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
		return &ast.Ident{Name: sanitizeGoIdent(identName)}
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
		if isJavaStringType(ctx.expectedType) {
			return javaNullStringExpr()
		}
		return &ast.Ident{Name: "nil"}
	case "decimal_integer_literal":
		literal := node.Content(source)
		switch literal[len(literal)-1] {
		case 'L', 'l':
			return &ast.CallExpr{Fun: &ast.Ident{Name: "int64"}, Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: literal[:len(literal)-1]}}}
		}
		return &ast.Ident{Name: literal}
	case "hex_integer_literal":
		return javaNonDecimalIntegerLiteral(node.Content(source), 16, 2)
	case "octal_integer_literal":
		return javaNonDecimalIntegerLiteral(node.Content(source), 8, 1)
	case "binary_integer_literal":
		return javaNonDecimalIntegerLiteral(node.Content(source), 2, 2)
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
		raw := node.Content(source)
		// Text blocks (Java 13+) are delimited by triple quotes; lower them to a Go
		// string literal after JLS incidental-whitespace stripping.
		if strings.HasPrefix(raw, "\"\"\"") {
			return textBlockLiteral(raw)
		}
		return &ast.Ident{Name: raw}
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

// buildTernaryExpressionIIFE lowers Java's conditional operator to a typed,
// immediately-invoked function. A normal Go function call evaluates all of its
// arguments before entering the callee, so the former
// stdjava.Ternary(condition, consequence, alternative) representation eagerly
// evaluated both branches. The branch-local returns below retain Java's lazy
// evaluation, including side effects and exceptions in the unselected branch.
func buildTernaryExpressionIIFE(node *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	conditionNode, consequenceNode, alternativeNode := ternaryExpressionParts(node)
	if conditionNode == nil || consequenceNode == nil || alternativeNode == nil {
		diag := reportUnsupported("expression", node, source, ctx)
		return &ast.CallExpr{
			Fun: &ast.Ident{Name: "panic"},
			Args: []ast.Expr{&ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("%q", strings.TrimPrefix(unsupportedComment(diag), "// ")),
			}},
		}
	}

	resultJavaType, known := inferTernaryResultJavaType(node, ctx, source)
	if !known {
		resultJavaType = "Object"
	}
	resultGoType := ternaryResultGoType(resultJavaType, consequenceNode, alternativeNode, ctx)

	conditionCtx := ctx.Clone()
	conditionCtx.expectedType = "boolean"
	conditionCtx.expectedTypeRoot = conditionNode
	condition := ParseExpr(conditionNode, source, conditionCtx)
	consequence := parseTernaryBranch(consequenceNode, resultJavaType, source, ctx)
	alternative := parseTernaryBranch(alternativeNode, resultJavaType, source, ctx)

	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{
			Params:  &ast.FieldList{},
			Results: &ast.FieldList{List: []*ast.Field{{Type: resultGoType}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.IfStmt{
				Cond: condition,
				Body: &ast.BlockStmt{List: []ast.Stmt{
					&ast.ReturnStmt{Results: []ast.Expr{consequence}},
				}},
			},
			&ast.ReturnStmt{Results: []ast.Expr{alternative}},
		}},
	}}
}

func ternaryExpressionParts(node *sitter.Node) (condition, consequence, alternative *sitter.Node) {
	if node == nil || node.Type() != "ternary_expression" {
		return nil, nil, nil
	}
	condition = node.ChildByFieldName("condition")
	consequence = node.ChildByFieldName("consequence")
	alternative = node.ChildByFieldName("alternative")
	if condition != nil && consequence != nil && alternative != nil {
		return condition, consequence, alternative
	}
	children := nodeutil.NamedChildrenOf(node)
	if len(children) == 3 {
		return children[0], children[1], children[2]
	}
	return nil, nil, nil
}

func parseTernaryBranch(node *sitter.Node, resultJavaType string, source []byte, ctx Ctx) ast.Expr {
	if unwrapped := unwrapParenthesizedExpressionNode(node); unwrapped != nil && unwrapped.Type() == "null_literal" {
		if isJavaStringType(resultJavaType) {
			return javaNullStringExpr()
		}
		return &ast.Ident{Name: "nil"}
	}

	branchCtx := ctx.Clone()
	branchCtx.expectedType = resultJavaType
	branchCtx.expectedTypeRoot = node
	expr := ParseExpr(node, source, branchCtx)

	// A primitive entering Object must retain its Java runtime width. In
	// particular, an untyped Go integer literal would otherwise become host int
	// rather than Java Integer/int32.
	if base, _ := parseJavaTypeString(resultJavaType); stripJavaQualifier(base) == "Object" {
		expr = boxPrimitiveForObject(expr, node, resultJavaType, ctx, source)
	}
	if boxedPrimitive, boxed := ternaryBoxedPrimitive(resultJavaType); boxed {
		if actualType, actualKnown := inferExprJavaType(node, ctx, source); actualKnown {
			if _, actualPrimitive := javaPrimitiveType(actualType); actualPrimitive {
				if conversion := goPrimitiveConversionName(boxedPrimitive); conversion != "" {
					expr = &ast.CallExpr{Fun: &ast.Ident{Name: conversion}, Args: []ast.Expr{expr}}
				}
			}
		}
	}

	if targetNumeric, ok := canonicalJavaNumericType(resultJavaType); ok {
		if actualType, actualKnown := inferExprJavaType(node, ctx, source); actualKnown {
			expr = convertJavaNumericOperand(expr, actualType, targetNumeric)
		}
	}
	return coerceArgumentToExpectedType(expr, node, resultJavaType, ctx, source)
}

func ternaryResultGoType(resultJavaType string, consequence, alternative *sitter.Node, ctx Ctx) ast.Expr {
	// String null is carried by the concrete sentinel, so every String arm is
	// coerced to the ordinary string ABI rather than widening the IIFE to any.
	if isJavaStringType(resultJavaType) {
		return javaTypeStringToGoTypeExpr(resultJavaType, inScopeTypeParameters(ctx), ctx)
	}
	// String and boxed primitives normally use Go value types, but a selected
	// Java null must remain distinguishable from their zero values. Use an
	// interface result when any nested conditional arm can produce literal null;
	// pointer/slice/interface reference representations can use nil directly.
	if usesNullableValueStorage(resultJavaType) &&
		(expressionCanProduceNull(consequence) || expressionCanProduceNull(alternative)) {
		return &ast.Ident{Name: "any"}
	}
	return abstractClassToInterface(
		javaTypeStringToGoTypeExpr(resultJavaType, inScopeTypeParameters(ctx), ctx),
		resultJavaType,
		ctx,
	)
}

func expressionCanProduceNull(node *sitter.Node) bool {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return false
	}
	if node.Type() == "null_literal" {
		return true
	}
	if node.Type() == "cast_expression" && node.NamedChildCount() > 1 {
		return expressionCanProduceNull(node.NamedChild(1))
	}
	if node.Type() != "ternary_expression" {
		return false
	}
	_, consequence, alternative := ternaryExpressionParts(node)
	return expressionCanProduceNull(consequence) || expressionCanProduceNull(alternative)
}

func expressionAlwaysProducesNull(node *sitter.Node) bool {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return false
	}
	if node.Type() == "null_literal" {
		return true
	}
	if node.Type() == "cast_expression" && node.NamedChildCount() > 1 {
		return expressionAlwaysProducesNull(node.NamedChild(1))
	}
	if node.Type() != "ternary_expression" {
		return false
	}
	_, consequence, alternative := ternaryExpressionParts(node)
	return expressionAlwaysProducesNull(consequence) && expressionAlwaysProducesNull(alternative)
}

// isStaticallyNullReference recognizes syntax that denotes the null reference
// without evaluating any user code. It intentionally excludes conditionals:
// even when both branches are null, their condition can have side effects and
// must not be folded away by equality lowering.
func isStaticallyNullReference(node *sitter.Node) bool {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return false
	}
	if node.Type() == "null_literal" {
		return true
	}
	return node.Type() == "cast_expression" && node.NamedChildCount() > 1 && isStaticallyNullReference(node.NamedChild(1))
}

// expressionUsesNullableValueStorage tracks interface-backed nullable locals
// through parentheses and conditional selection. String ABI boundaries coerce
// these values to the concrete null sentinel; boxed primitives retain their
// existing assertion/unboxing behavior.
func expressionUsesNullableValueStorage(node *sitter.Node, ctx Ctx, source []byte) bool {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return false
	}
	if expressionCanProduceNull(node) || isNullableValueBackedLocal(node, ctx, source) {
		return true
	}
	if node.Type() != "ternary_expression" {
		return false
	}
	_, consequence, alternative := ternaryExpressionParts(node)
	return expressionUsesNullableValueStorage(consequence, ctx, source) ||
		expressionUsesNullableValueStorage(alternative, ctx, source)
}

const ternaryNullJavaType = "<java-null>"

type ternaryExpressionKind uint8

const (
	ternaryReferenceExpression ternaryExpressionKind = iota
	ternaryBooleanExpression
	ternaryNumericExpression
)

func classifyTernaryExpression(node *sitter.Node, ctx Ctx, source []byte) ternaryExpressionKind {
	_, consequence, alternative := ternaryExpressionParts(node)
	left := unwrapParenthesizedExpressionNode(consequence)
	right := unwrapParenthesizedExpressionNode(alternative)
	if left == nil || right == nil || left.Type() == "null_literal" || right.Type() == "null_literal" {
		return ternaryReferenceExpression
	}
	leftType, leftKnown := inferExprJavaType(left, ctx, source)
	rightType, rightKnown := inferExprJavaType(right, ctx, source)
	if !leftKnown || !rightKnown {
		return ternaryReferenceExpression
	}
	if ternaryBooleanType(leftType) && ternaryBooleanType(rightType) {
		return ternaryBooleanExpression
	}
	if _, leftNumeric := canonicalJavaNumericType(leftType); leftNumeric {
		if _, rightNumeric := canonicalJavaNumericType(rightType); rightNumeric {
			return ternaryNumericExpression
		}
	}
	return ternaryReferenceExpression
}

func expectedTypeTargetsExpression(ctx Ctx, node *sitter.Node) bool {
	target := unwrapParenthesizedExpressionNode(ctx.expectedTypeRoot)
	node = unwrapParenthesizedExpressionNode(node)
	if target == nil || node == nil {
		return false
	}
	return target.Type() == node.Type() && target.StartByte() == node.StartByte() && target.EndByte() == node.EndByte()
}

func inferTernaryResultJavaType(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	expected := strings.TrimSpace(ctx.expectedType)
	if isVarKeywordType(expected) || !expectedTypeTargetsExpression(ctx, node) {
		expected = ""
	}

	// Boolean and numeric conditionals are standalone expressions under the JLS:
	// assignment/invocation context converts their already-determined result. Only
	// reference conditionals are poly expressions that can take their target type.
	// Keeping the target root in Ctx prevents an enclosing return/assignment type
	// from leaking through a binary expression into a nested conditional.
	inferenceCtx := ctx.Clone()
	inferenceCtx.expectedType = ""
	inferenceCtx.expectedTypeRoot = nil
	standaloneType, standaloneKnown := inferStandaloneTernaryResultJavaType(node, inferenceCtx, source)
	if classifyTernaryExpression(node, inferenceCtx, source) != ternaryReferenceExpression {
		return standaloneType, standaloneKnown
	}
	if expected != "" && (!standaloneKnown || ternaryCanTargetJavaType(node, standaloneType, expected, inferenceCtx, source)) {
		return expected, true
	}
	if standaloneType == ternaryNullJavaType {
		return "Object", true
	}
	return standaloneType, standaloneKnown
}

func inferStandaloneTernaryResultJavaType(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {

	_, consequence, alternative := ternaryExpressionParts(node)
	if consequence == nil || alternative == nil {
		return "", false
	}
	leftNode := unwrapParenthesizedExpressionNode(consequence)
	rightNode := unwrapParenthesizedExpressionNode(alternative)
	leftNull := leftNode != nil && leftNode.Type() == "null_literal"
	rightNull := rightNode != nil && rightNode.Type() == "null_literal"
	leftType, leftKnown := inferExprJavaType(consequence, ctx, source)
	rightType, rightKnown := inferExprJavaType(alternative, ctx, source)

	switch {
	case leftNull && rightNull:
		return ternaryNullJavaType, true
	case leftNull && rightKnown:
		if boxed := ternaryBoxedJavaType(rightType); boxed != "" {
			return boxed, true
		}
		return rightType, true
	case rightNull && leftKnown:
		if boxed := ternaryBoxedJavaType(leftType); boxed != "" {
			return boxed, true
		}
		return leftType, true
	case !leftKnown || !rightKnown:
		return "", false
	}

	if normalizeJavaReferenceType(leftType) == normalizeJavaReferenceType(rightType) {
		return leftType, true
	}
	if ternaryBooleanType(leftType) && ternaryBooleanType(rightType) {
		return "boolean", true
	}
	if numericType, ok := ternaryNumericResultJavaType(consequence, leftType, alternative, rightType, source); ok {
		return numericType, true
	}
	return ternaryCommonReferenceJavaType(leftType, rightType, ctx)
}

func ternaryCanTargetJavaType(node *sitter.Node, standaloneType, expectedType string, ctx Ctx, source []byte) bool {
	if javaTernaryAssignmentCompatible(standaloneType, expectedType, ctx) {
		return true
	}

	// A poly reference conditional can have a broad standalone LUB while each arm
	// is individually compatible with the target. Check both arms as a fallback,
	// keeping null compatible only with reference targets.
	_, consequence, alternative := ternaryExpressionParts(node)
	for _, branch := range []*sitter.Node{consequence, alternative} {
		branch = unwrapParenthesizedExpressionNode(branch)
		if branch == nil {
			return false
		}
		if branch.Type() == "null_literal" {
			if _, primitive := javaPrimitiveType(expectedType); primitive {
				return false
			}
			continue
		}
		branchType, known := inferExprJavaType(branch, ctx, source)
		if !known {
			// Lambdas and method references need the target type to become typed.
			continue
		}
		if !javaTernaryAssignmentCompatible(branchType, expectedType, ctx) {
			return false
		}
	}
	return true
}

func javaTernaryAssignmentCompatible(actualType, expectedType string, ctx Ctx) bool {
	actualType = strings.TrimSpace(actualType)
	expectedType = strings.TrimSpace(expectedType)
	if actualType == "" || expectedType == "" {
		return false
	}
	if normalizeJavaReferenceType(actualType) == normalizeJavaReferenceType(expectedType) {
		return true
	}
	if actualType == ternaryNullJavaType {
		_, expectedPrimitive := javaPrimitiveType(expectedType)
		return !expectedPrimitive
	}

	actualPrimitive, actualIsPrimitive := javaPrimitiveType(actualType)
	expectedPrimitive, expectedIsPrimitive := javaPrimitiveType(expectedType)
	if actualIsPrimitive && expectedIsPrimitive {
		if actualPrimitive == expectedPrimitive {
			return true
		}
		_, widening := javaPrimitiveWideningDistance(actualPrimitive, expectedPrimitive)
		return widening
	}

	expectedBase, _ := parseJavaTypeString(expectedType)
	if stripJavaQualifier(expectedBase) == "Object" {
		return true
	}
	actualComponent, actualArray := javaArrayComponentType(actualType)
	expectedComponent, expectedArray := javaArrayComponentType(expectedType)
	if actualArray || expectedArray {
		if !actualArray || !expectedArray {
			return false
		}
		actualComponentPrimitive, actualPrimitiveArray := javaPrimitiveType(actualComponent)
		expectedComponentPrimitive, expectedPrimitiveArray := javaPrimitiveType(expectedComponent)
		if actualPrimitiveArray || expectedPrimitiveArray {
			return actualPrimitiveArray && expectedPrimitiveArray && actualComponentPrimitive == expectedComponentPrimitive
		}
		return javaTernaryAssignmentCompatible(actualComponent, expectedComponent, ctx)
	}
	if actualIsPrimitive {
		boxedPrimitive, boxed := ternaryBoxedPrimitive(expectedType)
		return boxed && boxedPrimitive == actualPrimitive
	}
	if expectedIsPrimitive {
		boxedPrimitive, boxed := ternaryBoxedPrimitive(actualType)
		if !boxed {
			return false
		}
		if boxedPrimitive == expectedPrimitive {
			return true
		}
		_, widening := javaPrimitiveWideningDistance(boxedPrimitive, expectedPrimitive)
		return widening
	}

	actualBase, actualArgs := parseJavaTypeString(actualType)
	if sameJavaRawType(actualBase, expectedBase) {
		_, expectedArgs := parseJavaTypeString(expectedType)
		return javaGenericArgumentsApplicable(actualArgs, expectedArgs, nil)
	}
	actualScope := resolveClassScopeByQualifiedName(ctx, actualBase)
	expectedScope := resolveClassScopeByQualifiedName(ctx, expectedBase)
	_, assignable := javaReferenceTypeDistance(actualScope, expectedScope, ctx)
	return assignable
}

func ternaryBoxedJavaType(javaType string) string {
	primitive, ok := javaPrimitiveType(javaType)
	if !ok {
		return ""
	}
	switch primitive {
	case "byte":
		return "Byte"
	case "short":
		return "Short"
	case "char":
		return "Character"
	case "int":
		return "Integer"
	case "long":
		return "Long"
	case "float":
		return "Float"
	case "double":
		return "Double"
	case "boolean":
		return "Boolean"
	default:
		return ""
	}
}

func ternaryBoxedPrimitive(javaType string) (string, bool) {
	base, _ := parseJavaTypeString(javaType)
	switch stripJavaQualifier(base) {
	case "Byte":
		return "byte", true
	case "Short":
		return "short", true
	case "Character":
		return "char", true
	case "Integer":
		return "int", true
	case "Long":
		return "long", true
	case "Float":
		return "float", true
	case "Double":
		return "double", true
	case "Boolean":
		return "boolean", true
	default:
		return "", false
	}
}

func ternaryBooleanType(javaType string) bool {
	base, _ := parseJavaTypeString(javaType)
	switch stripJavaQualifier(base) {
	case "boolean", "Boolean":
		return true
	default:
		return false
	}
}

func ternaryNumericResultJavaType(leftNode *sitter.Node, leftType string, rightNode *sitter.Node, rightType string, source []byte) (string, bool) {
	left, leftNumeric := canonicalJavaNumericType(leftType)
	right, rightNumeric := canonicalJavaNumericType(rightType)
	if !leftNumeric || !rightNumeric {
		return "", false
	}
	if left == right {
		return left, true
	}
	if (left == "byte" && right == "short") || (left == "short" && right == "byte") {
		return "short", true
	}
	if ternaryIntConstantFits(rightNode, right, left, source) {
		return left, true
	}
	if ternaryIntConstantFits(leftNode, left, right, source) {
		return right, true
	}
	return javaNumericPromotionType(left, right)
}

func ternaryIntConstantFits(node *sitter.Node, sourceType, targetType string, source []byte) bool {
	if sourceType != "int" {
		return false
	}
	var minimum, maximum int64
	switch targetType {
	case "byte":
		minimum, maximum = -128, 127
	case "short":
		minimum, maximum = -32768, 32767
	case "char":
		minimum, maximum = 0, 65535
	default:
		return false
	}
	value, ok := javaIntConstantExpression(node, source)
	return ok && value >= minimum && value <= maximum
}

// javaIntConstantExpression evaluates the side-effect-free integral constant
// subset needed by the conditional operator's byte/short/char narrowing rule.
// Values use Java int32 wraparound, including hexadecimal, octal, and binary
// spellings whose high bit denotes a negative int.
func javaIntConstantExpression(node *sitter.Node, source []byte) (int64, bool) {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return 0, false
	}
	if node.Type() == "unary_expression" && node.NamedChildCount() > 0 {
		value, ok := javaIntConstantExpression(node.NamedChild(int(node.NamedChildCount())-1), source)
		if !ok {
			return 0, false
		}
		intValue := int32(value)
		switch node.Child(0).Content(source) {
		case "-":
			return int64(-intValue), true
		case "+":
			return int64(intValue), true
		case "~":
			return int64(^intValue), true
		default:
			return 0, false
		}
	}
	if node.Type() == "binary_expression" && node.ChildCount() >= 3 {
		left, leftOK := javaIntConstantExpression(node.Child(0), source)
		right, rightOK := javaIntConstantExpression(node.Child(2), source)
		if !leftOK || !rightOK {
			return 0, false
		}
		a, b := int32(left), int32(right)
		var result int32
		switch node.Child(1).Content(source) {
		case "+":
			result = a + b
		case "-":
			result = a - b
		case "*":
			result = a * b
		case "/":
			if b == 0 {
				return 0, false
			}
			result = int32(int64(a) / int64(b))
		case "%":
			if b == 0 {
				return 0, false
			}
			result = int32(int64(a) % int64(b))
		case "<<":
			result = a << (uint32(b) & 31)
		case ">>":
			result = a >> (uint32(b) & 31)
		case ">>>":
			result = int32(uint32(a) >> (uint32(b) & 31))
		case "&":
			result = a & b
		case "|":
			result = a | b
		case "^":
			result = a ^ b
		default:
			return 0, false
		}
		return int64(result), true
	}

	literalKind := node.Type()
	if literalKind != "decimal_integer_literal" && literalKind != "hex_integer_literal" &&
		literalKind != "octal_integer_literal" && literalKind != "binary_integer_literal" {
		return 0, false
	}
	literal := strings.ReplaceAll(node.Content(source), "_", "")
	if strings.HasSuffix(literal, "l") || strings.HasSuffix(literal, "L") {
		return 0, false
	}
	base := 10
	digits := literal
	switch literalKind {
	case "hex_integer_literal":
		base, digits = 16, strings.TrimPrefix(strings.TrimPrefix(literal, "0x"), "0X")
	case "binary_integer_literal":
		base, digits = 2, strings.TrimPrefix(strings.TrimPrefix(literal, "0b"), "0B")
	case "octal_integer_literal":
		base, digits = 8, strings.TrimPrefix(literal, "0")
		if digits == "" {
			digits = "0"
		}
	}
	value, err := strconv.ParseUint(digits, base, 32)
	if err != nil {
		return 0, false
	}
	return int64(int32(uint32(value))), true
}

func ternaryCommonReferenceJavaType(leftType, rightType string, ctx Ctx) (string, bool) {
	leftComponent, leftArray := javaArrayComponentType(leftType)
	rightComponent, rightArray := javaArrayComponentType(rightType)
	if leftArray || rightArray {
		if !leftArray || !rightArray {
			return "Object", true
		}
		leftPrimitive, leftPrimitiveArray := javaPrimitiveType(leftComponent)
		rightPrimitive, rightPrimitiveArray := javaPrimitiveType(rightComponent)
		if leftPrimitiveArray || rightPrimitiveArray {
			if leftPrimitiveArray && rightPrimitiveArray && leftPrimitive == rightPrimitive {
				return leftType, true
			}
			return "Object", true
		}
		componentType, known := ternaryCommonReferenceJavaType(leftComponent, rightComponent, ctx)
		if !known {
			return "Object", true
		}
		return componentType + "[]", true
	}

	leftScope := resolveClassScopeByQualifiedName(ctx, leftType)
	rightScope := resolveClassScopeByQualifiedName(ctx, rightType)
	if leftScope == nil || rightScope == nil {
		return "Object", true
	}
	if _, assignable := javaReferenceTypeDistance(leftScope, rightScope, ctx); assignable {
		return rightType, true
	}
	if _, assignable := javaReferenceTypeDistance(rightScope, leftScope, ctx); assignable {
		return leftType, true
	}

	for candidate := leftScope; candidate != nil; candidate = ternarySuperclassScope(candidate, ctx) {
		if _, assignable := javaReferenceTypeDistance(rightScope, candidate, ctx); assignable {
			return ternaryClassJavaType(candidate, ctx), true
		}
	}
	return "Object", true
}

func ternarySuperclassScope(scope *symbol.ClassScope, ctx Ctx) *symbol.ClassScope {
	declarationCtx := ctx.Clone()
	if file := findFileScopeForClassScope(scope); file != nil {
		declarationCtx.currentFile = file
	}
	return resolveSuperclassScope(declarationCtx, scope)
}

func ternaryClassJavaType(scope *symbol.ClassScope, ctx Ctx) string {
	if scope == nil || scope.Class == nil {
		return "Object"
	}
	name := scope.Class.OriginalName
	file := findFileScopeForClassScope(scope)
	if file == nil || ctx.currentFile == nil || file.Package == "" || file.Package == ctx.currentFile.Package {
		return name
	}
	return file.Package + "." + name
}

// javaNonDecimalIntegerLiteral preserves Java's signed, fixed-width meaning for
// hexadecimal, octal, and binary literals. Unlike decimal notation, Java allows
// the full unsigned bit pattern in these bases: 0xffffffff is the int value -1
// and 0xffffffffffffffffL is the long value -1. Go otherwise treats those same
// spellings as positive arbitrary-precision constants.
func javaNonDecimalIntegerLiteral(literal string, base, prefixLength int) ast.Expr {
	original := literal
	literal = strings.ReplaceAll(literal, "_", "")
	isLong := strings.HasSuffix(literal, "L") || strings.HasSuffix(literal, "l")
	if isLong {
		literal = literal[:len(literal)-1]
	}
	if prefixLength < 0 || prefixLength >= len(literal) {
		return &ast.Ident{Name: original}
	}

	bitSize := 32
	typeName := "int32"
	if isLong {
		bitSize = 64
		typeName = "int64"
	}
	unsigned, err := strconv.ParseUint(literal[prefixLength:], base, bitSize)
	if err != nil {
		return &ast.Ident{Name: original}
	}

	var signed int64
	if bitSize == 32 {
		signed = int64(int32(uint32(unsigned)))
	} else {
		signed = int64(unsigned)
	}
	return &ast.CallExpr{
		Fun:  &ast.Ident{Name: typeName},
		Args: []ast.Expr{signedIntegerConstant(signed)},
	}
}

func signedIntegerConstant(value int64) ast.Expr {
	if value >= 0 {
		return &ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(value, 10)}
	}
	// Avoid overflowing while taking the magnitude of math.MinInt64.
	magnitude := uint64(-(value + 1)) + 1
	return &ast.UnaryExpr{
		Op: token.SUB,
		X:  &ast.BasicLit{Kind: token.INT, Value: strconv.FormatUint(magnitude, 10)},
	}
}

// javaNullStringExpr is the concrete-string representation shared with
// stdjava.NullString. Keeping the literal at allocation/ABI boundaries avoids
// adding a runtime import to classes that merely declare a String field and
// never perform an operation that observes null.
func javaNullStringExpr() ast.Expr {
	return &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote("\xffjava2go:null-string\x00")}
}

// lowerReferenceArrayCompoundAssignment stages a compound assignment through
// the descriptor-bearing array API. The outer call evaluates and validates the
// array/index and loads the old component before the RHS is evaluated; the
// inner call computes, checks, stores, and returns the narrowed Java result.
func lowerReferenceArrayCompoundAssignment(node *sitter.Node, lhsJavaType, operator string, source []byte, ctx Ctx) (ast.Expr, bool) {
	if node == nil || node.ChildCount() < 3 || operator == "=" {
		return nil, false
	}
	lhsNode, rhsNode := node.Child(0), node.Child(2)
	if lhsNode == nil || lhsNode.Type() != "array_access" || rhsNode == nil {
		return nil, false
	}
	arrayNode := lhsNode.ChildByFieldName("array")
	indexNode := lhsNode.ChildByFieldName("index")
	if arrayNode == nil && lhsNode.NamedChildCount() > 0 {
		arrayNode = lhsNode.NamedChild(0)
	}
	if indexNode == nil && lhsNode.NamedChildCount() > 1 {
		indexNode = lhsNode.NamedChild(1)
	}
	if arrayNode == nil || indexNode == nil {
		return nil, false
	}
	_, componentType, componentID, reified := expressionUsesReifiedReferenceArray(arrayNode, ctx, source)
	if !reified {
		return nil, false
	}

	rhsJavaType, known := inferExprJavaType(rhsNode, ctx, source)
	if !known || strings.TrimSpace(rhsJavaType) == "" {
		rhsJavaType = "Object"
	}
	rhsType := javaTypeStringToGoTypeExpr(rhsJavaType, inScopeTypeParameters(ctx), ctx)
	value, ok := compoundAssignmentValue(operator, &ast.Ident{Name: "old"}, &ast.Ident{Name: "rhs"}, lhsJavaType, rhsJavaType, ctx)
	if !ok {
		return nil, false
	}
	inner := &ast.FuncLit{
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "rhs"}}, Type: rhsType}}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: componentType}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.AssignStmt{Lhs: []ast.Expr{&ast.Ident{Name: "value"}}, Tok: token.DEFINE, Rhs: []ast.Expr{value}},
			&ast.ExprStmt{X: stdjavaCall(ctx, "ReferenceArraySet", &ast.Ident{Name: "array"}, &ast.Ident{Name: "index"}, &ast.Ident{Name: "value"})},
			&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "value"}}},
		}},
	}
	outer := &ast.FuncLit{
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: []*ast.Field{
				{Names: []*ast.Ident{{Name: "array"}}, Type: reifiedReferenceArrayTypeExpr(ctx)},
				{Names: []*ast.Ident{{Name: "index"}}, Type: &ast.Ident{Name: "int32"}},
			}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: inner.Type}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: "old"}},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{stdjavaGenericCall(ctx, "ReferenceArrayGet", []ast.Expr{componentType}, []ast.Expr{
					&ast.Ident{Name: "array"}, &ast.Ident{Name: "index"}, componentID,
				})},
			},
			&ast.ReturnStmt{Results: []ast.Expr{inner}},
		}},
	}
	return &ast.CallExpr{
		Fun: &ast.CallExpr{Fun: outer, Args: []ast.Expr{
			ParseExpr(arrayNode, source, ctx),
			&ast.CallExpr{Fun: &ast.Ident{Name: "int32"}, Args: []ast.Expr{ParseExpr(indexNode, source, ctx)}},
		}},
		Args: []ast.Expr{ParseExpr(rhsNode, source, ctx)},
	}, true
}

// lowerAssignmentExpression implements Java assignments that occur in value
// position. Go assignments are statements, so the generated expression uses a
// small immediately-invoked closure and returns the stored value.
//
// The staging is deliberate. Non-array target addresses are evaluated before
// the right-hand side, and compound assignments capture the target's old value
// before evaluating that right-hand side. Simple array assignments take the
// ArraySet path instead because Java delays their null/bounds checks until after
// the RHS. Together these paths preserve single evaluation of complex targets
// and cases such as x += (x = 5), whose result uses x's pre-RHS value.
func lowerAssignmentExpression(node *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	if node == nil || node.ChildCount() < 3 {
		return &ast.BadExpr{}
	}

	lhsNode := node.Child(0)
	opNode := node.Child(1)
	rhsNode := node.Child(2)
	if lhsNode == nil || opNode == nil || rhsNode == nil {
		return &ast.BadExpr{}
	}
	if lowered, ok := lowerStaticFieldAssignment(node, source, ctx); ok {
		return lowered
	}

	lhsJavaType, lhsTypeKnown := inferExprJavaType(lhsNode, ctx, source)
	if !lhsTypeKnown || strings.TrimSpace(lhsJavaType) == "" {
		log.WithField("assignment", node.Content(source)).Warn("Could not infer assignment target type")
		return &ast.BadExpr{}
	}

	targetValueType := javaTypeStringToGoTypeExpr(lhsJavaType, inScopeTypeParameters(ctx), ctx)
	targetStorageType := targetValueType
	nullableValueStorage := isNullableValueBackedLocal(lhsNode, ctx, source)
	if nullableValueStorage {
		targetStorageType = &ast.Ident{Name: "any"}
	}
	targetAddress := &ast.UnaryExpr{Op: token.AND, X: ParseExpr(lhsNode, source, ctx)}
	operator := opNode.Content(source)
	if compound, ok := lowerReferenceArrayCompoundAssignment(node, lhsJavaType, operator, source, ctx); ok {
		return compound
	}

	if operator == "=" {
		if call, ok := lowerSimpleArrayAssignmentCall(node, source, ctx); ok {
			return call
		}
		rhsCtx := ctx.Clone()
		rhsCtx.expectedType = lhsJavaType
		rhsCtx.expectedTypeRoot = rhsNode
		rhs := ParseExpr(rhsNode, source, rhsCtx)
		rhs = coerceArgumentToExpectedType(rhs, rhsNode, lhsJavaType, ctx, source)
		return assignmentValueCall(targetStorageType, targetValueType, targetAddress, rhs)
	}

	rhsJavaType, rhsTypeKnown := inferExprJavaType(rhsNode, ctx, source)
	if !rhsTypeKnown || strings.TrimSpace(rhsJavaType) == "" {
		// An unknown expression can still be passed through an interface value.
		// Known primitive/reference expressions retain their concrete generated Go
		// type so arithmetic and method results remain statically checked.
		rhsJavaType = "Object"
	}
	rhsType := javaTypeStringToGoTypeExpr(rhsJavaType, inScopeTypeParameters(ctx), ctx)
	rhs := ParseExpr(rhsNode, source, ctx)

	oldValue := ast.Expr(&ast.Ident{Name: "old"})
	if nullableValueStorage {
		lhsBase, _ := parseJavaTypeString(lhsJavaType)
		if stripJavaQualifier(lhsBase) != "String" {
			oldValue = &ast.TypeAssertExpr{X: oldValue, Type: targetValueType}
		}
	}

	value, ok := compoundAssignmentValue(
		operator,
		oldValue,
		&ast.Ident{Name: "rhs"},
		lhsJavaType,
		rhsJavaType,
		ctx,
	)
	if !ok {
		log.WithFields(log.Fields{
			"assignment": node.Content(source),
			"operator":   operator,
		}).Warn("Unsupported assignment expression operator")
		return &ast.BadExpr{}
	}

	inner := &ast.FuncLit{
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: []*ast.Field{
				{Names: []*ast.Ident{{Name: "rhs"}}, Type: rhsType},
			}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: targetValueType}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: "value"}},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{value},
			},
			&ast.AssignStmt{
				Lhs: []ast.Expr{&ast.StarExpr{X: &ast.Ident{Name: "dst"}}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{&ast.Ident{Name: "value"}},
			},
			&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "value"}}},
		}},
	}

	outer := &ast.FuncLit{
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: []*ast.Field{
				{Names: []*ast.Ident{{Name: "dst"}}, Type: &ast.StarExpr{X: targetStorageType}},
			}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: inner.Type}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: "old"}},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{&ast.StarExpr{X: &ast.Ident{Name: "dst"}}},
			},
			&ast.ReturnStmt{Results: []ast.Expr{inner}},
		}},
	}

	return &ast.CallExpr{
		Fun:  &ast.CallExpr{Fun: outer, Args: []ast.Expr{targetAddress}},
		Args: []ast.Expr{rhs},
	}
}

func assignmentValueCall(storageType, valueType ast.Expr, targetAddress, rhs ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.FuncLit{
			Type: &ast.FuncType{
				Params: &ast.FieldList{List: []*ast.Field{
					{Names: []*ast.Ident{{Name: "dst"}}, Type: &ast.StarExpr{X: storageType}},
					{Names: []*ast.Ident{{Name: "value"}}, Type: valueType},
				}},
				Results: &ast.FieldList{List: []*ast.Field{{Type: valueType}}},
			},
			Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{&ast.StarExpr{X: &ast.Ident{Name: "dst"}}},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{&ast.Ident{Name: "value"}},
				},
				&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "value"}}},
			}},
		},
		Args: []ast.Expr{targetAddress, rhs},
	}
}

func compoundAssignmentValue(operator string, old, rhs ast.Expr, lhsJavaType, rhsJavaType string, ctx Ctx) (ast.Expr, bool) {
	lhsBase, _ := parseJavaTypeString(lhsJavaType)
	lhsBase = stripJavaQualifier(lhsBase)
	rhsBase, _ := parseJavaTypeString(rhsJavaType)
	rhsBase = stripJavaQualifier(rhsBase)

	if operator == "+=" && lhsBase == "String" {
		var rhsString ast.Expr
		switch rhsBase {
		case "String":
			rhsString = stdjavaCall(ctx, "StringValueOf", rhs)
		case "char", "Character":
			rhsString = &ast.CallExpr{Fun: &ast.Ident{Name: "string"}, Args: []ast.Expr{rhs}}
		default:
			rhsString = javaStringValueOfForType(rhsJavaType, rhs, ctx)
		}
		return &ast.BinaryExpr{X: stdjavaCall(ctx, "StringValueOf", old), Op: token.ADD, Y: rhsString}, true
	}

	if lhsBase == "boolean" || lhsBase == "Boolean" {
		switch operator {
		case "&=":
			return &ast.BinaryExpr{X: old, Op: token.LAND, Y: rhs}, true
		case "|=":
			return &ast.BinaryExpr{X: old, Op: token.LOR, Y: rhs}, true
		case "^=":
			return &ast.BinaryExpr{X: old, Op: token.NEQ, Y: rhs}, true
		default:
			return nil, false
		}
	}

	lhsNumeric, lhsOK := canonicalJavaNumericType(lhsJavaType)
	if !lhsOK {
		return nil, false
	}

	var operation ast.Expr
	switch operator {
	case "<<=", ">>=", ">>>=":
		promotedLeft := lhsNumeric
		if promotedLeft == "byte" || promotedLeft == "short" || promotedLeft == "char" {
			promotedLeft = "int"
		}
		left := convertJavaNumericOperand(old, lhsJavaType, promotedLeft)
		mask := int64(31)
		if promotedLeft == "long" {
			mask = 63
		}
		shift := &ast.BinaryExpr{
			X:  rhs,
			Op: token.AND,
			Y:  &ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(mask, 10)},
		}
		switch operator {
		case "<<=":
			operation = &ast.BinaryExpr{X: left, Op: token.SHL, Y: shift}
		case ">>=":
			operation = &ast.BinaryExpr{X: left, Op: token.SHR, Y: shift}
		case ">>>=":
			operation = stdjavaCall(ctx, "UnsignedRightShift", left, shift)
		}
	default:
		promoted, ok := javaNumericPromotionType(lhsJavaType, rhsJavaType)
		if !ok {
			return nil, false
		}
		left := convertJavaNumericOperand(old, lhsJavaType, promoted)
		right := convertJavaNumericOperand(rhs, rhsJavaType, promoted)
		var op token.Token
		switch operator {
		case "+=":
			op = token.ADD
		case "-=":
			op = token.SUB
		case "*=":
			op = token.MUL
		case "/=":
			op = token.QUO
		case "%=":
			op = token.REM
		case "&=":
			op = token.AND
		case "|=":
			op = token.OR
		case "^=":
			op = token.XOR
		default:
			return nil, false
		}
		operation = &ast.BinaryExpr{X: left, Op: op, Y: right}
	}

	if lhsNumeric == "char" {
		// Java char compound assignment narrows modulo 2^16. The project uses
		// rune/int32 for char values so text operations remain convenient, hence
		// the explicit uint16 step before converting back to rune.
		return &ast.CallExpr{
			Fun: &ast.Ident{Name: "rune"},
			Args: []ast.Expr{&ast.CallExpr{
				Fun:  &ast.Ident{Name: "uint16"},
				Args: []ast.Expr{operation},
			}},
		}, true
	}

	conversion := goPrimitiveConversionName(lhsNumeric)
	if conversion == "" {
		return operation, true
	}
	return &ast.CallExpr{Fun: &ast.Ident{Name: conversion}, Args: []ast.Expr{operation}}, true
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

// javaStringConversionExpr applies Java's String-conversion rules to one
// concatenation operand. Nested concatenations have already converted each of
// their operands, so their accumulated fmt.Sprintf call can be reused. A char
// needs static-type-aware rune-to-string conversion; all other values use the
// runtime bridge for Java null and floating-point spelling.
func javaStringConversionExpr(node *sitter.Node, expr ast.Expr, ctx Ctx, source []byte) ast.Expr {
	if isFmtSprintfCall(expr) {
		return expr
	}
	if isCharTypedExprNode(node, ctx, source) {
		return &ast.CallExpr{Fun: &ast.Ident{Name: "string"}, Args: []ast.Expr{expr}}
	}
	if execution := executionExpr(ctx); execution != nil && node != nil {
		if javaType, ok := inferExprJavaType(node, ctx, source); ok {
			base, _ := parseJavaTypeString(javaType)
			if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil && scope.IsEnum {
				return &ast.CallExpr{
					Fun:  &ast.SelectorExpr{X: expr, Sel: &ast.Ident{Name: enumExecutionStringMethodName(scope)}},
					Args: []ast.Expr{execution},
				}
			}
		}
	}
	if isNullableStringStorageExpression(node, ctx, source) {
		return stdjavaCall(ctx, "StringValueOf", expr)
	}
	if node != nil && node.Type() != "null_literal" && !isNullableValueBackedLocal(node, ctx, source) {
		if javaType, ok := inferExprJavaType(node, ctx, source); ok {
			base, _ := parseJavaTypeString(javaType)
			switch stripJavaQualifier(base) {
			case "String":
				// Any String reference can carry the concrete null sentinel after a
				// field read, method return, or parameter pass. Literals are the one
				// representation that is statically known non-null.
				if node.Type() == "string_literal" {
					return expr
				}
				return stdjavaCall(ctx, "StringValueOf", expr)
			case "byte", "Byte", "short", "Short", "int", "Integer", "long", "Long", "boolean", "Boolean":
				// fmt uses Java-compatible spelling for these concrete values.
				return expr
			}
		}
	}
	if node != nil {
		if javaType, ok := inferExprJavaType(node, ctx, source); ok {
			return javaStringValueOfForType(javaType, expr, ctx)
		}
	}
	return stdjavaCall(ctx, "StringValueOf", expr)
}

// javaStringValueOfForType preserves the current execution when the static
// type can erase an enum value. Generated enum String methods may synchronize,
// so calling their public fmt.Stringer wrapper from inside an already-held
// monitor would otherwise create a fresh token and deadlock.
func javaStringValueOfForType(javaType string, expr ast.Expr, ctx Ctx) ast.Expr {
	execution := executionExpr(ctx)
	if execution == nil {
		return stdjavaCall(ctx, "StringValueOf", expr)
	}
	base, _ := parseJavaTypeString(javaType)
	base = stripJavaQualifier(base)
	needsExecution := base == "Object" || base == "Enum"
	if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil {
		needsExecution = needsExecution || scope.IsEnum || scope.IsInterface || scope.IsAbstract
	}
	if needsExecution {
		return stdjavaCall(ctx, "StringValueOfExecution", execution, expr)
	}
	return stdjavaCall(ctx, "StringValueOf", expr)
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
func enclosingMemberMethodSelector(methodName string, argCount int, ctx Ctx) (ast.Expr, *methodResolution) {
	scope := ctx.currentClass
	if scope == nil || !scope.IsInner {
		return nil, nil
	}
	if ctx.localScope != nil && ctx.localScope.IsStatic {
		return nil, nil
	}
	recvName := ctx.className
	if recvName == "" && scope.Class != nil {
		recvName = scope.Class.Name
	}
	if recvName == "" {
		return nil, nil
	}

	var expr ast.Expr = &ast.Ident{Name: ShortName(recvName)}
	for cur := scope; cur != nil && cur.IsInner; cur = cur.Enclosing {
		expr = &ast.SelectorExpr{X: expr, Sel: &ast.Ident{Name: cur.EnclosingFieldName()}}
		encl := cur.Enclosing
		if encl == nil {
			break
		}
		if resolved := findInstanceMethodInHierarchy(encl, methodName, argCount, ctx); resolved != nil && resolved.def != nil {
			return &ast.SelectorExpr{X: expr, Sel: &ast.Ident{Name: executionMethodCallName(resolved.def, resolved.owner, ctx)}}, resolved
		}
	}
	return nil, nil
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
	// Method-local classes are synthesized while rendering a method body rather
	// than registered in the file's ordinary symbol tree. Keep their real scope
	// reachable by the Java source name so type-qualified static calls and values
	// declared with the local type use the same overload/member resolution as
	// ordinary classes.
	if info := ctx.localClasses[name]; info != nil && info.scope != nil {
		return info.scope
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

// resolveSuperclassScopeInDeclaringContext resolves an extends clause where it
// was written. Hierarchy walks often start from a receiver used in an unrelated
// package; resolving an unqualified superclass name against that call site's
// imports silently truncates the walk and leaves inherited selectors with their
// original Java casing.
func resolveSuperclassScopeInDeclaringContext(ctx Ctx, scope *symbol.ClassScope) *symbol.ClassScope {
	declarationCtx := ctx.Clone()
	if file := findFileScopeForClassScope(scope); file != nil {
		declarationCtx.currentFile = file
		declarationCtx.currentClass = scope
	}
	return resolveSuperclassScope(declarationCtx, scope)
}

func resolveImplementedInterfaceScopesInDeclaringContext(ctx Ctx, scope *symbol.ClassScope) []*symbol.ClassScope {
	if scope == nil {
		return nil
	}
	declarationCtx := ctx.Clone()
	if file := findFileScopeForClassScope(scope); file != nil {
		declarationCtx.currentFile = file
		declarationCtx.currentClass = scope
	}
	interfaces := make([]*symbol.ClassScope, 0, len(scope.ImplementedInterfaces))
	for _, implemented := range scope.ImplementedInterfaces {
		base, _ := parseJavaTypeString(implemented)
		if resolved := resolveClassScopeByQualifiedName(declarationCtx, base); resolved != nil {
			interfaces = append(interfaces, resolved)
		}
	}
	return interfaces
}

type methodResolution struct {
	def                *symbol.Definition
	owner              *symbol.ClassScope
	expandVarargsArray bool
}

// virtualDispatchMethodCall routes a call through the declaring class's stored
// dynamic receiver when that class can have a more-derived implementation.
// Go's embedded methods otherwise retain their original receiver, so a base
// method calling another virtual method would incorrectly invoke the base
// implementation. Explicit super calls bypass this helper at the call site.
func virtualDispatchMethodCall(receiver ast.Expr, resolution *methodResolution, args []ast.Expr, expandVarargsArray bool, ctx Ctx) ast.Expr {
	if receiver == nil || resolution == nil || resolution.def == nil || resolution.owner == nil {
		return nil
	}
	if resolution.def.IsStatic || resolution.def.IsPrivate || resolution.def.RequiresHelper || !classNeedsVirtualDispatch(resolution.owner, ctx) {
		return nil
	}
	return markDirectVarargsExpansion(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.SelectorExpr{
				X:   receiver,
				Sel: &ast.Ident{Name: classDispatchFieldName(resolution.owner)},
			},
			Sel: &ast.Ident{Name: executionMethodCallName(resolution.def, resolution.owner, ctx)},
		},
		Args: prependExecutionMethodArgument(ctx, resolution.def, args),
	}, expandVarargsArray)
}

// stageStaticInvocationQualifier preserves the otherwise-surprising Java rule
// that the primary expression in `value.staticMethod(args)` is evaluated even
// though static dispatch ignores its resulting value. The primary must run once
// before any argument. A zero-argument IIFE gives Go that sequence without
// inventing parameter types for the arguments.
func stageStaticInvocationQualifier(
	invocationNode *sitter.Node,
	qualifier ast.Expr,
	resolution *methodResolution,
	call ast.Expr,
	ctx Ctx,
	source []byte,
) ast.Expr {
	if qualifier == nil || resolution == nil || resolution.def == nil || !resolution.def.IsStatic || call == nil {
		return nil
	}
	results, ok := invocationClosureResults(invocationNode, resolution, ctx, source)
	if !ok {
		return nil
	}
	body := []ast.Stmt{&ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{qualifier},
	}}
	body = append(body, invocationClosureCallStatement(call, results))
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Results: results},
		Body: &ast.BlockStmt{List: body},
	}}
}

// stageVirtualDispatchInvocation models Java's invocation sequence around the
// synthetic self field used for inherited virtual dispatch. Selecting that Go
// field would otherwise dereference a null receiver before evaluating the Java
// arguments. The staged IIFE evaluates receiver and arguments first; selecting
// the dispatch field then naturally raises the null failure at Java's point.
// Ordinary direct calls remain direct and rely on the source method's entry
// guard to prevent Go pointer methods from executing on a null Java receiver.
func stageVirtualDispatchInvocation(
	invocationNode, receiverNode *sitter.Node,
	receiver ast.Expr,
	resolution *methodResolution,
	args []ast.Expr,
	buildCall func(ast.Expr, []ast.Expr) ast.Expr,
	ctx Ctx,
	source []byte,
) ast.Expr {
	if invocationNode == nil || receiverNode == nil || receiver == nil || resolution == nil || resolution.def == nil || resolution.def.IsStatic || buildCall == nil {
		return nil
	}
	results, ok := invocationClosureResults(invocationNode, resolution, ctx, source)
	if !ok {
		return nil
	}

	usedNames := affineLoopUsedNames(invocationNode, source, ctx)
	receiverName := synchronizedUniqueLocalName("__java2goInvocationReceiver", usedNames)
	body := []ast.Stmt{stagedInvocationLocal(receiverName, receiver)}
	stagedArgs := make([]ast.Expr, len(args))
	argumentNodes := nodeutil.NamedChildrenOf(invocationNode.ChildByFieldName("arguments"))
	for index, argument := range args {
		name := synchronizedUniqueLocalName("__java2goInvocationArg"+strconv.Itoa(index), usedNames)
		javaType := ""
		if index < len(resolution.def.Parameters) && resolution.def.Parameters[index] != nil {
			javaType = resolution.def.Parameters[index].OriginalType
		} else if len(resolution.def.Parameters) > 0 && executionParameterIsVariadic(resolution.def, len(resolution.def.Parameters)-1) {
			javaType = resolution.def.Parameters[len(resolution.def.Parameters)-1].OriginalType
		}
		var argumentNode *sitter.Node
		if index < len(argumentNodes) {
			argumentNode = argumentNodes[index]
		}
		statement, ok := stagedInvocationArgumentLocal(name, argument, argumentNode, javaType, resolution, ctx, source)
		if !ok {
			return nil
		}
		body = append(body, statement)
		stagedArgs[index] = &ast.Ident{Name: name}
	}

	receiverIdent := &ast.Ident{Name: receiverName}
	call := buildCall(receiverIdent, stagedArgs)
	if call == nil {
		return nil
	}
	body = append(body, invocationClosureCallStatement(call, results))
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Results: results},
		Body: &ast.BlockStmt{List: body},
	}}
}

func executionCompanionDispatchInvocation(
	invocationNode, receiverNode *sitter.Node,
	receiver ast.Expr,
	target *invocationTargetInfo,
	resolution *methodResolution,
	args []ast.Expr,
	expandVarargsArray bool,
	ctx Ctx,
	source []byte,
) ast.Expr {
	if invocationNode == nil || receiverNode == nil || receiver == nil || target == nil ||
		resolution == nil || resolution.def == nil || resolution.owner == nil || executionExpr(ctx) == nil {
		return nil
	}
	if resolution.def.DeclarationNode == nil {
		return nil
	}
	results, ok := invocationClosureResults(invocationNode, resolution, ctx, source)
	if !ok {
		return nil
	}
	companionType := executionCompanionTypeExpr(target, resolution, ctx)
	if companionType == nil {
		return nil
	}

	usedNames := affineLoopUsedNames(invocationNode, source, ctx)
	receiverName := synchronizedUniqueLocalName("__java2goInvocationReceiver", usedNames)
	companionName := synchronizedUniqueLocalName("__java2goExecutionReceiver", usedNames)
	okName := synchronizedUniqueLocalName("__java2goHasExecutionReceiver", usedNames)
	body := []ast.Stmt{stagedInvocationLocal(receiverName, receiver)}
	stagedArgs := make([]ast.Expr, len(args))
	argumentNodes := nodeutil.NamedChildrenOf(invocationNode.ChildByFieldName("arguments"))
	for index, argument := range args {
		name := synchronizedUniqueLocalName("__java2goInvocationArg"+strconv.Itoa(index), usedNames)
		javaType := ""
		if index < len(resolution.def.Parameters) && resolution.def.Parameters[index] != nil {
			javaType = resolution.def.Parameters[index].OriginalType
		} else if len(resolution.def.Parameters) > 0 && executionParameterIsVariadic(resolution.def, len(resolution.def.Parameters)-1) {
			javaType = resolution.def.Parameters[len(resolution.def.Parameters)-1].OriginalType
		}
		var argumentNode *sitter.Node
		if index < len(argumentNodes) {
			argumentNode = argumentNodes[index]
		}
		statement, staged := stagedInvocationArgumentLocal(name, argument, argumentNode, javaType, resolution, ctx, source)
		if !staged {
			return nil
		}
		body = append(body, statement)
		stagedArgs[index] = &ast.Ident{Name: name}
	}

	body = append(body, &ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: companionName}, &ast.Ident{Name: okName}},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{&ast.TypeAssertExpr{
			X: &ast.CallExpr{
				Fun:  &ast.InterfaceType{Methods: &ast.FieldList{}},
				Args: []ast.Expr{&ast.Ident{Name: receiverName}},
			},
			Type: companionType,
		}},
	})
	hiddenCall := markDirectVarargsExpansion(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: companionName},
			Sel: &ast.Ident{Name: executionImplementationName(resolution.def, resolution.owner)},
		},
		Args: prependExecutionMethodArgument(ctx, resolution.def, stagedArgs),
	}, expandVarargsArray)
	hiddenBody := []ast.Stmt{invocationClosureCallStatement(hiddenCall, results)}
	if results == nil || len(results.List) == 0 {
		hiddenBody = append(hiddenBody, &ast.ReturnStmt{})
	}
	body = append(body, &ast.IfStmt{
		Cond: &ast.Ident{Name: okName},
		Body: &ast.BlockStmt{List: hiddenBody},
	})
	publicCall := markDirectVarargsExpansion(&ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: receiverName},
			Sel: &ast.Ident{Name: resolution.def.Name},
		},
		Args: stagedArgs,
	}, expandVarargsArray)
	body = append(body, invocationClosureCallStatement(publicCall, results))
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Results: results},
		Body: &ast.BlockStmt{List: body},
	}}
}

func executionCompanionTypeExpr(target *invocationTargetInfo, resolution *methodResolution, ctx Ctx) ast.Expr {
	if target == nil || target.classScope == nil || resolution == nil || resolution.owner == nil {
		return nil
	}
	owner := resolution.owner
	typeArgs := target.classTypeArgs
	if owner != target.classScope {
		typeArgs = mapClassTypeArgsToAncestor(target.classScope, target.classTypeArgs, owner, ctx)
	}
	if len(owner.TypeParameters) > 0 && len(typeArgs) != len(owner.TypeParameters) {
		return nil
	}
	typeExpr := qualifiedNameExpr(
		executionCompanionInterfaceName(owner),
		findJavaPackageForClassScope(owner),
		ctx,
	)
	if len(typeArgs) > 0 {
		typeExpr = applyTypeArguments(typeExpr, typeArgs)
	}
	return typeExpr
}

func stagedInvocationLocal(name string, value ast.Expr) ast.Stmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: name}},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{value},
	}
}

func stagedInvocationArgumentLocal(
	name string,
	value ast.Expr,
	valueNode *sitter.Node,
	parameterJavaType string,
	resolution *methodResolution,
	ctx Ctx,
	source []byte,
) (ast.Stmt, bool) {
	if !invocationArgumentNeedsContextualType(value, valueNode) {
		return stagedInvocationLocal(name, value), true
	}
	javaType := strings.TrimSpace(parameterJavaType)
	if javaType == "" || invocationTypeUsesMethodParameter(javaType, resolution) {
		return nil, false
	}
	if valueNode != nil && valueNode.Type() == "null_literal" {
		base, _ := parseJavaTypeString(javaType)
		if _, primitive := canonicalJavaNumericType(base); primitive || stripJavaQualifier(base) == "boolean" || stripJavaQualifier(base) == "char" {
			return nil, false
		}
	}
	return &ast.DeclStmt{Decl: &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names:  []*ast.Ident{{Name: name}},
			Type:   javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx),
			Values: []ast.Expr{value},
		}},
	}}, true
}

func invocationArgumentNeedsContextualType(value ast.Expr, valueNode *sitter.Node) bool {
	if valueNode != nil {
		switch valueNode.Type() {
		case "null_literal", "decimal_integer_literal", "decimal_floating_point_literal":
			return true
		case "parenthesized_expression", "unary_expression":
			if valueNode.NamedChildCount() > 0 {
				return invocationArgumentNeedsContextualType(value, valueNode.NamedChild(int(valueNode.NamedChildCount())-1))
			}
		}
	}
	switch expr := value.(type) {
	case *ast.BasicLit:
		return expr.Kind == token.INT || expr.Kind == token.FLOAT
	case *ast.Ident:
		return expr.Name == "nil"
	case *ast.ParenExpr:
		return invocationArgumentNeedsContextualType(expr.X, nil)
	case *ast.UnaryExpr:
		return invocationArgumentNeedsContextualType(expr.X, nil)
	case *ast.BinaryExpr:
		return invocationArgumentNeedsContextualType(expr.X, nil) && invocationArgumentNeedsContextualType(expr.Y, nil)
	default:
		return false
	}
}

func invocationTypeUsesMethodParameter(javaType string, resolution *methodResolution) bool {
	if resolution == nil || resolution.def == nil {
		return true
	}
	typeParameters := resolution.def.TypeParameterNames()
	if resolution.owner != nil {
		typeParameters = append(typeParameters, resolution.owner.TypeParameterNames()...)
	}
	var uses func(string) bool
	uses = func(candidate string) bool {
		base, args := parseJavaTypeString(candidate)
		for _, typeParam := range typeParameters {
			if base == typeParam {
				return true
			}
		}
		for _, arg := range args {
			if uses(arg) {
				return true
			}
		}
		return false
	}
	return uses(javaType)
}

func invocationClosureResults(invocationNode *sitter.Node, resolution *methodResolution, ctx Ctx, source []byte) (*ast.FieldList, bool) {
	if resolution == nil || resolution.def == nil {
		return nil, false
	}
	declared := strings.TrimSpace(resolution.def.OriginalType)
	if declared == "" || declared == "void" {
		return nil, true
	}
	javaType := declared
	if inferred, ok := inferExprJavaType(invocationNode, ctx, source); ok && inferred != ternaryNullJavaType {
		javaType = inferred
	}
	if invocationTypeUsesMethodParameter(javaType, resolution) {
		return nil, false
	}
	return &ast.FieldList{List: []*ast.Field{{
		Type: javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx),
	}}}, true
}

func invocationClosureCallStatement(call ast.Expr, results *ast.FieldList) ast.Stmt {
	if results == nil || len(results.List) == 0 {
		return &ast.ExprStmt{X: call}
	}
	return &ast.ReturnStmt{Results: []ast.Expr{call}}
}

type methodCandidateScore struct {
	phase              int
	totalCost          int
	exactCount         int
	expandVarargsArray bool
}

func methodCandidateScoreBetter(candidate, current methodCandidateScore) bool {
	return candidate.phase < current.phase ||
		(candidate.phase == current.phase && candidate.totalCost < current.totalCost) ||
		(candidate.phase == current.phase && candidate.totalCost == current.totalCost && candidate.exactCount > current.exactCount)
}

func methodInvocationArityApplicable(def *symbol.Definition, argumentCount int) bool {
	if def == nil {
		return false
	}
	parameterCount := len(def.Parameters)
	if parameterCount == 0 || !executionParameterIsVariadic(def, parameterCount-1) {
		return parameterCount == argumentCount
	}
	return argumentCount >= parameterCount-1
}

// findBestConstructor selects one constructor declared by scope using the same
// strict Java invocation conversions as ordinary overload resolution. Unlike a
// method lookup, constructors are never inherited: a this(...) invocation
// searches exactly the current class and super(...) searches exactly the direct
// superclass. Keeping this selector source-node based also lets null choose the
// most-specific reference overload and numeric arguments choose the closest
// legal widening before their generated Go expressions are coerced.
func findBestConstructor(scope *symbol.ClassScope, argsNode *sitter.Node, ctx Ctx, source []byte) *methodResolution {
	if scope == nil {
		return nil
	}

	var argNodes []*sitter.Node
	if argsNode != nil {
		argNodes = nodeutil.NamedChildrenOf(argsNode)
	}
	var best *methodResolution
	var bestScore methodCandidateScore
	for _, def := range scope.Methods {
		if def == nil || !def.Constructor || !methodInvocationArityApplicable(def, len(argNodes)) {
			continue
		}
		candidateTypeParams := append([]string{}, scope.TypeParameterNames()...)
		candidateTypeParams = append(candidateTypeParams, def.TypeParameterNames()...)
		score, applicable := scoreMethodCandidate(def, scope, candidateTypeParams, argNodes, ctx, source)
		if !applicable {
			continue
		}
		candidate := &methodResolution{def: def, owner: scope, expandVarargsArray: score.expandVarargsArray}
		if best == nil || methodCandidateScoreBetter(score, bestScore) ||
			(score.phase == bestScore.phase && score.totalCost == bestScore.totalCost && score.exactCount == bestScore.exactCount && methodResolutionMoreSpecific(candidate, best, ctx)) {
			best = candidate
			bestScore = score
		}
	}
	return best
}

// findBestMethodInHierarchy selects the applicable user-defined overload for a
// method invocation. Java overloads are emitted as distinct Go names during the
// symbol-resolution pass, so every call site must recover the matching
// definition from the argument expressions' Java types before generating the
// call. Unknown expression types remain applicable (and preserve declaration
// order as a conservative fallback), while known incompatible types eliminate a
// candidate entirely.
func findBestMethodInHierarchy(
	start *symbol.ClassScope,
	methodName string,
	argsNode *sitter.Node,
	allowInstance bool,
	allowStatic bool,
	ctx Ctx,
	source []byte,
) *methodResolution {
	if start == nil {
		return nil
	}

	var argNodes []*sitter.Node
	if argsNode != nil {
		argNodes = nodeutil.NamedChildrenOf(argsNode)
	}
	var best *methodResolution
	var bestScore methodCandidateScore
	considerScope := func(scope *symbol.ClassScope, inheritedInterface bool) {
		for _, def := range scope.Methods {
			if def == nil || def.Constructor || def.OriginalName != methodName || !methodInvocationArityApplicable(def, len(argNodes)) {
				continue
			}
			// Java interface static and private methods are not inherited by
			// implementing classes or child interfaces. A direct lookup in the
			// declaring interface still uses the ordinary hierarchy pass.
			if inheritedInterface && (def.IsStatic || def.IsPrivate) {
				continue
			}
			if (def.IsStatic && !allowStatic) || (!def.IsStatic && !allowInstance) {
				continue
			}

			candidateTypeParams := append([]string{}, scope.TypeParameterNames()...)
			candidateTypeParams = append(candidateTypeParams, def.TypeParameterNames()...)
			score, applicable := scoreMethodCandidate(def, scope, candidateTypeParams, argNodes, ctx, source)
			if !applicable {
				continue
			}
			candidate := &methodResolution{def: def, owner: scope, expandVarargsArray: score.expandVarargsArray}
			if best == nil || methodCandidateScoreBetter(score, bestScore) ||
				(score.phase == bestScore.phase && score.totalCost == bestScore.totalCost && score.exactCount == bestScore.exactCount && methodResolutionMoreSpecific(candidate, best, ctx)) {
				best = candidate
				bestScore = score
			}
		}
	}

	classHierarchy := []*symbol.ClassScope{}
	seenClasses := map[*symbol.ClassScope]struct{}{}
	for scope := start; scope != nil; scope = resolveSuperclassScopeInDeclaringContext(ctx, scope) {
		if _, duplicate := seenClasses[scope]; duplicate {
			break
		}
		seenClasses[scope] = struct{}{}
		classHierarchy = append(classHierarchy, scope)
		considerScope(scope, false)
	}

	// Methods inherited from implemented/extended interfaces are members of the
	// receiver's Java type too. Their generated Go names carry export casing and,
	// for defaults, their implementations are promoted from the initialized
	// carrier embedded in the concrete class.
	interfaceQueue := []*symbol.ClassScope{}
	for _, scope := range classHierarchy {
		interfaceQueue = append(interfaceQueue, resolveImplementedInterfaceScopesInDeclaringContext(ctx, scope)...)
	}
	seenInterfaces := map[*symbol.ClassScope]struct{}{}
	for len(interfaceQueue) > 0 {
		current := interfaceQueue[0]
		interfaceQueue = interfaceQueue[1:]
		if current == nil {
			continue
		}
		if _, duplicate := seenInterfaces[current]; duplicate {
			continue
		}
		seenInterfaces[current] = struct{}{}
		considerScope(current, true)
		interfaceQueue = append(interfaceQueue, resolveImplementedInterfaceScopesInDeclaringContext(ctx, current)...)
	}

	return best
}

func scoreMethodCandidate(def *symbol.Definition, owner *symbol.ClassScope, candidateTypeParams []string, argNodes []*sitter.Node, ctx Ctx, source []byte) (methodCandidateScore, bool) {
	if def == nil || !methodInvocationArityApplicable(def, len(argNodes)) {
		return methodCandidateScore{}, false
	}

	parameterCount := len(def.Parameters)
	variadic := parameterCount > 0 && executionParameterIsVariadic(def, parameterCount-1)
	fixedArrayInvocation := variadic && len(argNodes) == parameterCount &&
		invocationArgumentCanTargetVarargsArray(argNodes[parameterCount-1], def.Parameters[parameterCount-1], owner, candidateTypeParams, ctx, source)

	score := methodCandidateScore{expandVarargsArray: fixedArrayInvocation}
	if variadic && !fixedArrayInvocation {
		// Java first considers fixed-arity declarations (including a varargs
		// declaration receiving one compatible array) and only then performs
		// variable-arity expansion. Keep that phase boundary stronger than the
		// per-argument conversion costs used within one phase.
		score.phase = 1
	}
	for index, argNode := range argNodes {
		parameterIndex := index
		if variadic && parameterIndex >= parameterCount-1 {
			parameterIndex = parameterCount - 1
		}
		if parameterIndex < 0 || parameterIndex >= parameterCount || def.Parameters[parameterIndex] == nil {
			return methodCandidateScore{}, false
		}
		parameter := def.Parameters[parameterIndex]
		// Parameter types are declared in the callee's file, not the caller's.
		// Preserve that package provenance before resolving reference conversions;
		// otherwise an unqualified imported type such as Rule<T> becomes invisible
		// when Engine<T>.addRule is invoked from a different package.
		expectedType := qualifyJavaTypeInDeclaringContext(parameter.OriginalType, owner)
		if fixedArrayInvocation && index == parameterCount-1 {
			expectedType += "[]"
		}
		cost, exact, applicable := javaInvocationConversionCost(argNode, expectedType, candidateTypeParams, ctx, source)
		if !applicable {
			return methodCandidateScore{}, false
		}
		score.totalCost += cost
		if exact {
			score.exactCount++
		}
	}
	return score, true
}

func invocationArgumentCanTargetVarargsArray(
	argNode *sitter.Node,
	parameter *symbol.Definition,
	owner *symbol.ClassScope,
	candidateTypeParams []string,
	ctx Ctx,
	source []byte,
) bool {
	if argNode == nil || parameter == nil {
		return false
	}
	unwrapped := unwrapParenthesizedExpressionNode(argNode)
	if unwrapped != nil && unwrapped.Type() == "null_literal" {
		return true
	}
	inferenceCtx := ctx.Clone()
	inferenceCtx.expectedType = ""
	inferenceCtx.expectedTypeRoot = nil
	actualType, known := inferExprJavaType(argNode, inferenceCtx, source)
	if !known || strings.TrimSpace(actualType) == "" {
		return false
	}
	if actualType == ternaryNullJavaType {
		return true
	}
	if _, rank := javaArrayTypeParts(actualType); rank == 0 {
		return false
	}
	expectedType := qualifyJavaTypeInDeclaringContext(parameter.OriginalType, owner) + "[]"
	_, _, applicable := javaInvocationConversionCost(argNode, expectedType, candidateTypeParams, ctx, source)
	return applicable
}

// methodResolutionMoreSpecific applies Java's most-specific tie-break after
// applicability scoring. It is especially important for null, which converts
// to every reference type with the same cost: pick(Mid) must win over
// pick(Parent) when Mid extends Parent.
func methodResolutionMoreSpecific(candidate, current *methodResolution, ctx Ctx) bool {
	if candidate == nil || candidate.def == nil || current == nil || current.def == nil ||
		len(candidate.def.Parameters) != len(current.def.Parameters) {
		return false
	}

	strict := false
	for index := range candidate.def.Parameters {
		candidateParam := candidate.def.Parameters[index]
		currentParam := current.def.Parameters[index]
		if candidateParam == nil || currentParam == nil {
			return false
		}
		candidateType := candidateParam.OriginalType
		if candidate.expandVarargsArray && executionParameterIsVariadic(candidate.def, index) {
			candidateType += "[]"
		}
		currentType := currentParam.OriginalType
		if current.expandVarargsArray && executionParameterIsVariadic(current.def, index) {
			currentType += "[]"
		}
		candidateType = qualifyJavaTypeInDeclaringContext(candidateType, candidate.owner)
		currentType = qualifyJavaTypeInDeclaringContext(currentType, current.owner)
		atLeastAsSpecific, parameterStrict := javaParameterAtLeastAsSpecific(candidateType, currentType, ctx)
		if !atLeastAsSpecific {
			return false
		}
		strict = strict || parameterStrict
	}
	return strict
}

func javaParameterAtLeastAsSpecific(candidateType, currentType string, ctx Ctx) (atLeastAsSpecific bool, strict bool) {
	candidateType = strings.TrimSpace(candidateType)
	currentType = strings.TrimSpace(currentType)
	if normalizeJavaReferenceType(candidateType) == normalizeJavaReferenceType(currentType) {
		return true, false
	}

	candidatePrimitive, candidateIsPrimitive := javaPrimitiveType(candidateType)
	currentPrimitive, currentIsPrimitive := javaPrimitiveType(currentType)
	if candidateIsPrimitive || currentIsPrimitive {
		return candidateIsPrimitive && currentIsPrimitive && candidatePrimitive == currentPrimitive, false
	}

	candidateComponent, candidateArray := javaArrayComponentType(candidateType)
	currentComponent, currentArray := javaArrayComponentType(currentType)
	if candidateArray || currentArray {
		if candidateArray && currentArray {
			return javaParameterAtLeastAsSpecific(candidateComponent, currentComponent, ctx)
		}
		currentBase, _ := parseJavaTypeString(currentType)
		if candidateArray && stripJavaQualifier(currentBase) == "Object" {
			return true, true
		}
		return false, false
	}

	candidateBase, candidateArgs := parseJavaTypeString(candidateType)
	currentBase, currentArgs := parseJavaTypeString(currentType)
	if stripJavaQualifier(currentBase) == "Object" && stripJavaQualifier(candidateBase) != "Object" {
		return true, true
	}
	if sameJavaRawType(candidateBase, currentBase) {
		// Generic reference types are invariant. Only identical concrete argument
		// lists (handled above) or a raw form can be ordered safely here.
		if len(candidateArgs) == 0 || len(currentArgs) == 0 {
			return true, len(candidateArgs) > 0 && len(currentArgs) == 0
		}
		return false, false
	}

	candidateScope := resolveClassScopeByQualifiedName(ctx, candidateBase)
	currentScope := resolveClassScopeByQualifiedName(ctx, currentBase)
	if distance, assignable := javaReferenceTypeDistance(candidateScope, currentScope, ctx); assignable {
		return true, distance > 0
	}
	return false, false
}

func javaArrayComponentType(javaType string) (string, bool) {
	javaType = strings.TrimSpace(javaType)
	if !strings.HasSuffix(javaType, "[]") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimSuffix(javaType, "[]")), true
}

func javaArrayCreationJavaType(node *sitter.Node, source []byte) (string, int) {
	if node == nil || node.Type() != "array_creation_expression" {
		return "", 0
	}
	typeNode := node.ChildByFieldName("type")
	if typeNode == nil {
		return "", 0
	}
	dimensions := 0
	for _, child := range nodeutil.NamedChildrenOf(node) {
		switch child.Type() {
		case "dimensions_expr":
			dimensions++
		case "dimensions":
			dimensions += strings.Count(child.Content(source), "[")
		}
	}
	if dimensions == 0 {
		return "", 0
	}
	return typeNode.Content(source) + strings.Repeat("[]", dimensions), dimensions
}

// javaInvocationConversionCost models the strict (non-varargs) portion of Java
// method-invocation conversion. Exact primitive/reference matches cost zero;
// legal primitive widening follows Java's byte/short/char/int/long/float/double
// graph; and reference upcasts are less preferred than exact reference matches.
// Boxing is deliberately not folded into primitive matching because Integer and
// int are distinct overloads in Java.
func javaInvocationConversionCost(argNode *sitter.Node, expectedType string, candidateTypeParams []string, ctx Ctx, source []byte) (cost int, exact bool, applicable bool) {
	expectedType = strings.TrimSpace(expectedType)
	if expectedType == "" {
		return 0, false, true
	}

	unwrappedArg := unwrapParenthesizedExpressionNode(argNode)
	if unwrappedArg != nil && unwrappedArg.Type() == "null_literal" {
		if _, primitive := javaPrimitiveType(expectedType); primitive {
			return 0, false, false
		}
		return 32, false, true
	}

	// Overload selection happens before a parameter target is chosen. Inherited
	// context from the invocation's enclosing return/assignment must not affect an
	// argument's standalone type. Reference conditionals whose every arm is null
	// retain the null type here so the normal most-specific tie-break can choose
	// String over Object, just as it does for a direct null argument.
	inferenceCtx := ctx.Clone()
	inferenceCtx.expectedType = ""
	inferenceCtx.expectedTypeRoot = nil
	actualType, known := "", false
	if unwrappedArg != nil && unwrappedArg.Type() == "ternary_expression" {
		actualType, known = inferStandaloneTernaryResultJavaType(unwrappedArg, inferenceCtx, source)
		if actualType == ternaryNullJavaType {
			if _, primitive := javaPrimitiveType(expectedType); primitive {
				return 0, false, false
			}
			return 32, false, true
		}
		if classifyTernaryExpression(unwrappedArg, inferenceCtx, source) == ternaryReferenceExpression &&
			ternaryCanTargetJavaType(unwrappedArg, actualType, expectedType, inferenceCtx, source) {
			// Continue with the standalone type when it already describes the
			// branches; this preserves exact String and normal reference-upcast costs.
			// A currently unknown poly expression remains eligible for the target.
			if !known || strings.TrimSpace(actualType) == "" {
				return 48, false, true
			}
		}
	} else {
		actualType, known = inferExprJavaType(argNode, inferenceCtx, source)
	}
	if !known || strings.TrimSpace(actualType) == "" {
		// Lambdas, method references, and currently-unmodelled expressions need the
		// selected parameter type as parsing context. Keep them eligible and let an
		// otherwise better-known overload win.
		return 64, false, true
	}

	expectedBase, _ := parseJavaTypeString(expectedType)
	if containsString(candidateTypeParams, stripJavaQualifier(expectedBase)) {
		// A bare candidate type parameter is inferred from the argument at this call
		// site (e.g. <T> T id(T value)). Treat the inferred parameter as an exact
		// match so generic methods remain in the overload set and explicit Go type
		// arguments are applied to their generated helper/function.
		return 0, true, true
	}
	// A generic array formal is inferred at the Java level even though its
	// generated Go parameter is the non-generic *ReferenceArray ABI. Keep the
	// candidate applicable when stripping the formal array rank exposes one of
	// its type parameters; call lowering will then emit the inferred Go argument.
	expectedArrayBase, expectedArrayRank := javaArrayTypeParts(expectedType)
	if expectedArrayRank > 0 && containsString(candidateTypeParams, stripJavaQualifier(expectedArrayBase)) {
		actualArrayBase, actualArrayRank := javaArrayTypeParts(actualType)
		if actualArrayRank >= expectedArrayRank {
			if actualArrayRank == expectedArrayRank {
				if _, primitive := javaPrimitiveType(actualArrayBase); primitive {
					return 0, false, false
				}
			}
			return 0, true, true
		}
		return 0, false, false
	}

	actualComponent, actualArray := javaArrayComponentType(actualType)
	expectedComponent, expectedArray := javaArrayComponentType(expectedType)
	if actualArray || expectedArray {
		if actualArray && expectedArray {
			actualPrimitive, actualComponentPrimitive := javaPrimitiveType(actualComponent)
			expectedPrimitive, expectedComponentPrimitive := javaPrimitiveType(expectedComponent)
			if actualComponentPrimitive || expectedComponentPrimitive {
				// Primitive array components are invariant. Equality was handled
				// above, before reaching this branch.
				return 0, false, actualComponentPrimitive && expectedComponentPrimitive && actualPrimitive == expectedPrimitive
			}
			if assignable, _ := javaParameterAtLeastAsSpecific(actualType, expectedType, ctx); assignable {
				return 16, false, true
			}
			return 0, false, false
		}
		if actualArray {
			expectedBase, _ := parseJavaTypeString(expectedType)
			switch stripJavaQualifier(expectedBase) {
			case "Object", "Cloneable", "Serializable":
				return 24, false, true
			}
		}
		return 0, false, false
	}

	actualPrimitive, actualIsPrimitive := javaPrimitiveType(actualType)
	expectedPrimitive, expectedIsPrimitive := javaPrimitiveType(expectedType)
	if actualIsPrimitive || expectedIsPrimitive {
		if !actualIsPrimitive || !expectedIsPrimitive {
			return 0, false, false
		}
		if actualPrimitive == expectedPrimitive {
			return 0, true, true
		}
		if distance, ok := javaPrimitiveWideningDistance(actualPrimitive, expectedPrimitive); ok {
			return distance, false, true
		}
		return 0, false, false
	}

	actualReference := normalizeJavaReferenceType(actualType)
	expectedReference := normalizeJavaReferenceType(expectedType)
	if actualReference == expectedReference {
		return 0, true, true
	}

	actualBase, actualArgs := parseJavaTypeString(actualType)
	// Parameterized Java types are invariant, but a candidate's own type
	// parameters are inferred/bound at the invocation site. Thus List<String> is
	// applicable to List<T>, while List<String> is not treated as List<Object>.
	// This check is also useful for runtime-modelled types such as List whose
	// class scope is intentionally absent from the user symbol table.
	if sameJavaRawType(actualBase, expectedBase) {
		_, expectedArgs := parseJavaTypeString(expectedType)
		if javaGenericArgumentsApplicable(actualArgs, expectedArgs, candidateTypeParams) {
			return 4, false, true
		}
		return 0, false, false
	}
	if stripJavaQualifier(expectedBase) == "Object" {
		return 24, false, true
	}
	actualScope := resolveClassScopeByQualifiedName(ctx, actualBase)
	expectedScope := resolveClassScopeByQualifiedName(ctx, expectedBase)
	if distance, assignable := javaReferenceTypeDistance(actualScope, expectedScope, ctx); assignable {
		return 16 + distance, false, true
	}

	// A type parameter is reference-like unless it has a primitive instantiation,
	// which Java generics do not permit. Keep it applicable when its concrete type
	// is unavailable at this call site.
	if isInScopeJavaTypeParameter(actualBase, ctx) || isInScopeJavaTypeParameter(expectedBase, ctx) {
		return 48, false, true
	}
	return 0, false, false
}

func javaPrimitiveType(javaType string) (string, bool) {
	base, _ := parseJavaTypeString(strings.TrimSpace(javaType))
	base = stripJavaQualifier(base)
	switch base {
	case "byte", "short", "char", "int", "long", "float", "double", "boolean":
		return base, true
	default:
		return "", false
	}
}

func javaPrimitiveWideningDistance(actual, expected string) (int, bool) {
	var widening []string
	switch actual {
	case "byte":
		widening = []string{"short", "int", "long", "float", "double"}
	case "short":
		widening = []string{"int", "long", "float", "double"}
	case "char":
		widening = []string{"int", "long", "float", "double"}
	case "int":
		widening = []string{"long", "float", "double"}
	case "long":
		widening = []string{"float", "double"}
	case "float":
		widening = []string{"double"}
	default:
		return 0, false
	}
	for index, candidate := range widening {
		if candidate == expected {
			return index + 1, true
		}
	}
	return 0, false
}

func normalizeJavaReferenceType(javaType string) string {
	javaType = strings.TrimSpace(javaType)
	arraySuffix := ""
	for strings.HasSuffix(javaType, "[]") {
		arraySuffix += "[]"
		javaType = strings.TrimSpace(javaType[:len(javaType)-2])
	}
	base, args := parseJavaTypeString(javaType)
	base = stripJavaQualifier(base)
	for index := range args {
		args[index] = normalizeJavaReferenceType(args[index])
	}
	if len(args) > 0 {
		return base + "<" + strings.Join(args, ",") + ">" + arraySuffix
	}
	return base + arraySuffix
}

// sameJavaRawType compares reference-type bases without conflating two
// different fully-qualified classes that happen to share a short name. One side
// may be qualified while the other uses an import-visible short spelling.
func sameJavaRawType(actual, expected string) bool {
	actual = strings.TrimSpace(actual)
	expected = strings.TrimSpace(expected)
	if actual == expected {
		return true
	}
	if strings.Contains(actual, ".") && strings.Contains(expected, ".") {
		return false
	}
	return stripJavaQualifier(actual) == stripJavaQualifier(expected)
}

// javaGenericArgumentsApplicable applies Java's invariant generic argument
// matching while treating the candidate method/class type parameters as values
// to be inferred at this invocation. Raw types remain applicable, matching
// Java's unchecked-conversion behavior; concrete mismatched arguments do not.
func javaGenericArgumentsApplicable(actual, expected, candidateTypeParams []string) bool {
	if len(actual) == 0 || len(expected) == 0 {
		return true
	}
	if len(actual) != len(expected) {
		return false
	}

	for index := range expected {
		actualArg := strings.TrimSpace(actual[index])
		expectedArg := strings.TrimSpace(expected[index])
		if normalizeJavaReferenceType(actualArg) == normalizeJavaReferenceType(expectedArg) {
			continue
		}
		if strings.HasPrefix(expectedArg, "?") || strings.HasPrefix(actualArg, "?") {
			// Wildcard-bound applicability is validated by javac for project input.
			// Keeping it eligible here is preferable to losing the method symbol and
			// emitting the original Java selector spelling.
			continue
		}

		actualBase, actualNested := parseJavaTypeString(actualArg)
		expectedBase, expectedNested := parseJavaTypeString(expectedArg)
		if (len(actualNested) == 0 && containsString(candidateTypeParams, stripJavaQualifier(actualBase))) ||
			(len(expectedNested) == 0 && containsString(candidateTypeParams, stripJavaQualifier(expectedBase))) {
			continue
		}
		if !sameJavaRawType(actualBase, expectedBase) ||
			!javaGenericArgumentsApplicable(actualNested, expectedNested, candidateTypeParams) {
			return false
		}
	}
	return true
}

func isInScopeJavaTypeParameter(typeName string, ctx Ctx) bool {
	typeName = stripJavaQualifier(strings.TrimSpace(typeName))
	for _, candidate := range inScopeTypeParameters(ctx) {
		if candidate == typeName {
			return true
		}
	}
	return false
}

func javaReferenceTypeAssignable(actual, expected *symbol.ClassScope, ctx Ctx) bool {
	_, assignable := javaReferenceTypeDistance(actual, expected, ctx)
	return assignable
}

// javaReferenceTypeDistance returns the shortest superclass/interface distance
// from actual to expected. The distance lets overload resolution prefer the
// nearest legal reference conversion (Child -> Mid) over a more distant one
// (Child -> Parent), independent of declaration order.
func javaReferenceTypeDistance(actual, expected *symbol.ClassScope, ctx Ctx) (int, bool) {
	if actual == nil || expected == nil {
		return 0, false
	}
	type referenceStep struct {
		scope    *symbol.ClassScope
		distance int
	}
	queue := []referenceStep{{scope: actual}}
	seen := map[*symbol.ClassScope]struct{}{}
	for len(queue) > 0 {
		step := queue[0]
		queue = queue[1:]
		scope := step.scope
		if scope == nil {
			continue
		}
		if scope == expected {
			return step.distance, true
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		// Superclasses and implemented interfaces are written in this class's
		// declaring file. Resolve their unqualified names against that file's
		// imports rather than the unrelated invocation site's imports.
		declarationCtx := ctx.Clone()
		if file := findFileScopeForClassScope(scope); file != nil {
			declarationCtx.currentFile = file
		}
		for _, implemented := range scope.ImplementedInterfaces {
			base, _ := parseJavaTypeString(implemented)
			if next := resolveClassScopeByQualifiedName(declarationCtx, base); next != nil {
				queue = append(queue, referenceStep{scope: next, distance: step.distance + 1})
			}
		}
		if next := resolveSuperclassScope(declarationCtx, scope); next != nil {
			queue = append(queue, referenceStep{scope: next, distance: step.distance + 1})
		}
	}
	return 0, false
}

func findInstanceMethodInHierarchy(start *symbol.ClassScope, methodName string, argCount int, ctx Ctx) *methodResolution {
	seen := map[*symbol.ClassScope]struct{}{}
	for scope := start; scope != nil; scope = resolveSuperclassScopeInDeclaringContext(ctx, scope) {
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

func findFieldInHierarchy(start *symbol.ClassScope, fieldName string, ctx Ctx) *symbol.Definition {
	resolution := findFieldResolutionInHierarchy(start, fieldName, ctx)
	if resolution == nil {
		return nil
	}
	return resolution.def
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

// parseResolvedInvocationArguments applies Java's two varargs call modes to the
// generated Go ABI. Element-style calls keep their individual arguments. A
// fixed-arity Java call whose final argument is already an array instead passes
// the wrapper's native element slice and asks the eventual Go call to expand it.
func parseResolvedInvocationArguments(
	resolution *methodResolution,
	argsNode *sitter.Node,
	source []byte,
	ctx Ctx,
	expectedTypes []string,
) ([]ast.Expr, bool) {
	if resolution == nil || resolution.def == nil {
		return parseArgumentListWithExpectedTypes(argsNode, source, ctx, expectedTypes), false
	}
	def := resolution.def
	argumentCount := 0
	if argsNode != nil {
		argumentCount = int(argsNode.NamedChildCount())
	}
	if len(expectedTypes) == 0 {
		expectedTypes = definitionParameterOriginalTypes(def)
	}
	if len(def.Parameters) > 0 && executionParameterIsVariadic(def, len(def.Parameters)-1) {
		formalCount := len(def.Parameters)
		varargType := def.Parameters[formalCount-1].OriginalType
		if formalCount-1 < len(expectedTypes) && strings.TrimSpace(expectedTypes[formalCount-1]) != "" {
			varargType = expectedTypes[formalCount-1]
		}
		invocationTypes := make([]string, argumentCount)
		for index := range invocationTypes {
			switch {
			case index < formalCount-1 && index < len(expectedTypes):
				invocationTypes[index] = expectedTypes[index]
			case index >= formalCount-1:
				invocationTypes[index] = varargType
			}
		}
		if resolution.expandVarargsArray && argumentCount == formalCount {
			invocationTypes[formalCount-1] = varargType + "[]"
		}
		expectedTypes = invocationTypes
	}

	args := parseArgumentListWithExpectedTypes(argsNode, source, ctx, expectedTypes)
	if !resolution.expandVarargsArray || len(args) == 0 || len(def.Parameters) == 0 {
		return args, false
	}
	last := len(args) - 1
	componentJavaType := def.Parameters[len(def.Parameters)-1].OriginalType
	if last < len(expectedTypes) {
		if expectedComponent, ok := javaArrayComponentType(expectedTypes[last]); ok {
			componentJavaType = expectedComponent
		}
	}
	if _, primitive := javaPrimitiveType(componentJavaType); primitive {
		args[last] = stdjavaCall(ctx, "PrimitiveArrayElements", args[last])
		return args, true
	}
	componentType := javaTypeStringToGoTypeExpr(componentJavaType, inScopeTypeParameters(ctx), ctx)
	componentDescriptor, ok := javaTypeDescriptorExpr(componentJavaType, ctx)
	if !ok {
		componentDescriptor = stdjavaQualifiedExpr("ObjectTypeID", ctx)
	}
	args[last] = stdjavaGenericCall(
		ctx,
		"ReferenceArrayElements",
		[]ast.Expr{componentType},
		[]ast.Expr{args[last], componentDescriptor},
	)
	return args, true
}

func markDirectVarargsExpansion(call *ast.CallExpr, expand bool) *ast.CallExpr {
	if call != nil && expand {
		call.Ellipsis = token.Pos(1)
	}
	return call
}

func markDirectVarargsExpansionExpr(expr ast.Expr, expand bool) {
	if call, ok := expr.(*ast.CallExpr); ok {
		markDirectVarargsExpansion(call, expand)
	}
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
			argCtx.expectedTypeRoot = argNode
		}
		parsed := ParseExpr(argNode, source, argCtx)
		// Generated boxed parameters still use Go value types. String parameters
		// instead preserve null with the concrete-string sentinel.
		if expectedType != "" && usesNullableValueStorage(expectedType) && expressionAlwaysProducesNull(argNode) {
			if isJavaStringType(expectedType) {
				parsed = javaNullStringExpr()
			} else {
				parsed = zeroValueForType(javaTypeStringToGoTypeExpr(expectedType, inScopeTypeParameters(ctx), ctx))
			}
		}
		args = append(args, coerceArgumentToExpectedType(parsed, argNode, expectedType, ctx, source))
	}
	return args
}

func classInheritsFrom(child *symbol.ClassScope, expected *symbol.ClassScope, ctx Ctx) bool {
	if child == nil || expected == nil || child == expected {
		return false
	}
	seen := map[*symbol.ClassScope]struct{}{}
	for scope := child; scope != nil; scope = resolveSuperclassScopeInDeclaringContext(ctx, scope) {
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
	if expectedType == "" || argNode == nil {
		return argExpr
	}
	if isJavaStringType(expectedType) && expressionUsesNullableValueStorage(argNode, ctx, source) {
		return stdjavaCall(ctx, "StringReferenceValue", argExpr)
	}

	actualType, actualKnown := inferExprJavaType(argNode, ctx, source)
	actualPrimitive, actualIsPrimitive := javaPrimitiveType(actualType)
	expectedPrimitive, expectedIsPrimitive := javaPrimitiveType(expectedType)
	if actualKnown && actualIsPrimitive && !expectedIsPrimitive {
		if boxedPrimitive, boxed := ternaryBoxedPrimitive(expectedType); boxed && boxedPrimitive == actualPrimitive {
			if conversion := goPrimitiveConversionName(boxedPrimitive); conversion != "" {
				return &ast.CallExpr{Fun: &ast.Ident{Name: conversion}, Args: []ast.Expr{argExpr}}
			}
		}
	}
	if actualKnown && actualIsPrimitive && expectedIsPrimitive && actualPrimitive != expectedPrimitive {
		if _, widening := javaPrimitiveWideningDistance(actualPrimitive, expectedPrimitive); widening {
			if conversion := goPrimitiveConversionName(expectedPrimitive); conversion != "" {
				return &ast.CallExpr{Fun: &ast.Ident{Name: conversion}, Args: []ast.Expr{argExpr}}
			}
		}
	}

	if ctx.currentFile == nil {
		return argExpr
	}

	expectedBase, _ := parseJavaTypeString(expectedType)
	expectedScope := resolveClassScopeByQualifiedName(ctx, expectedBase)
	if expectedScope == nil || expectedScope.IsInterface || expectedScope.IsAbstract || expectedScope.Class == nil {
		return argExpr
	}

	if !actualKnown || strings.TrimSpace(actualType) == "" {
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

// boxPrimitiveForObject pins Java's default-width int before it enters a Go
// interface value. Without the conversion, an integer literal defaults to the
// host-sized Go int, so a later `instanceof Integer` (represented as int32) does
// not observe the Java runtime type. Other primitive expressions already carry
// their Java width when parsed (long/float) or have the same Go default type.
func boxPrimitiveForObject(expr ast.Expr, exprNode *sitter.Node, expectedType string, ctx Ctx, source []byte) ast.Expr {
	expectedBase, _ := parseJavaTypeString(expectedType)
	if stripJavaQualifier(expectedBase) != "Object" || exprNode == nil {
		return expr
	}
	javaType, ok := inferExprJavaType(exprNode, ctx, source)
	if !ok {
		return expr
	}
	canonical, numeric := canonicalJavaNumericType(javaType)
	if !numeric || canonical != "int" {
		return expr
	}
	return &ast.CallExpr{Fun: &ast.Ident{Name: "int32"}, Args: []ast.Expr{expr}}
}

func goPrimitiveConversionName(javaType string) string {
	switch javaType {
	case "byte":
		return "int8"
	case "short":
		return "int16"
	case "char", "int":
		return "int32"
	case "long":
		return "int64"
	case "float":
		return "float32"
	case "double":
		return "float64"
	default:
		return ""
	}
}

func isExternalRunnableType(javaType string, ctx Ctx) bool {
	baseType, _ := parseJavaTypeString(strings.TrimSpace(javaType))
	return stripJavaQualifier(baseType) == "Runnable" && resolveClassScopeByQualifiedName(ctx, baseType) == nil
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

func inferLambdaParameterJavaTypes(ctx Ctx, parameterCount int) []string {
	if parameterCount <= 0 {
		return nil
	}

	method, typeBindings := resolveFunctionalInterfaceMethod(ctx, ctx.expectedType)
	if method == nil || len(method.Parameters) != parameterCount {
		return nil
	}

	types := make([]string, len(method.Parameters))
	for ind, param := range method.Parameters {
		if param == nil {
			continue
		}
		types[ind] = substituteJavaTypeParams(param.OriginalType, typeBindings)
	}
	return types
}

func inferLambdaParameterTypeExprs(ctx Ctx, parameterCount int) []ast.Expr {
	javaTypes := inferLambdaParameterJavaTypes(ctx, parameterCount)
	if len(javaTypes) == 0 {
		return nil
	}

	inScopeParams := inScopeTypeParameters(ctx)
	types := make([]ast.Expr, len(javaTypes))
	for ind, javaType := range javaTypes {
		if strings.TrimSpace(javaType) == "" {
			types[ind] = &ast.Ident{Name: "any"}
			continue
		}
		types[ind] = javaTypeStringToGoTypeExpr(javaType, inScopeParams, ctx)
	}
	return types
}

// contextWithLambdaParameters returns a context whose local scope includes the
// lambda's own bindings ahead of enclosing method bindings. The scope is copied,
// not mutated, because lambda parameters exist only inside this expression.
func contextWithLambdaParameters(ctx Ctx, parameters *sitter.Node, inferredTypes []string, returnType string, source []byte) Ctx {
	lambdaCtx := ctx.Clone()
	parameterNodes := []*sitter.Node{}
	if parameters != nil {
		parameterNodes = nodeutil.NamedChildrenOf(parameters)
		if parameters.Type() != "formal_parameters" && parameters.Type() != "inferred_parameters" {
			parameterNodes = []*sitter.Node{parameters}
		}
	}

	definitions := make([]*symbol.Definition, 0, len(parameterNodes))
	for ind, parameter := range parameterNodes {
		if parameter == nil {
			continue
		}
		nameNode := parameter
		if candidate := parameter.ChildByFieldName("name"); candidate != nil {
			nameNode = candidate
		}
		javaType := ""
		if ind < len(inferredTypes) {
			javaType = inferredTypes[ind]
		}
		if typeNode := parameter.ChildByFieldName("type"); typeNode != nil {
			javaType = typeNode.Content(source)
		}
		name := nameNode.Content(source)
		definitions = append(definitions, &symbol.Definition{
			OriginalName: name,
			Name:         sanitizeGoIdent(name),
			OriginalType: javaType,
		})
	}
	local := symbol.Definition{}
	if ctx.localScope != nil {
		local = *ctx.localScope
		local.Parameters = append([]*symbol.Definition(nil), ctx.localScope.Parameters...)
		local.Children = append([]*symbol.Definition(nil), ctx.localScope.Children...)
	}
	local.OriginalType = returnType
	local.Constructor = false
	local.Parameters = append(definitions, local.Parameters...)
	lambdaCtx.localScope = &local
	// A Java lambda is a function/control-flow boundary. Returns and any lowered
	// closure state inside it belong to the SAM invocation, never to an enclosing
	// method's try/finally or synchronized statement.
	lambdaCtx.tryReturnTarget = nil
	lambdaCtx.tryControlBoundary = nil
	return lambdaCtx
}

// capturedLocal describes an enclosing local/parameter referenced inside an
// anonymous or local class body, captured as a synthesized struct field.
type capturedLocal struct {
	name    string
	goType  ast.Expr
	javaDef *symbol.Definition
}

// collectCapturedLocals walks a body subtree and collects the enclosing locals
// and parameters it references, in first-seen order. These become fields of a
// synthesized struct so the class can close over them.
func collectCapturedLocals(body *sitter.Node, source []byte, ctx Ctx) []capturedLocal {
	if ctx.localScope == nil || body == nil {
		return nil
	}

	// Names declared inside the anonymous/local body (method params, locals, loop
	// variables, catch params, the class's own fields) are NOT captures — only
	// references to enclosing locals are. The enclosing method's symbol scope can
	// contain these inner names because symbol parsing flattens nested scopes, so
	// exclude them explicitly.
	declaredInside := collectDeclaredNames(body, source)

	seen := map[string]struct{}{}
	var captured []capturedLocal

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "identifier" {
			name := n.Content(source)
			if _, inside := declaredInside[name]; inside {
				return
			}
			if _, ok := seen[name]; !ok {
				if def := ctx.localScope.FindVariable(name); def != nil {
					seen[name] = struct{}{}
					captured = append(captured, capturedLocal{
						name:    def.Name,
						goType:  javaTypeStringToGoTypeExpr(def.OriginalType, inScopeTypeParameters(ctx), ctx),
						javaDef: def,
					})
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(body)
	return captured
}

// collectDeclaredNames gathers every identifier introduced as a binding within a
// subtree: local variable declarations, formal/spread/catch parameters, and
// for-loop / enhanced-for variables. These shadow or are local to the body and
// must not be treated as captures of the enclosing scope.
func collectDeclaredNames(node *sitter.Node, source []byte) map[string]struct{} {
	names := map[string]struct{}{}
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		switch n.Type() {
		case "variable_declarator":
			if nameNode := n.ChildByFieldName("name"); nameNode != nil {
				names[nameNode.Content(source)] = struct{}{}
			}
		case "formal_parameter", "spread_parameter", "catch_formal_parameter":
			if nameNode := n.ChildByFieldName("name"); nameNode != nil {
				names[nameNode.Content(source)] = struct{}{}
			}
		case "enhanced_for_statement":
			if nameNode := n.ChildByFieldName("name"); nameNode != nil {
				names[nameNode.Content(source)] = struct{}{}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(node)
	return names
}

// anonymousClassDeclaredFields extracts the instance fields declared directly in
// an anonymous/local class body, returning both the symbol definitions (for
// in-body resolution through the receiver) and the Go struct fields to emit.
func syntheticFieldDeclarators(fieldDeclaration *sitter.Node) []*sitter.Node {
	if fieldDeclaration == nil {
		return nil
	}
	var declarators []*sitter.Node
	for _, child := range nodeutil.NamedChildrenOf(fieldDeclaration) {
		if child.Type() == "variable_declarator" {
			declarators = append(declarators, child)
		}
	}
	if len(declarators) == 0 {
		if declarator := fieldDeclaration.ChildByFieldName("declarator"); declarator != nil {
			declarators = append(declarators, declarator)
		}
	}
	return declarators
}

func anonymousClassDeclaredFields(classBody *sitter.Node, source []byte, ctx Ctx) ([]*symbol.Definition, []*ast.Field) {
	var defs []*symbol.Definition
	var astFields []*ast.Field
	for _, member := range nodeutil.NamedChildrenOf(classBody) {
		if member.Type() != "field_declaration" {
			continue
		}
		typeNode := member.ChildByFieldName("type")
		if typeNode == nil {
			continue
		}
		for _, declarator := range syntheticFieldDeclarators(member) {
			fieldNameNode := declarator.ChildByFieldName("name")
			if fieldNameNode == nil {
				continue
			}
			fieldName := fieldNameNode.Content(source)
			defs = append(defs, &symbol.Definition{
				OriginalName: fieldName,
				Name:         fieldName,
				OriginalType: typeNode.Content(source),
			})
			astFields = append(astFields, &ast.Field{
				Names: []*ast.Ident{{Name: fieldName}},
				Type:  javaTypeStringToGoTypeExpr(typeNode.Content(source), inScopeTypeParameters(ctx), ctx),
			})
		}
	}
	return defs, astFields
}

// textBlockLiteral converts a Java text block (triple-quoted string) into a Go
// string literal expression, applying JLS incidental-whitespace stripping. It
// emits a raw string literal (backticks) when the content has no backticks;
// otherwise it falls back to a double-quoted interpreted literal.
func textBlockLiteral(raw string) ast.Expr {
	content := stripTextBlockIncidentalWhitespace(raw)

	if !strings.ContainsRune(content, '`') {
		return &ast.BasicLit{Kind: token.STRING, Value: "`" + content + "`"}
	}
	return &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(content)}
}

// stripTextBlockIncidentalWhitespace implements the JLS text-block algorithm:
// remove the opening/closing delimiters, strip the common leading indentation
// (the minimum across all non-blank lines and the closing-delimiter line), and
// trim trailing whitespace from each line.
func stripTextBlockIncidentalWhitespace(raw string) string {
	// Strip the opening delimiter and the rest of its line (whitespace then the
	// required line terminator).
	inner := strings.TrimPrefix(raw, "\"\"\"")
	if nl := strings.IndexByte(inner, '\n'); nl >= 0 {
		inner = inner[nl+1:]
	}
	// Strip the closing delimiter.
	inner = strings.TrimSuffix(inner, "\"\"\"")

	lines := strings.Split(inner, "\n")

	// Determine the minimal indentation. Blank lines are ignored except the last
	// line (which corresponds to the closing delimiter's indentation).
	minIndent := -1
	for i, line := range lines {
		isLast := i == len(lines)-1
		if strings.TrimSpace(line) == "" && !isLast {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent < 0 {
		minIndent = 0
	}

	for i, line := range lines {
		if len(line) >= minIndent {
			line = line[minIndent:]
		} else {
			line = strings.TrimLeft(line, " \t")
		}
		// Trailing whitespace is incidental and removed.
		lines[i] = strings.TrimRight(line, " \t")
	}

	// Join preserves a trailing newline when the closing delimiter was on its own
	// line (the final element is empty), and omits it when the closing delimiter
	// followed the last content line — exactly the JLS behavior.
	return strings.Join(lines, "\n")
}

// buildSwitchExpressionIIFE lowers a switch expression into an immediately
// invoked function literal whose arms return the switch's value.
func buildSwitchExpressionIIFE(node *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	condNode := node.ChildByFieldName("condition")
	bodyNode := node.ChildByFieldName("body")
	if condNode == nil || bodyNode == nil {
		// Fall back to a stub so the rest of the file still converts.
		diag := reportUnsupported("expression", node, source, ctx)
		return &ast.CallExpr{Fun: &ast.Ident{Name: "panic"}, Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", strings.TrimPrefix(unsupportedComment(diag), "// "))}}}
	}

	// Result type of the IIFE: prefer the expected type, else any.
	var resultType ast.Expr = &ast.Ident{Name: "any"}
	if strings.TrimSpace(ctx.expectedType) != "" {
		resultType = javaTypeStringToGoTypeExpr(ctx.expectedType, inScopeTypeParameters(ctx), ctx)
	}

	switchStmt := &ast.SwitchStmt{
		Tag:  ParseExpr(condNode, source, ctx),
		Body: buildSwitchExpressionBody(bodyNode, source, ctx),
	}

	body := &ast.BlockStmt{List: []ast.Stmt{
		switchStmt,
		// Go cannot prove switch exhaustiveness, so guard the fallthrough path.
		&ast.ExprStmt{X: &ast.CallExpr{
			Fun:  &ast.Ident{Name: "panic"},
			Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: "\"unreachable switch expression\""}},
		}},
	}}

	return &ast.CallExpr{
		Fun: &ast.FuncLit{
			Type: &ast.FuncType{
				Params:  &ast.FieldList{},
				Results: &ast.FieldList{List: []*ast.Field{{Type: resultType}}},
			},
			Body: body,
		},
	}
}

// buildSwitchExpressionBody builds the case clauses for a switch expression,
// turning each arm's produced value into a `return`. Handles both arrow-form
// (`case X -> expr` / `case X -> { yield V; }`) and colon-form arms with yield.
func buildSwitchExpressionBody(bodyNode *sitter.Node, source []byte, ctx Ctx) *ast.BlockStmt {
	switchBlock := &ast.BlockStmt{}

	for _, child := range nodeutil.NamedChildrenOf(bodyNode) {
		switch child.Type() {
		case "switch_rule":
			caseExprs, isDefault, ruleBody := splitSwitchRule(child, source, ctx)
			clause := &ast.CaseClause{}
			if !isDefault {
				clause.List = caseExprs
			}
			clause.Body = switchArmReturnStmts(ruleBody, source, ctx)
			switchBlock.List = append(switchBlock.List, clause)
		case "switch_block_statement_group":
			caseExprs, isDefault, groupBody := splitSwitchGroup(child, source, ctx)
			clause := &ast.CaseClause{}
			if !isDefault {
				clause.List = caseExprs
			}
			clause.Body = switchArmReturnStmts(groupBody, source, ctx)
			switchBlock.List = append(switchBlock.List, clause)
		}
	}

	return switchBlock
}

// switchArmReturnStmts converts the body nodes of a switch-expression arm into Go
// statements where the produced value is returned. A bare expression statement
// becomes `return expr`; `yield X` becomes `return X`; other statements are
// preserved as-is.
func switchArmReturnStmts(bodyNodes []*sitter.Node, source []byte, ctx Ctx) []ast.Stmt {
	var stmts []ast.Stmt
	for _, n := range bodyNodes {
		switch n.Type() {
		case "expression_statement":
			stmts = append(stmts, &ast.ReturnStmt{Results: []ast.Expr{ParseExpr(n.NamedChild(0), source, ctx)}})
		case "block":
			stmts = append(stmts, convertYieldBlock(n, source, ctx)...)
		case "yield_statement":
			stmts = append(stmts, &ast.ReturnStmt{Results: []ast.Expr{ParseExpr(n.NamedChild(0), source, ctx)}})
		default:
			stmts = append(stmts, ParseStmt(n, source, ctx))
		}
	}
	return stmts
}

// convertYieldBlock converts the statements of a block in a switch-expression
// arm, rewriting `yield X` into `return X`.
func convertYieldBlock(block *sitter.Node, source []byte, ctx Ctx) []ast.Stmt {
	var stmts []ast.Stmt
	for _, n := range nodeutil.NamedChildrenOf(block) {
		if n.Type() == "yield_statement" {
			stmts = append(stmts, &ast.ReturnStmt{Results: []ast.Expr{ParseExpr(n.NamedChild(0), source, ctx)}})
			continue
		}
		if parsed := TryParseStmts(n, source, ctx); parsed != nil {
			stmts = append(stmts, parsed...)
		} else {
			stmts = append(stmts, ParseStmt(n, source, ctx))
		}
	}
	return stmts
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

func localClassConstructors(classBody *sitter.Node) []*sitter.Node {
	var constructors []*sitter.Node
	for _, child := range nodeutil.NamedChildrenOf(classBody) {
		if child.Type() == "constructor_declaration" {
			constructors = append(constructors, child)
		}
	}
	return constructors
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
	// A synchronized anonymous implementation needs a stable Java object as its
	// monitor. The closure adapter has no anonymous-object identity of its own,
	// so let the synthesized-struct lowering below model this case instead.
	if declarationHasModifier(implMethod, "synchronized") {
		return nil
	}

	// Build a function literal from the implementing method's signature and body.
	// The body is parsed in a scope that knows the method's parameters so captured
	// locals from the enclosing method resolve naturally as closure captures.
	implScope := scopeForAnonymousMethod(implMethod, method, source)
	methodCtx := ctx.Clone()
	methodCtx.localScope = implScope
	executionName := executionParameterName(implMethod, source, methodCtx)
	methodCtx.executionContextName = executionName

	params := ParseNode(implMethod.ChildByFieldName("parameters"), source, methodCtx).(*ast.FieldList)
	params.List = append([]*ast.Field{executionParameterField(executionName, ctx)}, params.List...)

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

	return wrapLambdaWithFunctionalInterfaceAdapter(funcLit, supertype, true, ctx)
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

// lowerAnonymousClassToStruct synthesizes a uniquely-named, file-scoped struct
// for an anonymous class that is not a simple SAM implementation (it declares
// multiple methods, fields, or extends a class). Referenced enclosing locals are
// captured as struct fields; the supertype is embedded (an interface by name, a
// class as *Super). The creation site becomes a composite literal initializing
// the captured fields. Returns nil if the anonymous class cannot be modeled this
// way (e.g. the collector is not initialized).
func lowerAnonymousClassToStruct(node, objectType, classBody *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	if ctx.hoistedDecls == nil || objectType == nil {
		return nil
	}

	supertype := objectType.Content(source)
	baseType, _ := parseJavaTypeString(supertype)
	superScope := resolveClassScopeByQualifiedName(ctx, baseType)

	// Name the synthesized struct uniquely within the file.
	prefix := ""
	if ctx.className != "" {
		prefix = ctx.className
	}
	structName := fmt.Sprintf("%sAnon%d", prefix, ctx.nextAnonClassIndex())

	declaredFieldDefs, declaredAstFields := anonymousClassDeclaredFields(classBody, source, ctx)
	captured := collectCapturedLocals(classBody, source, ctx)

	// Build the struct fields: the supertype, declared fields, then captured
	// locals. User-defined supertypes are embedded (an interface by name, a class
	// as *Super). Runnable is satisfied structurally by the exported Run method;
	// other concrete stdjava-backed supertypes retain their runtime methods by
	// embedding the corresponding runtime type.
	fields := &ast.FieldList{}
	if superScope != nil && superScope.Class != nil {
		if superScope.IsInterface {
			fields.List = append(fields.List, &ast.Field{Type: &ast.Ident{Name: superScope.Class.Name}})
		} else {
			fields.List = append(fields.List, &ast.Field{Type: &ast.StarExpr{X: &ast.Ident{Name: superScope.Class.Name}}})
		}
	} else if stripJavaQualifier(baseType) == "Runnable" {
		// Go interface satisfaction is structural: the exported Run method emitted
		// below is sufficient. Embedding stdjava.Runnable would add a nil interface
		// field and needlessly register a runtime import.
	} else if rt, ok := stdjavaRuntimeTypeExpr(stripJavaQualifier(baseType), nil, inScopeTypeParameters(ctx), ctx); ok {
		// Concrete stdjava-backed supertypes are embedded to retain their methods.
		fields.List = append(fields.List, &ast.Field{Type: rt})
	} else {
		// Unresolved supertype: embed it by its written name as a best effort.
		fields.List = append(fields.List, &ast.Field{Type: &ast.Ident{Name: stripJavaQualifier(baseType)}})
	}
	fields.List = append(fields.List, declaredAstFields...)
	for _, cap := range captured {
		fields.List = append(fields.List, &ast.Field{
			Names: []*ast.Ident{{Name: cap.name}},
			Type:  cap.goType,
		})
	}

	ctx.addHoistedDecl(GenStruct(structName, fields))

	// Emit a method for each declared method in the anonymous body. Register the
	// complete synthetic method table before parsing any body so self-recursion
	// and calls between sibling methods resolve exactly like ordinary class
	// methods.
	bodyMethods := anonymousClassMethods(classBody)
	syntheticScope := synthAnonClassScope(structName, declaredFieldDefs, captured, bodyMethods, source, false)
	for _, methodNode := range bodyMethods {
		methodDecls := buildAnonymousStructMethod(structName, methodNode, syntheticScope, source, ctx)
		for _, methodDecl := range methodDecls {
			ctx.addHoistedDecl(methodDecl)
		}
	}

	// A non-SAM anonymous implementation has no source ClassScope of its own, but
	// it is still a real Java object that can be stored in an interface array.
	// Give it a unique nominal type and register every inherited interface edge;
	// the generated pointer already satisfies those Go interface views directly.
	if superScope != nil && superScope.IsInterface && classNeedsReferenceIdentity(superScope, ctx) {
		interfaceScopes := append([]*symbol.ClassScope{superScope}, transitiveImplementedInterfaceScopes(superScope, ctx)...)
		interfaceIDs := make([]ast.Expr, 0, len(interfaceScopes))
		seenInterfaces := make(map[*symbol.ClassScope]struct{})
		for _, interfaceScope := range interfaceScopes {
			if interfaceScope == nil || interfaceScope.Class == nil {
				continue
			}
			if _, duplicate := seenInterfaces[interfaceScope]; duplicate {
				continue
			}
			seenInterfaces[interfaceScope] = struct{}{}
			interfaceIDs = append(interfaceIDs, javaTypeIDLiteral(javaClassBinaryName(interfaceScope), ctx))
		}
		ownerID := ""
		if ctx.currentClass != nil {
			ownerID = javaClassBinaryName(ctx.currentClass)
		}
		if ownerID == "" {
			ownerID = ctx.className
		}
		for _, declaration := range syntheticReferenceIdentityDecls(
			structName,
			ownerID+"$"+structName,
			nil,
			interfaceIDs,
			ctx,
		) {
			ctx.addHoistedDecl(declaration)
		}
	}

	// Construct the value with its real Java superclass subobject initialized.
	// A bare embedded pointer has a nil method set at runtime and cannot carry
	// virtual calls back to the anonymous override.
	elts := make([]ast.Expr, 0, len(captured)+1)
	var superclassInitializer ast.Expr
	if superScope != nil && !superScope.IsInterface && superScope.Class != nil {
		superclassInitializer = anonymousSuperclassConstructorExpr(node, objectType, superScope, source, ctx)
		if superclassInitializer != nil {
			elts = append(elts, &ast.KeyValueExpr{
				Key:   &ast.Ident{Name: superScope.Class.Name},
				Value: superclassInitializer,
			})
		}
	}
	for _, cap := range captured {
		elts = append(elts, &ast.KeyValueExpr{
			Key:   &ast.Ident{Name: cap.name},
			Value: &ast.Ident{Name: cap.name},
		})
	}

	composite := &ast.UnaryExpr{
		Op: token.AND,
		X:  &ast.CompositeLit{Type: &ast.Ident{Name: structName}, Elts: elts},
	}
	if superclassInitializer == nil || !classHasSelfSetter(superScope, ctx) {
		return composite
	}

	// Use a tiny immediately-invoked constructor so the ancestor's dispatch
	// slot can point at the fully-created anonymous value before it escapes.
	instanceName := "__java2goAnonymous"
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{
			Type: &ast.StarExpr{X: &ast.Ident{Name: structName}},
		}}}},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: instanceName}},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{composite},
			},
			&ast.ExprStmt{X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.SelectorExpr{
						X:   &ast.Ident{Name: instanceName},
						Sel: &ast.Ident{Name: superScope.Class.Name},
					},
					Sel: &ast.Ident{Name: classSelfSetterName(superScope)},
				},
				Args: []ast.Expr{&ast.Ident{Name: instanceName}},
			}},
			&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: instanceName}}},
		}},
	}}
}

// anonymousSuperclassConstructorExpr lowers the constructor portion of
// `new Base(args) { ... }`. It mirrors normal object creation while leaving the
// anonymous body to the synthesized struct around that base value.
func anonymousSuperclassConstructorExpr(node, objectType *sitter.Node, superScope *symbol.ClassScope, source []byte, ctx Ctx) ast.Expr {
	if node == nil || objectType == nil || superScope == nil || superScope.Class == nil {
		return nil
	}
	argsNode := node.ChildByFieldName("arguments")
	argNodes := nodeutil.NamedChildrenOf(argsNode)
	argTypes := make([]string, len(argNodes))
	for i, argNode := range argNodes {
		if inferred, ok := inferExprJavaType(argNode, ctx, source); ok {
			argTypes[i] = inferred
		}
	}

	constructor := findMatchingConstructor(superScope, superScope.Class.OriginalName, argTypes)
	var expectedTypes []string
	constructorName := constructorFuncName(superScope)
	if constructor != nil {
		constructorName = constructor.Name
		expectedTypes = definitionParameterOriginalTypes(constructor)
	}
	if constructorName == "" {
		constructorName = defaultConstructorName(superScope.Class.Name)
	}
	args := parseArgumentListWithExpectedTypes(argsNode, source, ctx, expectedTypes)
	if executionExpr(ctx) != nil && constructorHasExecutionImplementation(constructor, superScope) {
		constructorName = executionConstructorImplementationName(constructorName, superScope)
		args = prependExecutionArgument(ctx, args)
	}

	baseType, typeArgs := parseJavaTypeString(objectType.Content(source))
	constructorExpr := qualifiedNameExpr(
		constructorName,
		resolveJavaPackageForType(ctx, baseType, superScope),
		ctx,
	)
	if len(typeArgs) > 0 {
		goTypeArgs := make([]ast.Expr, 0, len(typeArgs))
		for _, typeArg := range typeArgs {
			goTypeArgs = append(goTypeArgs, javaTypeStringToGoTypeExpr(typeArg, inScopeTypeParameters(ctx), ctx))
		}
		constructorExpr = applyTypeArguments(constructorExpr, goTypeArgs)
	}
	call := &ast.CallExpr{Fun: constructorExpr, Args: args}
	resultType := javaTypeStringToGoTypeExpr(objectType.Content(source), inScopeTypeParameters(ctx), ctx)
	return guardClassInitializationBeforeExpr(superScope, call, resultType, ctx)
}

// buildAnonymousStructMethod builds a method declaration on a synthesized
// anonymous-class struct. Captured locals are made available inside the body by
// reading them back from the receiver's fields, so the original body references
// synthAnonClassScope builds a synthetic ClassScope for a synthesized anonymous
// or local class. Both the class's own declared instance fields and its captured
// enclosing locals are registered as fields, so that references inside method
// bodies resolve through the receiver (recv.field) via the normal field
// resolution path. This is correct for mutation (e.g. n++ -> recv.n++), unlike
// unpacking captures into locals.
func synthAnonClassScope(
	structName string,
	declaredFields []*symbol.Definition,
	captured []capturedLocal,
	methodNodes []*sitter.Node,
	source []byte,
	keepOriginalMethodNames bool,
) *symbol.ClassScope {
	scope := &symbol.ClassScope{
		Class: &symbol.Definition{OriginalName: structName, Name: structName},
	}
	scope.Fields = append(scope.Fields, declaredFields...)
	for _, cap := range captured {
		if cap.javaDef != nil {
			scope.Fields = append(scope.Fields, cap.javaDef)
		}
	}
	usedMethodNames := map[string]struct{}{}
	if keepOriginalMethodNames {
		// Local-class instance members share one selector namespace after lowering
		// to Go even though Java fields and methods do not. Reserve every generated
		// field spelling before allocating method overloads. Local-class call sites
		// resolve through this synthetic scope, so they follow any renamed method.
		// Anonymous-class call sites still require separate exact-type tracking and
		// intentionally retain their previous spellings here.
		for _, field := range scope.Fields {
			if field != nil && field.Name != "" {
				usedMethodNames[field.Name] = struct{}{}
			}
		}
	}
	for _, methodNode := range methodNodes {
		// Local classes are now registered in the synthetic scope before their
		// call sites are rendered, so calls can follow the final selector name.
		// Honor Java visibility here as ordinary/anonymous classes do; in
		// particular a public method implementing an interface must be exported
		// in Go to satisfy that interface outside the concrete method set.
		method := synthAnonClassMethodDefinition(methodNode, source, false)
		if method == nil {
			continue
		}
		baseName := method.Name
		for suffix := 0; ; suffix++ {
			if _, duplicate := usedMethodNames[method.Name]; !duplicate {
				break
			}
			method.Name = baseName + strconv.Itoa(suffix)
		}
		usedMethodNames[method.Name] = struct{}{}
		// Static Java methods have no receiver and are emitted into Go's package
		// namespace. Prefix them with the already-unique synthetic type name so
		// separate local/anonymous classes may use the same Java method spelling.
		if method.IsStatic {
			method.Name = structName + "_" + method.Name
		}
		scope.Methods = append(scope.Methods, method)
	}
	return scope
}

type localConstructorCapture struct {
	capture       capturedLocal
	parameterName string
}

func localConstructorCaptureBindings(
	classBody *sitter.Node,
	captured []capturedLocal,
	source []byte,
) []localConstructorCapture {
	used := collectDeclaredNames(classBody, source)
	bindings := make([]localConstructorCapture, 0, len(captured))
	for _, capture := range captured {
		base := "__java2goCaptured" + symbol.Uppercase(sanitizeGoIdent(capture.name))
		if base == "__java2goCaptured" {
			base += "Value"
		}
		name := synchronizedUniqueLocalName(base, used)
		used[name] = struct{}{}
		bindings = append(bindings, localConstructorCapture{
			capture:       capture,
			parameterName: name,
		})
	}
	return bindings
}

func syntheticConstructorParameterDefinitions(node *sitter.Node, source []byte) []*symbol.Definition {
	if node == nil {
		return nil
	}
	parametersNode := node.ChildByFieldName("parameters")
	if parametersNode == nil {
		return nil
	}
	var parameters []*symbol.Definition
	for _, parameter := range nodeutil.NamedChildrenOf(parametersNode) {
		var nameNode, typeNode *sitter.Node
		if parameter.Type() == "spread_parameter" {
			if parameter.NamedChildCount() > 1 {
				nameNode = parameter.NamedChild(1).ChildByFieldName("name")
			}
			typeNode = parameter.NamedChild(0)
		} else {
			nameNode = parameter.ChildByFieldName("name")
			typeNode = parameter.ChildByFieldName("type")
		}
		if nameNode == nil || typeNode == nil {
			continue
		}
		name := nameNode.Content(source)
		parameters = append(parameters, &symbol.Definition{
			OriginalName: name,
			Name:         sanitizeGoIdent(name),
			OriginalType: typeNode.Content(source),
		})
	}
	return parameters
}

func synthLocalClassScope(
	classNode *sitter.Node,
	classBody *sitter.Node,
	javaName string,
	structName string,
	declaredFields []*symbol.Definition,
	captured []capturedLocal,
	captureBindings []localConstructorCapture,
	methodNodes []*sitter.Node,
	constructorNodes []*sitter.Node,
	source []byte,
) *symbol.ClassScope {
	scope := synthAnonClassScope(structName, declaredFields, captured, methodNodes, source, true)
	scope.Class.OriginalName = javaName
	scope.Class.Name = structName
	scope.Class.DeclarationNode = classNode
	if superclass := classNode.ChildByFieldName("superclass"); superclass != nil {
		if types := collectTypeNodes(superclass); len(types) > 0 {
			scope.Superclass = types[0].Content(source)
		} else {
			scope.Superclass = superclass.Content(source)
		}
	}
	if interfaces := classNode.ChildByFieldName("interfaces"); interfaces != nil {
		for _, interfaceType := range collectTypeNodes(interfaces) {
			scope.ImplementedInterfaces = append(scope.ImplementedInterfaces, interfaceType.Content(source))
		}
	}

	constructorBaseName := defaultConstructorName(structName)
	usedConstructorNames := make(map[string]struct{}, len(constructorNodes))
	for _, constructorNode := range constructorNodes {
		constructorName := constructorBaseName
		for suffix := 0; ; suffix++ {
			if _, duplicate := usedConstructorNames[constructorName]; !duplicate {
				break
			}
			constructorName = constructorBaseName + strconv.Itoa(suffix)
		}
		usedConstructorNames[constructorName] = struct{}{}
		constructor := &symbol.Definition{
			OriginalName:    javaName,
			Name:            constructorName,
			Type:            structName,
			Constructor:     true,
			HasBody:         constructorNode.ChildByFieldName("body") != nil,
			DeclarationNode: constructorNode,
			Parameters:      syntheticConstructorParameterDefinitions(constructorNode, source),
		}
		for _, binding := range captureBindings {
			if binding.capture.javaDef == nil {
				continue
			}
			captureDefinition := *binding.capture.javaDef
			captureDefinition.Name = binding.parameterName
			constructor.Children = append(constructor.Children, &captureDefinition)
		}
		scope.Methods = append(scope.Methods, constructor)
	}
	scope.HasInstanceFieldInitializers = localClassHasInstanceFieldInitializers(classBody)
	return scope
}

func synthAnonClassMethodDefinition(methodNode *sitter.Node, source []byte, keepOriginalName bool) *symbol.Definition {
	if methodNode == nil {
		return nil
	}
	nameNode := methodNode.ChildByFieldName("name")
	typeNode := methodNode.ChildByFieldName("type")
	if nameNode == nil || typeNode == nil {
		return nil
	}

	originalName := nameNode.Content(source)
	public := false
	private := false
	static := false
	if mods := methodNode.NamedChild(0); mods != nil && mods.Type() == "modifiers" {
		for _, modifier := range nodeutil.UnnamedChildrenOf(mods) {
			switch modifier.Type() {
			case "public":
				public = true
			case "private":
				private = true
			case "static":
				static = true
			}
		}
	}
	generatedName := symbol.HandleExportStatus(public, originalName)
	if keepOriginalName {
		generatedName = originalName
	}
	method := &symbol.Definition{
		OriginalName:    originalName,
		Name:            generatedName,
		OriginalType:    typeNode.Content(source),
		IsStatic:        static,
		IsPrivate:       private,
		HasBody:         methodNode.ChildByFieldName("body") != nil,
		DeclarationNode: methodNode,
	}
	for _, param := range nodeutil.NamedChildrenOf(methodNode.ChildByFieldName("parameters")) {
		var name, javaType *sitter.Node
		if param.Type() == "spread_parameter" {
			if param.NamedChildCount() > 1 {
				name = param.NamedChild(1).ChildByFieldName("name")
			}
			javaType = param.NamedChild(0)
		} else {
			name = param.ChildByFieldName("name")
			javaType = param.ChildByFieldName("type")
		}
		if name == nil || javaType == nil {
			continue
		}
		method.Parameters = append(method.Parameters, &symbol.Definition{
			OriginalName: name.Content(source),
			Name:         name.Content(source),
			OriginalType: javaType.Content(source),
		})
	}
	return method
}

// buildAnonymousStructMethod builds a method on a synthesized struct. When
// keepOriginalName is false (anonymous classes), the method name follows Go
// export rules so it satisfies the embedded interface; when true (local
// classes, whose call sites are unresolved), the original Java method name is
// kept so `m.method()` call sites match. declaredFields are the class's own
// instance fields; together with captures they form the synthetic class scope
// used to resolve field references inside the body.
func buildAnonymousStructMethod(structName string, methodNode *sitter.Node, syntheticScope *symbol.ClassScope, source []byte, ctx Ctx) []ast.Decl {
	nameNode := methodNode.ChildByFieldName("name")
	bodyNode := methodNode.ChildByFieldName("body")
	if nameNode == nil || bodyNode == nil {
		return nil
	}

	recvName := ShortName(structName)
	var methodScope *symbol.Definition
	if syntheticScope != nil {
		for _, candidate := range syntheticScope.Methods {
			if candidate != nil && candidate.DeclarationNode == methodNode {
				methodScope = candidate
				break
			}
		}
	}
	if methodScope == nil {
		return nil
	}

	// Parse the body with the synthetic class scope in context, so references to
	// the class's own fields and captured locals resolve to receiver selectors.
	methodCtx := ctx.Clone()
	methodCtx.localScope = methodScope
	methodCtx.currentClass = syntheticScope
	methodCtx.className = structName
	executionName := executionParameterName(methodNode, source, methodCtx)
	methodCtx.executionContextName = executionName

	params := ParseNode(methodNode.ChildByFieldName("parameters"), source, methodCtx).(*ast.FieldList)

	var results *ast.FieldList
	typeNode := methodNode.ChildByFieldName("type")
	if typeNode != nil && strings.TrimSpace(typeNode.Content(source)) != "void" {
		results = &ast.FieldList{
			List: []*ast.Field{
				{Type: javaTypeStringToGoTypeExpr(typeNode.Content(source), inScopeTypeParameters(ctx), ctx)},
			},
		}
	}

	body := ParseStmt(bodyNode, source, methodCtx).(*ast.BlockStmt)
	if declarationHasModifier(methodNode, "synchronized") {
		body.List = append(synchronizedMethodPrologue(methodCtx, methodScope.IsStatic, methodNode, source), body.List...)
	}
	// Synthetic local/anonymous methods bypass ParseDecl, where ordinary
	// source-backed instance methods receive their Java nil-invocation boundary.
	// Mirror that entry guard here: Go evaluates the receiver and arguments before
	// entering the method, then the guard prevents a nil pointer receiver from
	// executing a body that happens not to dereference it.
	if !methodScope.IsStatic {
		body.List = append([]ast.Stmt{instanceMethodNilReceiverGuard(recvName)}, body.List...)
	}
	if results != nil && bodyNeedsFallbackReturn(body) {
		body.List = append(body.List, &ast.ReturnStmt{
			Results: []ast.Expr{zeroValueForType(results.List[0].Type)},
		})
	}

	decl := &ast.FuncDecl{
		Name: &ast.Ident{Name: methodScope.Name},
		Type: &ast.FuncType{Params: params, Results: results},
		Body: body,
	}
	if !methodScope.IsStatic {
		decl.Recv = &ast.FieldList{List: []*ast.Field{{
			Names: []*ast.Ident{{Name: recvName}},
			Type:  &ast.StarExpr{X: &ast.Ident{Name: structName}},
		}}}
	}
	return buildExecutionAwareFuncDecls(
		decl,
		executionImplementationName(methodScope, syntheticScope),
		executionName,
		methodCtx,
	)
}

func localClassHasInstanceFieldInitializers(classBody *sitter.Node) bool {
	for _, member := range nodeutil.NamedChildrenOf(classBody) {
		if member.Type() != "field_declaration" || fieldDeclarationIsStatic(member) {
			continue
		}
		for _, declarator := range syntheticFieldDeclarators(member) {
			if declarator.ChildByFieldName("value") != nil {
				return true
			}
		}
	}
	return false
}

func localClassFieldInitializerMethodName(scope *symbol.ClassScope) string {
	used := make(map[string]struct{})
	if scope != nil {
		for _, field := range scope.Fields {
			if field != nil {
				used[field.Name] = struct{}{}
			}
		}
		for _, method := range scope.Methods {
			if method != nil {
				used[method.Name] = struct{}{}
			}
		}
	}
	candidate := fieldInitMethodName
	for suffix := 0; ; suffix++ {
		if _, collision := used[candidate]; !collision {
			return candidate
		}
		candidate = fieldInitMethodName + strconv.Itoa(suffix)
	}
}

func buildLocalClassFieldInitializerMethod(
	structName string,
	methodName string,
	classBody *sitter.Node,
	syntheticScope *symbol.ClassScope,
	source []byte,
	ctx Ctx,
) []ast.Decl {
	if classBody == nil || syntheticScope == nil || methodName == "" {
		return nil
	}

	receiverName := ShortName(structName)
	initializerScope := &symbol.Definition{OriginalName: methodName, Name: methodName}
	initializerCtx := ctx.Clone()
	initializerCtx.currentClass = syntheticScope
	initializerCtx.className = structName
	initializerCtx.localScope = initializerScope
	executionName := executionParameterName(classBody, source, initializerCtx)
	initializerCtx.executionContextName = executionName

	var statements []ast.Stmt
	for _, member := range nodeutil.NamedChildrenOf(classBody) {
		if member.Type() != "field_declaration" || fieldDeclarationIsStatic(member) {
			continue
		}
		for _, declarator := range syntheticFieldDeclarators(member) {
			nameNode := declarator.ChildByFieldName("name")
			valueNode := declarator.ChildByFieldName("value")
			if nameNode == nil || valueNode == nil {
				continue
			}
			field := syntheticScope.FindFieldByName(nameNode.Content(source))
			if field == nil {
				continue
			}
			valueCtx := initializerCtx.Clone()
			valueCtx.expectedType = field.OriginalType
			valueCtx.expectedTypeRoot = valueNode
			statements = append(statements, &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.SelectorExpr{
					X:   &ast.Ident{Name: receiverName},
					Sel: &ast.Ident{Name: field.Name},
				}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{ParseExpr(valueNode, source, valueCtx)},
			})
		}
	}
	if len(statements) == 0 {
		return nil
	}

	declaration := &ast.FuncDecl{
		Name: &ast.Ident{Name: methodName},
		Recv: &ast.FieldList{List: []*ast.Field{{
			Names: []*ast.Ident{{Name: receiverName}},
			Type:  &ast.StarExpr{X: &ast.Ident{Name: structName}},
		}}},
		Type: &ast.FuncType{Params: &ast.FieldList{}},
		Body: &ast.BlockStmt{List: statements},
	}
	return buildExecutionAwareFuncDecls(
		declaration,
		methodName+executionMethodSuffix,
		executionName,
		initializerCtx,
	)
}

func localConstructorOptions(
	structName string,
	captures []localConstructorCapture,
	fieldInitializerMethodName string,
) constructorLoweringOptions {
	receiverName := ShortName(structName)
	options := constructorLoweringOptions{
		fieldInitializerMethodName: fieldInitializerMethodName,
		skipClassInitialization:    true,
	}
	for _, binding := range captures {
		options.leadingParams = append(options.leadingParams, &ast.Field{
			Names: []*ast.Ident{{Name: binding.parameterName}},
			Type:  binding.capture.goType,
		})
		options.leadingThisArgs = append(options.leadingThisArgs, &ast.Ident{Name: binding.parameterName})
		options.terminalPreSuper = append(options.terminalPreSuper, &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.SelectorExpr{
				X:   &ast.Ident{Name: receiverName},
				Sel: &ast.Ident{Name: binding.capture.name},
			}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.Ident{Name: binding.parameterName}},
		})
	}
	return options
}

// hoistLocalClass lifts a class declared inside a method body to file scope.
// Referenced enclosing locals are captured as struct fields; the class's own
// instance fields and methods are emitted on the synthesized struct. The local
// class name is registered so subsequent `new Name(...)` in the same body builds
// the hoisted struct with its captured locals.
func hoistLocalClass(node *sitter.Node, source []byte, ctx Ctx) {
	if ctx.hoistedDecls == nil || ctx.localClasses == nil {
		return
	}
	nameNode := node.ChildByFieldName("name")
	classBody := node.ChildByFieldName("body")
	if nameNode == nil || classBody == nil {
		return
	}
	javaName := nameNode.Content(source)

	prefix := ""
	if ctx.className != "" {
		prefix = ctx.className
	}
	structName := fmt.Sprintf("%sLocal%s%d", prefix, javaName, ctx.nextAnonClassIndex())

	declaredFieldDefs, declaredAstFields := anonymousClassDeclaredFields(classBody, source, ctx)
	declaredFieldNames := map[string]struct{}{}
	for _, def := range declaredFieldDefs {
		declaredFieldNames[def.Name] = struct{}{}
	}

	captured := collectCapturedLocals(classBody, source, ctx)
	// Drop captures that collide with a declared field name to avoid duplicates.
	deduped := captured[:0]
	for _, cap := range captured {
		if _, clash := declaredFieldNames[cap.name]; clash {
			continue
		}
		deduped = append(deduped, cap)
	}
	captured = deduped

	bodyMethods := anonymousClassMethods(classBody)
	constructorNodes := localClassConstructors(classBody)
	captureBindings := localConstructorCaptureBindings(classBody, captured, source)
	syntheticScope := synthLocalClassScope(
		node,
		classBody,
		javaName,
		structName,
		declaredFieldDefs,
		captured,
		captureBindings,
		bodyMethods,
		constructorNodes,
		source,
	)
	fieldInitializerMethod := ""
	if syntheticScope.HasInstanceFieldInitializers {
		fieldInitializerMethod = localClassFieldInitializerMethodName(syntheticScope)
	}
	// Register before rendering method bodies: a static member may use a
	// class-qualified call to a sibling (`LocalType.helper()`), an initializer may
	// recursively allocate the same local type, and later code in the enclosing
	// method uses the same entry for `LocalType.method()`.
	ctx.localClasses[javaName] = &localClassInfo{
		structName:                 structName,
		captured:                   captured,
		scope:                      syntheticScope,
		fieldInitializerMethodName: fieldInitializerMethod,
	}

	// Mirror ordinary class layout: an immediate user superclass is embedded so
	// inherited fields/methods promote naturally, followed by implemented
	// interfaces, then this class's own fields and captured locals.
	fields := &ast.FieldList{}
	if superclassNode := node.ChildByFieldName("superclass"); superclassNode != nil {
		for _, superType := range collectTypeNodes(superclassNode) {
			javaType := superType.Content(source)
			base, _ := parseJavaTypeString(javaType)
			builtin := stripJavaQualifier(base)
			switch {
			case isBuiltinExceptionType(builtin) && resolveClassScopeByQualifiedName(ctx, builtin) == nil:
				fields.List = append(fields.List, &ast.Field{Type: stdjavaQualifiedExpr(builtin, ctx)})
			case builtin == "Thread" && resolveClassScopeByQualifiedName(ctx, builtin) == nil:
				fields.List = append(fields.List, &ast.Field{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("Thread", ctx)}})
			default:
				fields.List = append(fields.List, &ast.Field{Type: javaTypeStringToGoTypeExpr(
					javaType,
					inScopeTypeParameters(ctx),
					ctx,
				)})
			}
		}
	}
	if interfacesNode := node.ChildByFieldName("interfaces"); interfacesNode != nil {
		for _, interfaceType := range collectTypeNodes(interfacesNode) {
			if embed := implementedInterfaceTypeExpr(interfaceType.Content(source), inScopeTypeParameters(ctx), ctx); embed != nil {
				fields.List = append(fields.List, &ast.Field{Type: embed})
			}
		}
	}
	fields.List = append(fields.List, declaredAstFields...)
	for _, capture := range captured {
		fields.List = append(fields.List, &ast.Field{
			Names: []*ast.Ident{{Name: capture.name}},
			Type:  capture.goType,
		})
	}
	ctx.addHoistedDecl(GenStruct(structName, fields))

	localCtx := ctx.Clone()
	localCtx.currentClass = syntheticScope
	localCtx.className = structName
	localCtx.localScope = nil

	if fieldInitializerMethod != "" {
		initializerDecls := buildLocalClassFieldInitializerMethod(
			structName,
			fieldInitializerMethod,
			classBody,
			syntheticScope,
			source,
			ctx,
		)
		if len(initializerDecls) > 0 {
			for _, initializerDecl := range initializerDecls {
				ctx.addHoistedDecl(initializerDecl)
			}
		} else {
			ctx.localClasses[javaName].fieldInitializerMethodName = ""
			fieldInitializerMethod = ""
			syntheticScope.HasInstanceFieldInitializers = false
		}
	}
	if selfSetter := generateClassSelfSetter(localCtx); selfSetter != nil {
		ctx.addHoistedDecl(selfSetter)
	}

	constructorOptions := localConstructorOptions(structName, captureBindings, fieldInitializerMethod)
	for _, constructorNode := range constructorNodes {
		constructorOptions.definition = nil
		for _, definition := range syntheticScope.Methods {
			if definition != nil && definition.Constructor && definition.DeclarationNode == constructorNode {
				constructorOptions.definition = definition
				break
			}
		}
		for _, constructorDecl := range buildSourceConstructorDecls(
			constructorNode,
			source,
			localCtx,
			constructorOptions,
		) {
			ctx.addHoistedDecl(constructorDecl)
		}
	}
	if len(constructorNodes) == 0 {
		for _, constructorDecl := range buildDefaultConstructorDeclsWithOptions(localCtx, constructorOptions) {
			ctx.addHoistedDecl(constructorDecl)
		}
	}
	for _, methodNode := range bodyMethods {
		for _, methodDecl := range buildAnonymousStructMethod(structName, methodNode, syntheticScope, source, ctx) {
			ctx.addHoistedDecl(methodDecl)
		}
	}
}

func executionAwareSAMFuncType(
	node *sitter.Node,
	method *symbol.Definition,
	bindings map[string]string,
	source []byte,
	ctx Ctx,
) (*ast.FuncType, string) {
	if method == nil {
		return nil, ""
	}
	params := &ast.FieldList{}
	for index, parameter := range method.Parameters {
		javaType := substituteJavaTypeParams(parameter.OriginalType, bindings)
		params.List = append(params.List, &ast.Field{
			Names: []*ast.Ident{{Name: parameter.Name}},
			Type:  executionParameterTypeExpr(method, index, javaType, inScopeTypeParameters(ctx), ctx),
		})
	}
	executionName := executionParameterName(node, source, ctx)
	if executionName == "" {
		executionName = executionNameForParams(params)
	}
	params.List = append([]*ast.Field{executionParameterField(executionName, ctx)}, params.List...)
	var results *ast.FieldList
	if strings.TrimSpace(method.OriginalType) != "" && strings.TrimSpace(method.OriginalType) != "void" {
		javaType := substituteJavaTypeParams(method.OriginalType, bindings)
		results = &ast.FieldList{List: []*ast.Field{{
			Type: javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx),
		}}}
	}
	return &ast.FuncType{Params: params, Results: results}, executionName
}

func executionAwareMethodReferenceForwarder(
	boundReceiver ast.Expr,
	resolution *methodResolution,
	target *invocationTargetInfo,
	functionType *ast.FuncType,
	executionName string,
	unbound bool,
	node *sitter.Node,
	source []byte,
	ctx Ctx,
) ast.Expr {
	if resolution == nil || resolution.def == nil || resolution.owner == nil || target == nil || target.classScope == nil || functionType == nil || functionType.Params == nil {
		return nil
	}
	targetScope := target.classScope
	params := cloneFieldList(functionType.Params)
	args := methodCallArgs(params)
	if len(args) == 0 {
		return nil
	}
	execution := args[0]
	javaArgs := args[1:]
	usedNames := affineLoopUsedNames(node, source, ctx)
	originalBoundReceiver := boundReceiver
	boundReceiverName := ""
	receiver := boundReceiver
	if !unbound {
		boundReceiverName = synchronizedUniqueLocalName("__java2goMethodReferenceReceiver", usedNames)
		receiver = &ast.Ident{Name: boundReceiverName}
	}
	if unbound {
		if len(javaArgs) == 0 {
			return nil
		}
		receiver = javaArgs[0]
		javaArgs = javaArgs[1:]
	}
	if receiver == nil {
		return nil
	}
	if executionParameterIsVariadic(resolution.def, len(resolution.def.Parameters)-1) && len(javaArgs) > 0 {
		// A non-varargs SAM parameter such as int[] now uses PrimitiveArray,
		// while the target's Go variadic ABI necessarily receives []int32.
		// Unwrap only when the forwarding closure's final parameter has that
		// descriptor-bearing wrapper shape; a varargs SAM already has []T here.
		if last := functionType.Params.List[len(functionType.Params.List)-1]; last != nil && primitiveArrayWrapperType(last.Type) {
			javaArgs[len(javaArgs)-1] = stdjavaCall(ctx, "PrimitiveArrayElements", javaArgs[len(javaArgs)-1])
		}
	}

	executionReceiverName := synchronizedUniqueLocalName("__java2goExecutionReceiver", usedNames)
	hasExecutionReceiverName := synchronizedUniqueLocalName("__java2goHasExecutionReceiver", usedNames)
	body := []ast.Stmt{}
	callReceiver := receiver
	if targetScope.IsInterface || targetScope.IsAbstract {
		companionType := executionCompanionTypeExpr(target, resolution, ctx)
		if companionType == nil {
			return nil
		}
		body = append(body, &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: executionReceiverName}, &ast.Ident{Name: hasExecutionReceiverName}},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{&ast.TypeAssertExpr{
				X: &ast.CallExpr{
					Fun:  &ast.InterfaceType{Methods: &ast.FieldList{}},
					Args: []ast.Expr{receiver},
				},
				Type: companionType,
			}},
		})
		hiddenCall := &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: executionReceiverName},
				Sel: &ast.Ident{Name: executionImplementationName(resolution.def, resolution.owner)},
			},
			Args: append([]ast.Expr{execution}, javaArgs...),
		}
		markVariadicForwardCall(hiddenCall, resolution.def)
		hiddenBody := []ast.Stmt{invocationClosureCallStatement(hiddenCall, functionType.Results)}
		if functionType.Results == nil || len(functionType.Results.List) == 0 {
			hiddenBody = append(hiddenBody, &ast.ReturnStmt{})
		}
		body = append(body, &ast.IfStmt{
			Cond: &ast.Ident{Name: hasExecutionReceiverName},
			Body: &ast.BlockStmt{List: hiddenBody},
		})
		publicCall := &ast.CallExpr{
			Fun:  &ast.SelectorExpr{X: receiver, Sel: &ast.Ident{Name: resolution.def.Name}},
			Args: javaArgs,
		}
		markVariadicForwardCall(publicCall, resolution.def)
		body = append(body, invocationClosureCallStatement(publicCall, functionType.Results))
	} else {
		if classNeedsVirtualDispatch(resolution.owner, ctx) {
			callReceiver = &ast.SelectorExpr{X: receiver, Sel: &ast.Ident{Name: classDispatchFieldName(resolution.owner)}}
		}
		call := &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   callReceiver,
				Sel: &ast.Ident{Name: executionImplementationName(resolution.def, resolution.owner)},
			},
			Args: append([]ast.Expr{execution}, javaArgs...),
		}
		markVariadicForwardCall(call, resolution.def)
		body = append(body, invocationClosureCallStatement(call, functionType.Results))
	}

	innerType := &ast.FuncType{Params: params, Results: cloneFieldList(functionType.Results)}
	inner := &ast.FuncLit{Type: innerType, Body: &ast.BlockStmt{List: body}}
	if unbound {
		return inner
	}

	receiverJavaType, ok := inferExprJavaType(node.NamedChild(0), ctx, source)
	if !ok {
		return inner
	}
	receiverType := javaTypeStringToGoTypeExpr(receiverJavaType, inScopeTypeParameters(ctx), ctx)
	if targetScope.IsAbstract {
		receiverType = abstractClassToInterface(receiverType, receiverJavaType, ctx)
	}
	// Rebuild the inner closure against the staged receiver so a bound primary is
	// evaluated exactly once when the Java method reference is created.
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: boundReceiverName}}, Type: receiverType}}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: innerType}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{inner}}}},
	}, Args: []ast.Expr{originalBoundReceiver}}
}

func primitiveArrayWrapperType(expr ast.Expr) bool {
	pointer, ok := expr.(*ast.StarExpr)
	if !ok || pointer == nil {
		return false
	}
	indexed, ok := pointer.X.(*ast.IndexExpr)
	if !ok || indexed == nil {
		return false
	}
	switch base := indexed.X.(type) {
	case *ast.SelectorExpr:
		return base.Sel != nil && base.Sel.Name == "PrimitiveArray"
	case *ast.Ident:
		return base.Name == "PrimitiveArray"
	default:
		return false
	}
}

func wrapLambdaWithFunctionalInterfaceAdapter(lambdaExpr ast.Expr, expectedType string, executionAware bool, ctx Ctx) ast.Expr {
	method, _ := resolveFunctionalInterfaceMethod(ctx, expectedType)
	if method == nil {
		return nil
	}

	baseType, typeArgs := parseJavaTypeString(expectedType)
	interfaceScope := resolveClassScopeByQualifiedName(ctx, baseType)
	if interfaceScope == nil || interfaceScope.Class == nil || interfaceScope.Class.Name == "" {
		return nil
	}

	constructorName := "New" + interfaceScope.Class.Name + "FuncAdapter"
	if executionAware {
		constructorName += executionMethodSuffix
	}
	constructor := qualifiedNameExpr(constructorName, findJavaPackageForClassScope(interfaceScope), ctx)
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

func isBoxedPrimitiveJavaType(javaType string) bool {
	base, _ := parseJavaTypeString(javaType)
	switch stripJavaQualifier(base) {
	case "Boolean", "Byte", "Short", "Character", "Integer", "Long", "Float", "Double":
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

// instanceofSubjectExpr preserves Java's runtime class identity when a subclass
// has been coerced to an embedded concrete superclass pointer. Such pointers
// retain their most-derived receiver in the superclass dispatch slot, so a
// downcast-style instanceof must inspect that receiver rather than the embedded
// pointer itself. The small IIFE evaluates the Java operand exactly once and
// keeps null instanceof T false without dereferencing a nil pointer.
func instanceofSubjectExpr(left *sitter.Node, targetJavaType string, source []byte, ctx Ctx) ast.Expr {
	leftExpr := ParseExpr(left, source, ctx)
	staticJavaType, ok := inferExprJavaType(left, ctx, source)
	if !ok {
		return leftExpr
	}

	staticBase, _ := parseJavaTypeString(staticJavaType)
	targetBase, _ := parseJavaTypeString(targetJavaType)
	staticScope := resolveClassScopeByQualifiedName(ctx, staticBase)
	targetScope := resolveClassScopeByQualifiedName(ctx, targetBase)
	if staticScope == nil || targetScope == nil || staticScope == targetScope ||
		staticScope.IsAbstract || staticScope.IsInterface || staticScope.IsEnum || targetScope.IsInterface ||
		!classNeedsVirtualDispatch(staticScope, ctx) || !javaReferenceTypeAssignable(targetScope, staticScope, ctx) {
		return leftExpr
	}

	valueName := "__java2goInstanceofValue"
	valueExpr := &ast.Ident{Name: valueName}
	dispatchExpr := &ast.SelectorExpr{
		X:   valueExpr,
		Sel: &ast.Ident{Name: classDispatchFieldName(staticScope)},
	}
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "any"}}}}},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{valueExpr},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{leftExpr},
			},
			&ast.IfStmt{
				Cond: &ast.BinaryExpr{X: valueExpr, Op: token.EQL, Y: &ast.Ident{Name: "nil"}},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "nil"}}}}},
			},
			&ast.IfStmt{
				Cond: &ast.BinaryExpr{X: dispatchExpr, Op: token.NEQ, Y: &ast.Ident{Name: "nil"}},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{dispatchExpr}}}},
			},
			&ast.ReturnStmt{Results: []ast.Expr{valueExpr}},
		}},
	}}
}

// defaultConstructorName returns the synthesized default-constructor name for a
// generated class struct, matching the casing used when one is emitted: a public
// (capitalized) class gets `New<Name>`, a package-private one gets `new<Name>`.
func defaultConstructorName(className string) string {
	if className == "" {
		return "New"
	}
	// A class is exported iff its generated struct name is already capitalized.
	exported := className == symbol.Uppercase(className)
	prefix := "new"
	if exported {
		prefix = "New"
	}
	return prefix + symbol.Uppercase(className)
}

// maskedShiftAmount returns the Go expression for a Java shift count. Java masks
// the count to the low 5 bits when the left operand is int (or a narrower type
// promoted to int) and to the low 6 bits when it is long, before shifting, while
// Go applies the full count. For a constant decimal count we fold the mask at
// transpile time (e.g. int `1 << 32` becomes `1 << 0`, but long `1L << 32` stays
// `1L << 32`). Variable counts are left as-is: masking them would change the type
// of the shift and the common case (counts already in range) is unaffected.
func maskedShiftAmount(leftNode, rightNode *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	if rightNode != nil && rightNode.Type() == "decimal_integer_literal" {
		literal := rightNode.Content(source)
		// Drop a trailing long suffix if present; the count itself is always int.
		trimmed := strings.TrimRight(literal, "lL")
		if value, ok := parseDecimalUint(trimmed); ok {
			var mask uint64 = 31
			if shiftOperandIsLong(leftNode, source, ctx) {
				mask = 63
			}
			masked := value & mask
			if masked != value {
				return &ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", masked)}
			}
		}
	}
	return ParseExpr(rightNode, source, ctx)
}

// shiftOperandIsLong reports whether a shift's left operand has Java type long
// (so the shift count masks to 6 bits rather than 5). It recognizes long
// literals (1L), casts to long, and operands whose inferred type is long/Long.
// Anything else is treated as int, matching Java's promotion of narrower types.
func shiftOperandIsLong(node *sitter.Node, source []byte, ctx Ctx) bool {
	if node == nil {
		return false
	}
	switch node.Type() {
	case "decimal_integer_literal", "hex_integer_literal", "octal_integer_literal", "binary_integer_literal":
		lit := node.Content(source)
		return strings.HasSuffix(lit, "L") || strings.HasSuffix(lit, "l")
	case "parenthesized_expression":
		if node.NamedChildCount() > 0 {
			return shiftOperandIsLong(node.NamedChild(0), source, ctx)
		}
	case "cast_expression":
		if typeNode := node.NamedChild(0); typeNode != nil {
			return isLongJavaType(typeNode.Content(source))
		}
	case "unary_expression":
		// Sign/complement preserve the operand's type.
		if count := int(node.NamedChildCount()); count > 0 {
			return shiftOperandIsLong(node.NamedChild(count-1), source, ctx)
		}
	case "binary_expression":
		// A binary op is long if either side is long (Java numeric promotion).
		if node.NamedChildCount() >= 2 {
			return shiftOperandIsLong(node.Child(0), source, ctx) || shiftOperandIsLong(node.Child(2), source, ctx)
		}
	}
	if javaType, ok := inferExprJavaType(node, ctx, source); ok {
		return isLongJavaType(javaType)
	}
	return false
}

// isLongJavaType reports whether a Java type string denotes the 64-bit long type.
func isLongJavaType(javaType string) bool {
	base, _ := parseJavaTypeString(javaType)
	switch stripJavaQualifier(base) {
	case "long", "Long":
		return true
	}
	return false
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

// substituteJavaTypeParameters replaces class type parameters in a Java type
// string with the receiver's concrete type arguments. It handles nested generic
// arguments and arrays, e.g. Map<String, T[]> with T=Event becomes
// Map<String, Event[]>.
func substituteJavaTypeParameters(typeStr string, bindings map[string]string) string {
	typeStr = strings.TrimSpace(typeStr)
	if typeStr == "" || len(bindings) == 0 {
		return typeStr
	}

	arraySuffix := ""
	for strings.HasSuffix(typeStr, "[]") {
		arraySuffix += "[]"
		typeStr = strings.TrimSpace(typeStr[:len(typeStr)-2])
	}

	if replacement, ok := bindings[typeStr]; ok {
		return replacement + arraySuffix
	}

	base, args := parseJavaTypeString(typeStr)
	if len(args) == 0 {
		return typeStr + arraySuffix
	}
	for i, arg := range args {
		args[i] = substituteJavaTypeParameters(arg, bindings)
	}
	return base + "<" + strings.Join(args, ", ") + ">" + arraySuffix
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
	appendUnique := func(names ...string) {
		for _, name := range names {
			found := false
			for _, existing := range params {
				if existing == name {
					found = true
					break
				}
			}
			if !found {
				params = append(params, name)
			}
		}
	}

	// Java class type parameters are unavailable inside static methods. Any
	// parameters needed to model a raw generic signature are added explicitly as
	// synthetic function parameters below.
	if ctx.currentClass != nil && (ctx.localScope == nil || !ctx.localScope.IsStatic) {
		appendUnique(ctx.currentClass.TypeParameterNames()...)
	}
	if ctx.localScope != nil {
		appendUnique(ctx.localScope.TypeParameterNames()...)
	}
	appendUnique(symbol.TypeParamNames(ctx.syntheticTypeParameters)...)
	return params
}

// javaNumericPromotionType returns the primitive type produced by Java binary
// numeric promotion. Java first unboxes wrapper operands, then chooses double,
// float, long, or int in that order; byte, short, and char never remain narrow
// in a binary arithmetic expression.
func canonicalJavaNumericType(javaType string) (string, bool) {
	base, _ := parseJavaTypeString(javaType)
	switch stripJavaQualifier(base) {
	case "byte", "Byte":
		return "byte", true
	case "short", "Short":
		return "short", true
	case "char", "Character":
		return "char", true
	case "int", "Integer":
		return "int", true
	case "long", "Long":
		return "long", true
	case "float", "Float":
		return "float", true
	case "double", "Double":
		return "double", true
	default:
		return "", false
	}
}

func javaNumericPromotionType(a, b string) (string, bool) {
	left, leftOK := canonicalJavaNumericType(a)
	right, rightOK := canonicalJavaNumericType(b)
	if !leftOK || !rightOK {
		return "", false
	}
	if left == "double" || right == "double" {
		return "double", true
	}
	if left == "float" || right == "float" {
		return "float", true
	}
	if left == "long" || right == "long" {
		return "long", true
	}
	return "int", true
}

// promoteJavaBinaryNumericOperands inserts the explicit conversions Go needs
// to model Java's implicit binary numeric promotion. For example, Java permits
// intValue + 1L and intValue < longValue; their Go equivalents must convert the
// int32 operand to int64. The conversion is driven solely by inferred Java
// types and applies to arithmetic, numeric comparison/equality, and bitwise
// operators. String concatenation and boolean/reference operators are excluded.
func promoteJavaBinaryNumericOperands(
	operator string,
	leftNode, rightNode *sitter.Node,
	leftExpr, rightExpr ast.Expr,
	source []byte,
	ctx Ctx,
) (ast.Expr, ast.Expr) {
	switch operator {
	case "+", "-", "*", "/", "%", "&", "|", "^", "<", "<=", ">", ">=", "==", "!=":
		// eligible below
	default:
		return leftExpr, rightExpr
	}

	leftType, leftOK := inferExprJavaType(leftNode, ctx, source)
	rightType, rightOK := inferExprJavaType(rightNode, ctx, source)
	if !leftOK || !rightOK {
		return leftExpr, rightExpr
	}
	targetType, ok := javaNumericPromotionType(leftType, rightType)
	if !ok {
		return leftExpr, rightExpr
	}

	return convertJavaNumericOperand(leftExpr, leftType, targetType),
		convertJavaNumericOperand(rightExpr, rightType, targetType)
}

func convertJavaNumericOperand(expr ast.Expr, sourceType, targetType string) ast.Expr {
	normalizedSource, ok := canonicalJavaNumericType(sourceType)
	if !ok || normalizedSource == targetType {
		return expr
	}

	var conversion string
	switch targetType {
	case "byte":
		conversion = "int8"
	case "short":
		conversion = "int16"
	case "char":
		conversion = "rune"
	case "int":
		conversion = "int32"
	case "long":
		conversion = "int64"
	case "float":
		conversion = "float32"
	case "double":
		conversion = "float64"
	default:
		return expr
	}
	return &ast.CallExpr{Fun: &ast.Ident{Name: conversion}, Args: []ast.Expr{expr}}
}

// promoteJavaUnaryNumericOperand applies JLS unary numeric promotion before an
// operation is evaluated. Go otherwise keeps -byteValue and byteValue << n in
// int8, so values such as -Byte.MIN_VALUE or 64 << 2 overflow before a later
// conversion can repair them. Java widens byte, short, and char to int first.
func promoteJavaUnaryNumericOperand(node *sitter.Node, expr ast.Expr, ctx Ctx, source []byte) ast.Expr {
	javaType, ok := inferExprJavaType(node, ctx, source)
	if !ok {
		return expr
	}
	canonical, numeric := canonicalJavaNumericType(javaType)
	if !numeric {
		return expr
	}
	switch canonical {
	case "byte", "short", "char":
		return convertJavaNumericOperand(expr, javaType, "int")
	default:
		return expr
	}
}

// goIndexExpr parses a Java array/slice index expression and coerces it to Go's
// required `int` index type. Java int indices are emitted as int32 now that int
// locals are pinned (K1), so a variable or compound index must be wrapped in
// int(...). Plain integer-literal indices are untyped constants and are left
// uncast to avoid noise like a[int(0)].
func goIndexExpr(node *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	expr := ParseExpr(node, source, ctx)
	switch node.Type() {
	case "decimal_integer_literal", "hex_integer_literal", "octal_integer_literal", "binary_integer_literal":
		return expr
	}
	return &ast.CallExpr{Fun: &ast.Ident{Name: "int"}, Args: []ast.Expr{expr}}
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

	originalType := typeStr
	// Arrays like Foo[][].
	arrayDims := 0
	for strings.HasSuffix(typeStr, "[]") {
		arrayDims++
		typeStr = strings.TrimSpace(typeStr[:len(typeStr)-2])
	}
	if arrayDims > 0 {
		if primitive, ok := primitiveArrayTypeExpr(originalType, ctx); ok {
			return primitive
		}
		if _, _, reified := reifiedSourceReferenceArrayComponent(originalType, ctx); reified {
			return reifiedReferenceArrayTypeExpr(ctx)
		}
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
	resolvedScope := resolveClassScopeByQualifiedName(ctx, base)

	// java.util collection types map to the stdjava runtime types, provided the
	// name is not shadowed by a user-defined class.
	if resolvedScope == nil {
		if collExpr := collectionTypeExpr(baseName, typeArgs, typeParams, ctx); collExpr != nil {
			expr := collExpr
			for i := 0; i < arrayDims; i++ {
				expr = &ast.ArrayType{Elt: expr}
			}
			return expr
		}
	}

	if resolvedScope != nil && resolvedScope.Class != nil {
		isInterface = resolvedScope.IsInterface
		if resolvedScope.Class.Name != "" {
			baseName = resolvedScope.Class.Name
		}
		targetPkg = resolveJavaPackageForType(ctx, base, resolvedScope)

		// A non-static nested Java class implicitly carries its enclosing class's
		// type parameters. References such as `Node next` inside `Outer<T>` are
		// therefore `Node<T>` even though Java does not repeat the argument. Go
		// requires every generic type to be instantiated. The same fallback maps a
		// true Java raw type to its erased upper bound (`Object` when unbounded),
		// which is the closest valid Go representation.
		if len(typeArgs) == 0 && len(resolvedScope.TypeParameters) > 0 {
			available := make(map[string]struct{}, len(typeParams))
			for _, typeParam := range typeParams {
				available[typeParam] = struct{}{}
			}
			typeArgs = make([]string, 0, len(resolvedScope.TypeParameters))
			for _, typeParam := range resolvedScope.TypeParameters {
				if _, ok := available[typeParam.Name]; ok {
					typeArgs = append(typeArgs, typeParam.Name)
					continue
				}
				if len(typeParam.Bounds) > 0 && strings.TrimSpace(typeParam.Bounds[0].Original) != "" {
					typeArgs = append(typeArgs, typeParam.Bounds[0].Original)
					continue
				}
				typeArgs = append(typeArgs, "Object")
			}
		}
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
		case "Object":
			return &ast.Ident{Name: "any"}, true
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
			return &ast.Ident{Name: "int8"}, true
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
			return &ast.Ident{Name: "int8"}, true
		case "float":
			return &ast.Ident{Name: "float32"}, true
		case "double":
			return &ast.Ident{Name: "float64"}, true
		}
		return nil, false
	}

	var expr ast.Expr
	if resolvedScope == nil && baseName == "Throwable" {
		// Throwable is the runtime's common exception interface. Keep this narrow
		// and after source resolution so a user-defined class with the same simple
		// name retains its generated type.
		expr = stdjavaQualifiedExpr("Throwable", ctx)
	} else if prim, ok := primitive(baseName); ok {
		expr = prim
	} else if rt, ok := stdjavaRuntimeTypeExpr(baseName, typeArgs, typeParams, ctx); ok {
		// java.util.concurrent / java.lang.Thread types backed by the stdjava
		// runtime (AtomicInteger, Thread, ConcurrentHashMap, ...).
		expr = rt
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
	if rewritten, ok := ctx.rawGenericParameterTypes[name]; ok && strings.TrimSpace(rewritten) != "" {
		return rewritten, true
	}
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

// qualifyJavaTypeInDeclaringContext preserves where user-defined types in a
// method signature were declared. A chained call is resolved while converting
// the caller's file, but an unqualified return such as Priority belongs to the
// callee's package and may not be imported by the caller. Returning a qualified
// Java type keeps that provenance available to resolveInvocationTarget.
func qualifyJavaTypeInDeclaringContext(typeStr string, owner *symbol.ClassScope) string {
	typeStr = strings.TrimSpace(typeStr)
	ownerFile := findFileScopeForClassScope(owner)
	if typeStr == "" || ownerFile == nil {
		return typeStr
	}

	arraySuffix := ""
	for strings.HasSuffix(typeStr, "[]") {
		arraySuffix += "[]"
		typeStr = strings.TrimSpace(typeStr[:len(typeStr)-2])
	}

	if strings.HasPrefix(typeStr, "?") {
		rest := strings.TrimSpace(strings.TrimPrefix(typeStr, "?"))
		switch {
		case rest == "":
			return "?" + arraySuffix
		case strings.HasPrefix(rest, "extends"):
			bound := strings.TrimSpace(strings.TrimPrefix(rest, "extends"))
			return "? extends " + qualifyJavaTypeInDeclaringContext(bound, owner) + arraySuffix
		case strings.HasPrefix(rest, "super"):
			bound := strings.TrimSpace(strings.TrimPrefix(rest, "super"))
			return "? super " + qualifyJavaTypeInDeclaringContext(bound, owner) + arraySuffix
		default:
			return typeStr + arraySuffix
		}
	}

	base, args := parseJavaTypeString(typeStr)
	qualifiedArgs := make([]string, len(args))
	for index, arg := range args {
		qualifiedArgs[index] = qualifyJavaTypeInDeclaringContext(arg, owner)
	}

	qualifiedBase := base
	declCtx := Ctx{currentFile: ownerFile}
	if scope := resolveClassScopeByQualifiedName(declCtx, base); scope != nil {
		if javaPkg := resolveJavaPackageForType(declCtx, base, scope); javaPkg != "" && !strings.Contains(base, ".") {
			qualifiedBase = javaPkg + "." + base
		}
	}

	if len(qualifiedArgs) > 0 {
		return qualifiedBase + "<" + strings.Join(qualifiedArgs, ", ") + ">" + arraySuffix
	}
	return qualifiedBase + arraySuffix
}

// inferUserMethodReturnType returns the declared Java return type of a
// user-defined method invocation, resolving the method from the receiver's class
// (for X.m()) or the current class (for an unqualified m()). Returns false when
// the method is unknown, has a void/empty return type, or is a builtin.
func inferUserMethodReturnType(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return "", false
	}
	methodName := nameNode.Content(source)

	argListNode := node.ChildByFieldName("arguments")

	var scope *symbol.ClassScope
	allowInstance := true
	allowStatic := true
	var receiverTypeArgs []string
	if objectNode := node.ChildByFieldName("object"); objectNode != nil {
		if classScope := resolveClassScopeByIdentifier(ctx, source, objectNode); classScope != nil {
			scope = classScope
			allowInstance = false
		} else {
			if receiverType, ok := inferExprJavaType(objectNode, ctx, source); ok {
				_, receiverTypeArgs = parseJavaTypeString(receiverType)
			}
			if target := resolveInvocationTarget(objectNode, ctx, source); target != nil {
				scope = target.classScope
			}
		}
	} else {
		scope = ctx.currentClass
		allowInstance = ctx.localScope == nil || !ctx.localScope.IsStatic
	}
	if scope == nil {
		return "", false
	}

	resolution := findBestMethodInHierarchy(scope, methodName, argListNode, allowInstance, allowStatic, ctx, source)
	if resolution == nil || resolution.def == nil {
		return "", false
	}
	rt := strings.TrimSpace(resolution.def.OriginalType)
	if rt == "" || rt == "void" {
		return "", false
	}
	// A bare type-parameter return (e.g. R from Mapper<T,R>) is not a concrete
	// type, so don't report it — the caller's declared type is more useful and
	// would otherwise be overwritten with an unusable type variable.
	base, _ := parseJavaTypeString(rt)
	for _, tp := range resolution.def.TypeParameterNames() {
		if base == tp {
			return "", false
		}
	}
	if resolution.owner != nil {
		bindings := make(map[string]string)
		for index, tp := range resolution.owner.TypeParameterNames() {
			if resolution.owner == scope && index < len(receiverTypeArgs) {
				bindings[tp] = receiverTypeArgs[index]
			}
		}
		if substituted := substituteJavaTypeParameters(rt, bindings); substituted != rt {
			rt = substituted
			base, _ = parseJavaTypeString(rt)
		}
		for _, tp := range resolution.owner.TypeParameterNames() {
			if base == tp {
				return "", false
			}
		}
	}
	return qualifyJavaTypeInDeclaringContext(rt, resolution.owner), true
}

// inferStreamMapResultType infers the element type produced by a Stream.map(...)
// call from the mapper lambda's body, binding the lambda parameter to the
// receiver's element type. Returns the Java type of the result, or false when it
// cannot be determined (the caller then falls back to a bare Stream).
func inferStreamMapResultType(mapNode *sitter.Node, recvArgs []string, ctx Ctx, source []byte) (string, bool) {
	if len(recvArgs) != 1 {
		return "", false
	}
	argsNode := mapNode.ChildByFieldName("arguments")
	if argsNode == nil || argsNode.NamedChildCount() != 1 {
		return "", false
	}
	lambda := argsNode.NamedChild(0)
	if lambda == nil || lambda.Type() != "lambda_expression" {
		return "", false
	}
	paramNode := lambda.ChildByFieldName("parameters")
	bodyNode := lambda.ChildByFieldName("body")
	if paramNode == nil || bodyNode == nil || bodyNode.Type() == "block" {
		return "", false
	}
	paramName := paramNode.Content(source)
	if paramNode.NamedChildCount() == 1 {
		paramName = paramNode.NamedChild(0).Content(source)
	}
	lambdaCtx := ctx.Clone()
	lambdaCtx.localScope = &symbol.Definition{
		Parameters: []*symbol.Definition{
			{OriginalName: paramName, Name: paramName, OriginalType: recvArgs[0]},
		},
	}
	return inferExprJavaType(bodyNode, lambdaCtx, source)
}

func inferExprJavaType(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	switch node.Type() {
	case "identifier":
		return inferIdentifierJavaType(node.Content(source), ctx)
	case "assignment_expression":
		// Every Java assignment expression has the type of its left-hand side,
		// including compound assignments whose operation is promoted and then
		// implicitly narrowed back to the target type.
		if node.ChildCount() > 0 {
			return inferExprJavaType(node.Child(0), ctx, source)
		}
		return "", false
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
	case "cast_expression":
		// A cast's static type is its target type, e.g. `(char)(c+1)` is char.
		// This lets println wrap a char-casted value in string(...) so it prints
		// the character rather than its code point.
		if target := node.NamedChild(0); target != nil {
			return target.Content(source), true
		}
		return "", false
	case "array_creation_expression":
		javaType, dimensions := javaArrayCreationJavaType(node, source)
		return javaType, dimensions > 0
	case "array_access":
		// The element type of an array access is the array's type with one
		// dimension removed (e.g. Worker[] indexed -> Worker).
		arrayNode := node.ChildByFieldName("array")
		if arrayNode == nil {
			arrayNode = node.NamedChild(0)
		}
		if arrayNode == nil {
			return "", false
		}
		if arrayType, ok := inferExprJavaType(arrayNode, ctx, source); ok {
			trimmed := strings.TrimSpace(arrayType)
			if strings.HasSuffix(trimmed, "[]") {
				return strings.TrimSpace(trimmed[:len(trimmed)-2]), true
			}
		}
		return "", false
	case "string_literal":
		// A string literal is a java.lang.String, so chained calls on a literal
		// (e.g. "  x  ".trim()) resolve as String intrinsics.
		return "String", true
	case "decimal_integer_literal", "hex_integer_literal", "octal_integer_literal", "binary_integer_literal":
		// An integer literal is `long` if it carries an L suffix, otherwise `int`.
		// Used to infer `var x = 0` as int (-> int32) for K1 pinning.
		content := node.Content(source)
		if len(content) > 0 {
			if last := content[len(content)-1]; last == 'L' || last == 'l' {
				return "long", true
			}
		}
		return "int", true
	case "decimal_floating_point_literal", "hex_floating_point_literal":
		content := node.Content(source)
		if len(content) > 0 {
			switch content[len(content)-1] {
			case 'f', 'F':
				return "float", true
			}
		}
		return "double", true
	case "character_literal":
		return "char", true
	case "true", "false":
		return "boolean", true
	case "unary_expression":
		// A sign or bitwise complement has the unary-promoted operand type.
		// This is especially important for overloads invoked with negative numeric
		// literals, which tree-sitter represents as a unary expression around the
		// literal node.
		if count := int(node.NamedChildCount()); count > 0 {
			operandType, ok := inferExprJavaType(node.NamedChild(count-1), ctx, source)
			if !ok {
				return "", false
			}
			operator := node.Child(0).Content(source)
			if operator == "+" || operator == "-" || operator == "~" {
				if canonical, numeric := canonicalJavaNumericType(operandType); numeric {
					switch canonical {
					case "byte", "short", "char":
						return "int", true
					}
				}
			}
			return operandType, true
		}
	case "parenthesized_expression":
		if inner := node.NamedChild(0); inner != nil {
			return inferExprJavaType(inner, ctx, source)
		}
	case "ternary_expression":
		return inferTernaryResultJavaType(node, ctx, source)
	case "binary_expression":
		// Java's `+` is String concatenation when either operand is a String, so
		// the whole expression is a String (e.g. `var g = "a" + n;` makes g a
		// String). This lets intrinsics dispatch on a concatenation result.
		if op := node.Child(1); op != nil && op.Content(source) == "+" {
			if isStringLikeExprNode(node.Child(0), ctx, source) || isStringLikeExprNode(node.Child(2), ctx, source) {
				return "String", true
			}
		}
		// For an arithmetic or bitwise binary op, infer the type chosen by Java
		// binary numeric promotion. This lets both integer and floating-point mixed
		// expressions drive explicit Go conversions and `var` type pinning.
		if op := node.Child(1); op != nil {
			switch op.Content(source) {
			case "+", "-", "*", "/", "%", "&", "|", "^":
				lt, lok := inferExprJavaType(node.Child(0), ctx, source)
				rt, rok := inferExprJavaType(node.Child(2), ctx, source)
				if lok && rok {
					if combined, ok := javaNumericPromotionType(lt, rt); ok {
						return combined, true
					}
				}
			case "<<", ">>", ">>>":
				// Shift expressions have the unary-promoted type of their left side;
				// the right operand never widens the result.
				if leftType, ok := inferExprJavaType(node.Child(0), ctx, source); ok {
					if promoted, ok := javaNumericPromotionType(leftType, leftType); ok {
						return promoted, true
					}
				}
			}
		}
	case "method_invocation":
		// Chained String intrinsics: if the inner call is itself a String method
		// that returns a String, the result type is String so the outer call also
		// resolves (e.g. s.trim().toUpperCase()).
		if objectNode := node.ChildByFieldName("object"); objectNode != nil {
			nameNode := node.ChildByFieldName("name")
			if nameNode != nil {
				methodName := nameNode.Content(source)
				if methodName == "getCause" || methodName == "getSuppressed" {
					if receiverType, ok := inferExprJavaType(objectNode, ctx, source); ok && isExceptionJavaType(ctx, receiverType) {
						if methodName == "getCause" {
							return "Throwable", true
						}
						return "Throwable[]", true
					}
				}
				// Stream.of(...) returns a Stream, so a chained terminal/intermediate
				// op (Stream.of(..).count()) resolves.
				if objectNode.Type() == "identifier" && objectNode.Content(source) == "Stream" && methodName == "of" {
					return "Stream", true
				}
				// Character.toUpperCase/toLowerCase(char) return char.
				if objectNode.Type() == "identifier" && objectNode.Content(source) == "Character" {
					switch methodName {
					case "toUpperCase", "toLowerCase":
						return "char", true
					}
				}
				if recvType, ok := inferExprJavaType(objectNode, ctx, source); ok {
					base, recvArgs := parseJavaTypeString(recvType)
					recvBase := stripJavaQualifier(base)
					// List.get(int) returns the receiver's element type. Retain it so
					// a call chained directly on the result can resolve the generated
					// method name (plan.get(i).getId() -> plan.Get(i).GetId()).
					if methodName == "get" && len(recvArgs) == 1 && containsString(listTypeNames, recvBase) {
						return recvArgs[0], true
					}
					switch recvBase {
					case "String":
						switch {
						case methodName == "charAt":
							// String.charAt returns char.
							return "char", true
						case methodName == "length" || methodName == "indexOf" || methodName == "lastIndexOf" || methodName == "compareTo":
							return "int", true
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
					case "Optional":
						// Optional.map/filter return an Optional, so a chained call
						// (o.map(f).get()) resolves the outer receiver as an Optional.
						switch methodName {
						case "map", "filter":
							return "Optional", true
						}
					case "File":
						// File methods that return a String, so chained String calls
						// (f.getName().endsWith(...)) resolve.
						switch methodName {
						case "getName", "getPath", "getAbsolutePath":
							return "String", true
						}
					case "BufferedReader", "FileReader":
						if methodName == "readLine" {
							return "String", true
						}
					case "Scanner":
						switch methodName {
						case "next", "nextLine":
							return "String", true
						}
					}
					// Collection.stream() yields a Stream of the collection's element type
					// so a chained .filter/.map lambda is typed.
					if methodName == "stream" && len(recvArgs) == 1 &&
						(containsString(listTypeNames, recvBase) || containsString(setTypeNames, recvBase)) {
						return "Stream<" + recvArgs[0] + ">", true
					}
					// Stream intermediate ops that keep the element type preserve Stream<T>
					// for further chaining; map changes the element type so reports bare Stream.
					if containsString(streamTypeNames, recvBase) {
						switch methodName {
						case "filter", "sorted", "limit", "distinct", "skip", "peek":
							return recvType, true
						case "map":
							if r, ok := inferStreamMapResultType(node, recvArgs, ctx, source); ok {
								return "Stream<" + r + ">", true
							}
							return "Stream", true
						}
					}
				}
			}
		}
		// Fall back to the declared return type of a user-defined method, so a
		// chained call on its result (e.g. nums().stream()) can be typed.
		if rt, ok := inferUserMethodReturnType(node, ctx, source); ok {
			return rt, true
		}
	case "field_access":
		// Java arrays expose length as an int-valued pseudo-field. Preserve that
		// static type so compound assignments such as total += values.length can
		// apply Java numeric promotion instead of falling back to an unknown Object.
		obj := node.ChildByFieldName("object")
		fieldNode := node.ChildByFieldName("field")
		if obj != nil && fieldNode != nil && fieldNode.Content(source) == "length" && isArrayTypedExprNode(obj, ctx, source) {
			return "int", true
		}

		// A qualified enum constant access (Day.WED) has the enum's type, so that
		// chained calls like Day.WED.ordinal() resolve to the enum's methods.
		if obj != nil && obj.Type() == "identifier" {
			if scope := resolveClassScopeByIdentifier(ctx, source, obj); scope != nil && scope.IsEnum {
				return obj.Content(source), true
			}
		}

		// Resolve ordinary field accesses from the receiver's declared Java type.
		// Method calls commonly use an explicit `this.field` receiver; without this
		// branch both user-method resolution and stdlib intrinsic dispatch lose the
		// field's type and retain Java's lowercase method spelling in generated Go.
		if obj == nil || fieldNode == nil {
			return "", false
		}

		var owner *symbol.ClassScope
		switch obj.Type() {
		case "this":
			owner = ctx.currentClass
		case "super":
			owner = resolveSuperclassScope(ctx, ctx.currentClass)
		default:
			if ownerType, ok := inferExprJavaType(obj, ctx, source); ok {
				base, _ := parseJavaTypeString(ownerType)
				owner = resolveClassScopeByQualifiedName(ctx, base)
			}
			if owner == nil && obj.Type() == "identifier" {
				// Static field access through a class name.
				owner = resolveClassScopeByIdentifier(ctx, source, obj)
			}
		}

		if field := findFieldInHierarchy(owner, fieldNode.Content(source), ctx); field != nil && field.OriginalType != "" {
			return field.OriginalType, true
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
		// A value whose declared type is a type parameter gets its callable method
		// set from its Java upper bound. This lets `T extends Ranked` resolve calls
		// such as value.primaryScore() against Ranked's generated method names.
		if bound := firstResolvableTypeParameterBound(className, ctx); bound != "" {
			className, classTypeArgs = parseJavaTypeString(bound)
			classScope = resolveClassScopeByQualifiedName(ctx, className)
		}
	}
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

// firstResolvableTypeParameterBound returns the first declared upper bound that
// names a class or interface visible from ctx. Method type parameters shadow
// class type parameters, matching Java's scope rules.
func firstResolvableTypeParameterBound(name string, ctx Ctx) string {
	find := func(params []symbol.TypeParam) string {
		for _, param := range params {
			if param.Name != name {
				continue
			}
			for _, bound := range param.Bounds {
				base, _ := parseJavaTypeString(bound.Original)
				if resolveClassScopeByQualifiedName(ctx, base) != nil {
					return bound.Original
				}
			}
			return ""
		}
		return ""
	}

	if ctx.localScope != nil {
		if bound := find(ctx.localScope.TypeParameters); bound != "" {
			return bound
		}
	}
	if ctx.currentClass != nil {
		return find(ctx.currentClass.TypeParameters)
	}
	return ""
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
	for name, javaType := range genericArrayInvocationTypeBindings(def, invocationNode, ctx, source) {
		resolved[name] = javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx)
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

// genericArrayInvocationTypeBindings reconstructs the type arguments Java
// infers from bare T and T[] formals before generic reference arrays collapse
// to *ReferenceArray in Go. Every applicable argument contributes a lower
// bound: choosing the first one is unsound for calls such as f(Child[], Base)
// or f(Child[], Sibling[]), whose inferred T is their common Base.
//
// A concrete class bound still uses its erased class view because the current
// Go constraint is that concrete pointer type; scalar arguments are
// target-typed to the same view by genericArrayInvocationExpectedTypes below.
func genericArrayInvocationTypeBindings(def *symbol.Definition, invocationNode *sitter.Node, ctx Ctx, source []byte) map[string]string {
	bindings := map[string]string{}
	if def == nil || invocationNode == nil {
		return bindings
	}
	if typeArguments := invocationNode.ChildByFieldName("type_arguments"); typeArguments != nil {
		explicit := nodeutil.NamedChildrenOf(typeArguments)
		if len(explicit) == len(def.TypeParameters) {
			for index, parameter := range def.TypeParameters {
				bindings[parameter.Name] = explicit[index].Content(source)
			}
			return bindings
		}
	}
	argsNode := invocationNode.ChildByFieldName("arguments")
	if argsNode == nil {
		return bindings
	}
	argNodes := nodeutil.NamedChildrenOf(argsNode)
	lowerBounds := make(map[string][]string, len(def.TypeParameters))
	for index, parameter := range def.Parameters {
		if parameter == nil || index >= len(argNodes) {
			continue
		}
		formalBase, formalRank := javaArrayTypeParts(parameter.OriginalType)
		for _, typeParameter := range def.TypeParameters {
			if stripJavaQualifier(formalBase) != typeParameter.Name {
				continue
			}
			actualType, ok := inferExprJavaType(argNodes[index], ctx, source)
			if !ok {
				continue
			}
			actualBase, actualRank := javaArrayTypeParts(actualType)
			if actualRank < formalRank {
				continue
			}
			inferred := actualBase + strings.Repeat("[]", actualRank-formalRank)
			// A primitive expression supplied for scalar T participates through
			// Java boxing. Primitive arrays are references only while at least one
			// array dimension remains after matching the formal rank.
			if actualRank == 0 && formalRank == 0 {
				if boxed := ternaryBoxedJavaType(inferred); boxed != "" {
					inferred = boxed
				}
			}
			lowerBounds[typeParameter.Name] = append(lowerBounds[typeParameter.Name], inferred)
		}
	}

	for _, typeParameter := range def.TypeParameters {
		inferred := javaInferenceLeastUpperBound(lowerBounds[typeParameter.Name], ctx)
		if inferred == "" {
			continue
		}
		if len(typeParameter.Bounds) > 0 {
			bound := strings.TrimSpace(typeParameter.Bounds[0].Original)
			boundBase, _ := parseJavaTypeString(bound)
			if boundScope := resolveClassScopeByQualifiedName(ctx, boundBase); boundScope != nil && !boundScope.IsInterface {
				inferred = bound
			}
		}
		bindings[typeParameter.Name] = inferred
	}
	return bindings
}

// javaInferenceLeastUpperBound returns a deterministic, representable Java
// least upper bound for method-inference constraints. Java can describe some
// LUBs as intersection/capture types that have no direct generated-Go spelling;
// in those cases Object is the safe erased view.
func javaInferenceLeastUpperBound(javaTypes []string, ctx Ctx) string {
	var result string
	for _, javaType := range javaTypes {
		javaType = strings.TrimSpace(javaType)
		if javaType == "" {
			continue
		}
		if result == "" {
			result = javaType
			continue
		}
		result = javaInferencePairLeastUpperBound(result, javaType, ctx)
	}
	return result
}

func javaInferencePairLeastUpperBound(left, right string, ctx Ctx) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	if javaInferenceSameType(left, right, ctx) {
		return left
	}

	// Preserve an existing wider candidate. Besides ordinary class inheritance,
	// this handles Object and covariant arrays without rebuilding their spelling.
	if javaInferenceTypeAssignable(left, right, ctx) {
		return right
	}
	if javaInferenceTypeAssignable(right, left, ctx) {
		return left
	}

	leftComponent, leftArray := javaArrayComponentType(left)
	rightComponent, rightArray := javaArrayComponentType(right)
	if leftArray || rightArray {
		if !leftArray || !rightArray {
			return "Object"
		}

		// Array covariance applies only when both immediate components are
		// references. Thus Child[] and int[] meet at Object, while Child[][]
		// and Sibling[][] recursively retain their common Base[][] rank.
		_, leftPrimitive := javaPrimitiveType(leftComponent)
		_, rightPrimitive := javaPrimitiveType(rightComponent)
		if leftPrimitive || rightPrimitive {
			return "Object"
		}
		componentLUB := javaInferencePairLeastUpperBound(leftComponent, rightComponent, ctx)
		if componentLUB == "" {
			return "Object"
		}
		return componentLUB + "[]"
	}

	leftBase, leftArgs := parseJavaTypeString(left)
	rightBase, rightArgs := parseJavaTypeString(right)
	leftScope := resolveClassScopeByQualifiedName(ctx, leftBase)
	rightScope := resolveClassScopeByQualifiedName(ctx, rightBase)
	if leftScope == nil || rightScope == nil {
		return "Object"
	}
	if leftScope == rightScope {
		// Invariant generic instantiations with distinct arguments have a
		// wildcard LUB in Java. The raw class is its representable erased view.
		if len(leftArgs) > 0 || len(rightArgs) > 0 {
			return javaInferenceTypeName(leftScope)
		}
		return left
	}

	if common := javaInferenceCommonSuperclass(leftScope, rightScope, ctx); common != nil {
		return javaInferenceTypeName(common)
	}
	return "Object"
}

func javaInferenceSameType(left, right string, ctx Ctx) bool {
	leftBase, leftRank := javaArrayTypeParts(left)
	rightBase, rightRank := javaArrayTypeParts(right)
	if leftRank != rightRank {
		return false
	}
	leftRaw, leftArgs := parseJavaTypeString(leftBase)
	rightRaw, rightArgs := parseJavaTypeString(rightBase)
	leftScope := resolveClassScopeByQualifiedName(ctx, leftRaw)
	rightScope := resolveClassScopeByQualifiedName(ctx, rightRaw)
	if leftScope != nil && rightScope != nil {
		if leftScope != rightScope {
			return false
		}
	} else if !sameJavaRawType(leftRaw, rightRaw) {
		return false
	}
	if len(leftArgs) != len(rightArgs) {
		return false
	}
	for index := range leftArgs {
		if !javaInferenceSameType(leftArgs[index], rightArgs[index], ctx) {
			return false
		}
	}
	return true
}

func javaInferenceTypeAssignable(actual, expected string, ctx Ctx) bool {
	if javaInferenceSameType(actual, expected, ctx) {
		return true
	}
	actualBase, actualRank := javaArrayTypeParts(actual)
	expectedBase, expectedRank := javaArrayTypeParts(expected)
	if expectedRank == 0 && stripJavaQualifier(expectedBase) == "Object" {
		_, scalarPrimitive := javaPrimitiveType(actualBase)
		return actualRank > 0 || !scalarPrimitive
	}
	if actualRank > 0 || expectedRank > 0 {
		if actualRank == 0 || expectedRank == 0 {
			return false
		}
		actualComponent := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(actual), "[]"))
		expectedComponent := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(expected), "[]"))
		_, actualPrimitive := javaPrimitiveType(actualComponent)
		_, expectedPrimitive := javaPrimitiveType(expectedComponent)
		if actualPrimitive || expectedPrimitive {
			return actualPrimitive && expectedPrimitive && javaInferenceSameType(actualComponent, expectedComponent, ctx)
		}
		return javaInferenceTypeAssignable(actualComponent, expectedComponent, ctx)
	}

	actualPrimitive, actualIsPrimitive := javaPrimitiveType(actualBase)
	expectedPrimitive, expectedIsPrimitive := javaPrimitiveType(expectedBase)
	if actualIsPrimitive || expectedIsPrimitive {
		return actualIsPrimitive && expectedIsPrimitive && actualPrimitive == expectedPrimitive
	}

	actualRaw, actualArgs := parseJavaTypeString(actualBase)
	expectedRaw, expectedArgs := parseJavaTypeString(expectedBase)
	actualScope := resolveClassScopeByQualifiedName(ctx, actualRaw)
	expectedScope := resolveClassScopeByQualifiedName(ctx, expectedRaw)
	if actualScope == nil || expectedScope == nil {
		return false
	}
	if actualScope == expectedScope {
		if len(actualArgs) == 0 || len(expectedArgs) == 0 {
			return true
		}
		return javaInferenceSameType(actual, expected, ctx)
	}
	return javaReferenceTypeAssignable(actualScope, expectedScope, ctx)
}

// javaInferenceCommonSuperclass follows Java's single superclass chain, which
// makes selection stable regardless of map iteration. Shared interfaces can
// form intersection LUBs, so they deliberately fall back to Object above.
func javaInferenceCommonSuperclass(left, right *symbol.ClassScope, ctx Ctx) *symbol.ClassScope {
	leftAncestors := map[*symbol.ClassScope]struct{}{}
	seen := map[*symbol.ClassScope]struct{}{}
	for scope := left; scope != nil; scope = resolveSuperclassScopeInDeclaringContext(ctx, scope) {
		if _, exists := seen[scope]; exists {
			break
		}
		seen[scope] = struct{}{}
		if !scope.IsInterface {
			leftAncestors[scope] = struct{}{}
		}
	}

	seen = map[*symbol.ClassScope]struct{}{}
	for scope := right; scope != nil; scope = resolveSuperclassScopeInDeclaringContext(ctx, scope) {
		if _, exists := seen[scope]; exists {
			break
		}
		seen[scope] = struct{}{}
		if _, common := leftAncestors[scope]; common && !scope.IsInterface {
			return scope
		}
	}
	return nil
}

func javaInferenceTypeName(scope *symbol.ClassScope) string {
	if scope == nil || scope.Class == nil {
		return "Object"
	}
	return qualifyJavaTypeInDeclaringContext(scope.Class.OriginalName, scope)
}

func genericArrayInvocationExpectedTypes(def *symbol.Definition, invocationNode *sitter.Node, ctx Ctx, source []byte) []string {
	expected := definitionParameterOriginalTypes(def)
	bindings := genericArrayInvocationTypeBindings(def, invocationNode, ctx, source)
	if len(bindings) == 0 {
		return expected
	}
	for index, javaType := range expected {
		expected[index] = substituteJavaTypeParameters(javaType, bindings)
	}
	return expected
}

// genericArrayFormalNeedsExplicitTypeArguments identifies generic calls for
// which the common *ReferenceArray ABI has erased the Go parameter shape that
// previously let the compiler infer T. Java has already inferred T from the
// source-level T[] formal, so call lowering must spell that argument explicitly.
func genericArrayFormalNeedsExplicitTypeArguments(def *symbol.Definition) bool {
	if def == nil || len(def.TypeParameters) == 0 {
		return false
	}
	for _, parameter := range def.Parameters {
		base, rank := javaArrayTypeParts(parameter.OriginalType)
		if rank == 0 {
			continue
		}
		for _, typeParameter := range def.TypeParameters {
			if stripJavaQualifier(base) == typeParameter.Name {
				return true
			}
		}
	}
	return false
}

func maybeRewriteInstanceGenericMethodInvocationWithTarget(target *invocationTargetInfo, resolved *methodResolution, objectExpr ast.Expr, methodName string, args []ast.Expr, invocationNode *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	if target == nil {
		return nil
	}

	if resolved == nil {
		resolved = findInstanceMethodInHierarchy(target.classScope, methodName, len(args), ctx)
	}
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
			Sel: &ast.Ident{Name: executionMethodCallName(helperDef, ownerScope, ctx)},
		},
		Args: prependExecutionMethodArgument(ctx, helperDef, args),
	}
}
