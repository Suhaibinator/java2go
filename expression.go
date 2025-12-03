package main

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
		// This can either be a pre or post expression
		// a pre expression has the identifier second, while the post expression
		// has the identifier first

		// Post-update expression, e.g. `i++`
		if node.Child(0).IsNamed() {
			return &ast.CallExpr{
				Fun:  &ast.Ident{Name: "PostUpdate"},
				Args: []ast.Expr{ParseExpr(node.Child(0), source, ctx)},
			}
		}

		// Otherwise, pre-update expression
		return &ast.CallExpr{
			Fun:  &ast.Ident{Name: "PreUpdate"},
			Args: []ast.Expr{ParseExpr(node.Child(1), source, ctx)},
		}
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
		return &ast.BadExpr{}
	case "lambda_expression":
		// Lambdas can either be called with a list of expressions
		// (ex: (n1, n1) -> {}), or with a single expression
		// (ex: n1 -> {})

		var lambdaBody *ast.BlockStmt

		var lambdaParameters *ast.FieldList

		bodyNode := node.ChildByFieldName("body")

		switch bodyNode.Type() {
		case "block":
			lambdaBody = ParseStmt(bodyNode, source, ctx).(*ast.BlockStmt)
		default:
			// Lambdas can be called inline without a block expression
			lambdaBody = &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{
						X: ParseExpr(bodyNode, source, ctx),
					},
				},
			}
		}

		paramNode := node.ChildByFieldName("parameters")

		switch paramNode.Type() {
		case "inferred_parameters", "formal_parameters":
			lambdaParameters = ParseNode(paramNode, source, ctx).(*ast.FieldList)
		default:
			// If we can't identify the types of the parameters, then just set their
			// types to any
			lambdaParameters = &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Names: []*ast.Ident{ParseExpr(paramNode, source, ctx).(*ast.Ident)},
						Type:  &ast.Ident{Name: "any"},
					},
				},
			}
		}

		return &ast.FuncLit{
			Type: &ast.FuncType{
				Params: lambdaParameters,
			},
			Body: lambdaBody,
		}
	case "method_reference":
		// This refers to manually selecting a function from a specific class and
		// passing it in as an argument in the `func(className::methodName)` style

		// For class constructors such as `Class::new`, you only get one node
		if node.NamedChildCount() < 2 {
			return &ast.SelectorExpr{
				X:   ParseExpr(node.NamedChild(0), source, ctx),
				Sel: &ast.Ident{Name: "new"},
			}
		}

		return &ast.SelectorExpr{
			X:   ParseExpr(node.NamedChild(0), source, ctx),
			Sel: ParseExpr(node.NamedChild(1), source, ctx).(*ast.Ident),
		}
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
			methodIdent := ParseExpr(node.ChildByFieldName("name"), source, ctx).(*ast.Ident)

			// Check if this is an enum values() call
			// Transform EnumName.values() to EnumNameValues()
			if objectNode.Type() == "identifier" && methodName == "values" {
				enumName := objectNode.Content(source)
				// Check if this identifier refers to an enum
				if ctx.currentFile != nil && ctx.currentFile.BaseClass != nil {
					// Check base class
					if ctx.currentFile.BaseClass.Class.OriginalName == enumName && ctx.currentFile.BaseClass.IsEnum {
						return &ast.CallExpr{
							Fun:  &ast.Ident{Name: ctx.currentFile.BaseClass.Class.Name + "Values"},
							Args: []ast.Expr{},
						}
					}
					// Check subclasses
					for _, subclass := range ctx.currentFile.BaseClass.Subclasses {
						if subclass.Class.OriginalName == enumName && subclass.IsEnum {
							return &ast.CallExpr{
								Fun:  &ast.Ident{Name: subclass.Class.Name + "Values"},
								Args: []ast.Expr{},
							}
						}
					}
				}
			}

			objectExpr := ParseExpr(objectNode, source, ctx)
			args := ParseNode(node.ChildByFieldName("arguments"), source, ctx).([]ast.Expr)
			if rewritten := maybeRewriteInstanceGenericMethodInvocation(objectNode, objectExpr, methodName, args, node, ctx, source); rewritten != nil {
				return rewritten
			}

			return &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   objectExpr,
					Sel: methodIdent,
				},
				Args: args,
			}
		}
		return &ast.CallExpr{
			Fun:  ParseExpr(node.ChildByFieldName("name"), source, ctx),
			Args: ParseNode(node.ChildByFieldName("arguments"), source, ctx).([]ast.Expr),
		}
	case "object_creation_expression":
		// This is called when anything is created with a constructor

		objectType := node.ChildByFieldName("type")

		// TODO: Handle case where object is created with this format:
		// parentClass.new NestedClass()
		// (when node.NamedChild(0) != objectType)

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

		var constructor *symbol.Definition
		// Find the respective constructor, and call it
		constructor = ctx.currentClass.FindMethodByName(className, argumentTypes)

		// Helper function to add type arguments to a function expression
		addTypeArgs := func(funExpr ast.Expr, args []string) ast.Expr {
			if len(args) == 0 {
				return funExpr
			}
			typeArgExprs := make([]ast.Expr, len(args))
			for i, ta := range args {
				typeArgExprs[i] = &ast.Ident{Name: ta}
			}
			if len(typeArgExprs) == 1 {
				return &ast.IndexExpr{
					X:     funExpr,
					Index: typeArgExprs[0],
				}
			}
			return &ast.IndexListExpr{
				X:       funExpr,
				Indices: typeArgExprs,
			}
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
						effectiveTypeArgs = ctx.currentClass.TypeParameters
						break
					}
				}
			}
		}

		if constructor != nil {
			funExpr := addTypeArgs(&ast.Ident{Name: constructor.Name}, effectiveTypeArgs)
			return &ast.CallExpr{
				Fun:  funExpr,
				Args: arguments,
			}
		}

		// It is also possible that a constructor could be unresolved, so we handle
		// this by calling the type of the type + "Construct" at the beginning
		funExpr := addTypeArgs(&ast.Ident{Name: "Construct" + className}, effectiveTypeArgs)
		return &ast.CallExpr{
			Fun:  funExpr,
			Args: arguments,
		}
	case "array_creation_expression":
		dimensions := []ast.Expr{}
		arrayType := astutil.ParseType(node.ChildByFieldName("type"), source)
		var initializer ast.Expr

		for _, child := range nodeutil.NamedChildrenOf(node) {
			if child.Type() == "dimensions_expr" {
				dimensions = append(dimensions, ParseExpr(child, source, ctx))
			} else if child.Type() == "array_initializer" {
				initCtx := ctx.Clone()
				initCtx.lastType = arrayType
				initializer = ParseExpr(child, source, initCtx)
			}
		}

		if initializer != nil {
			return initializer
		}

		if len(dimensions) == 0 {
			panic("Array had zero dimensions")
		}

		return GenMultiDimArray(symbol.NodeToStr(arrayType), dimensions)
	case "instanceof_expression":
		return &ast.BadExpr{}
	case "dimensions_expr":
		return ParseExpr(node.NamedChild(0), source, ctx)
	case "binary_expression":
		if node.Child(1).Content(source) == ">>>" {
			return &ast.CallExpr{
				Fun:  &ast.Ident{Name: "UnsignedRightShift"},
				Args: []ast.Expr{ParseExpr(node.Child(0), source, ctx), ParseExpr(node.Child(2), source, ctx)},
			}
		}
		return &ast.BinaryExpr{
			X:  ParseExpr(node.Child(0), source, ctx),
			Op: StrToToken(node.Child(1).Content(source)),
			Y:  ParseExpr(node.Child(2), source, ctx),
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
		return &ast.CallExpr{
			Fun:  &ast.Ident{Name: "ternary"},
			Args: args,
		}
	case "cast_expression":
		// TODO: This probably should be a cast function, instead of an assertion
		return &ast.TypeAssertExpr{
			X:    ParseExpr(node.NamedChild(1), source, ctx),
			Type: astutil.ParseType(node.NamedChild(0), source),
		}
	case "field_access":
		// X.Sel
		obj := node.ChildByFieldName("object")

		if obj.Type() == "this" {
			def := ctx.currentClass.FindField().ByOriginalName(node.ChildByFieldName("field").Content(source))
			if len(def) == 0 {
				// TODO: This field could not be found in the current class, because it exists in the superclass
				// definition for the class
				def = []*symbol.Definition{&symbol.Definition{
					Name: node.ChildByFieldName("field").Content(source),
				}}
			}

			return &ast.SelectorExpr{
				X:   ParseExpr(node.ChildByFieldName("object"), source, ctx),
				Sel: &ast.Ident{Name: def[0].Name},
			}
		}
		return &ast.SelectorExpr{
			X:   ParseExpr(obj, source, ctx),
			Sel: ParseExpr(node.ChildByFieldName("field"), source, ctx).(*ast.Ident),
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
		return &ast.Ident{Name: node.Content(source)}
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
	panic("Unhandled expression: " + node.Type())
}

func findClassScopeByName(scope *symbol.ClassScope, name string) *symbol.ClassScope {
	if scope == nil {
		return nil
	}
	if scope.Class.OriginalName == name {
		return scope
	}
	for _, sub := range scope.Subclasses {
		if found := findClassScopeByName(sub, name); found != nil {
			return found
		}
	}
	return nil
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

func javaTypeComponentToExpr(typeName string) ast.Expr {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return &ast.Ident{Name: "any"}
	}
	switch typeName {
	case "String":
		return &ast.Ident{Name: "string"}
	default:
		return &ast.Ident{Name: typeName}
	}
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
		if field := ctx.currentClass.FindFieldByName(name); field != nil && field.OriginalType != "" {
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
		return fmt.Sprintf("%s<%s>", base, strings.Join(ctx.currentClass.TypeParameters, ", ")), true
	case "object_creation_expression":
		typeNode := node.ChildByFieldName("type")
		if typeNode == nil {
			return "", false
		}
		return typeNode.Content(source), true
	}
	return "", false
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
	if ctx.currentFile == nil || ctx.currentFile.BaseClass == nil {
		return nil
	}

	var className string
	var classTypeArgs []string
	switch objectNode.Type() {
	case "this":
		if ctx.currentClass == nil {
			return nil
		}
		className = ctx.currentClass.Class.OriginalName
		classTypeArgs = ctx.currentClass.TypeParameters
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

	classScope := findClassScopeByName(ctx.currentFile.BaseClass, className)
	if classScope == nil {
		return nil
	}

	classTypeArgExprs := make([]ast.Expr, 0, len(classTypeArgs))
	for _, arg := range classTypeArgs {
		classTypeArgExprs = append(classTypeArgExprs, javaTypeComponentToExpr(arg))
	}

	return &invocationTargetInfo{
		classScope:    classScope,
		classTypeArgs: classTypeArgExprs,
	}
}

func explicitTypeArgumentExprs(node *sitter.Node, source []byte) []ast.Expr {
	typeArgsNode := node.ChildByFieldName("type_arguments")
	if typeArgsNode == nil {
		return nil
	}
	var exprs []ast.Expr
	for _, arg := range nodeutil.NamedChildrenOf(typeArgsNode) {
		exprs = append(exprs, javaTypeComponentToExpr(arg.Content(source)))
	}
	return exprs
}

func inferMethodTypeArguments(def *symbol.Definition, invocationNode *sitter.Node, ctx Ctx, source []byte) []ast.Expr {
	if len(def.TypeParameters) == 0 {
		return nil
	}

	if explicit := explicitTypeArgumentExprs(invocationNode, source); len(explicit) == len(def.TypeParameters) && len(explicit) > 0 {
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
			if param.OriginalType == tp && idx < len(argNodes) {
				if javaType, ok := inferExprJavaType(argNodes[idx], ctx, source); ok {
					resolved[tp] = javaTypeComponentToExpr(javaType)
				}
			}
		}
	}

	result := make([]ast.Expr, len(def.TypeParameters))
	for i, tp := range def.TypeParameters {
		if expr, ok := resolved[tp]; ok {
			result[i] = expr
		} else {
			result[i] = &ast.Ident{Name: "any"}
		}
	}
	return result
}

func maybeRewriteInstanceGenericMethodInvocation(objectNode *sitter.Node, objectExpr ast.Expr, methodName string, args []ast.Expr, invocationNode *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	target := resolveInvocationTarget(objectNode, ctx, source)
	if target == nil {
		return nil
	}

	methodDefs := target.classScope.FindMethod().By(func(d *symbol.Definition) bool {
		return d.OriginalName == methodName
	})
	var helperDef *symbol.Definition
	for _, def := range methodDefs {
		if def.RequiresHelper {
			helperDef = def
			break
		}
	}
	if helperDef == nil {
		return nil
	}

	classTypeArgs := target.classTypeArgs
	methodTypeArgs := inferMethodTypeArguments(helperDef, invocationNode, ctx, source)
	helperTypeArgs := append(classTypeArgs, methodTypeArgs...)

	constructorIdent := &ast.Ident{Name: "New" + helperDef.HelperName}
	helperConstructor := applyTypeArguments(constructorIdent, helperTypeArgs)
	helperCall := &ast.CallExpr{
		Fun:  helperConstructor,
		Args: []ast.Expr{objectExpr},
	}

	selIdent := &ast.Ident{Name: helperDef.Name}

	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   helperCall,
			Sel: selIdent,
		},
		Args: args,
	}
}
