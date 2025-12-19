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
		return superSelectorExpr(ctx)
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
						Names: []*ast.Ident{identFromNode(paramNode, source)},
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
			Sel: identFromNode(node.NamedChild(1), source),
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
			methodIdent := identFromNode(node.ChildByFieldName("name"), source)

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
			typeArgs := explicitTypeArgumentExprs(node, source, inScopeTypeParameters(ctx))

			// If this is a static call on a class name (e.g., Utils.<T>id(...)),
			// rewrite it to a plain function call to match how static methods are emitted.
			if classScope := resolveClassScopeByIdentifier(ctx, source, objectNode); classScope != nil {
				if staticDef := findStaticMethodByNameAndArgCount(classScope, methodName, len(args)); staticDef != nil {
					fun := ast.Expr(&ast.Ident{Name: staticDef.Name})
					fun = applyTypeArguments(fun, typeArgs)
					return &ast.CallExpr{Fun: fun, Args: args}
				}
			}

			target := resolveInvocationTarget(objectNode, ctx, source)
			if rewritten := maybeRewriteInstanceGenericMethodInvocationWithTarget(target, objectExpr, methodName, args, node, ctx, source); rewritten != nil {
				return rewritten
			}

			if target != nil {
				if resolved := findInstanceMethodInHierarchy(target.classScope, methodName, len(args), ctx); resolved != nil {
					methodIdent = &ast.Ident{Name: resolved.def.Name}
				} else if resolved := findStaticMethodInHierarchy(target.classScope, methodName, len(args), ctx); resolved != nil {
					// Java permits calling static methods via an instance expression; rewrite
					// to a plain function call to match codegen.
					fun := ast.Expr(&ast.Ident{Name: resolved.def.Name})
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
		args := ParseNode(node.ChildByFieldName("arguments"), source, ctx).([]ast.Expr)
		typeArgs := explicitTypeArgumentExprs(node, source, inScopeTypeParameters(ctx))

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
			if resolved := findInstanceMethodInHierarchy(ctx.currentClass, methodName, len(args), ctx); resolved != nil {
				return &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   recv,
						Sel: &ast.Ident{Name: resolved.def.Name},
					},
					Args: args,
				}
			}
		}

		// Otherwise, treat as a plain function call (static methods are emitted as
		// functions).
		if ctx.currentClass != nil {
			if resolved := findStaticMethodInHierarchy(ctx.currentClass, methodName, len(args), ctx); resolved != nil {
				fun := ast.Expr(&ast.Ident{Name: resolved.def.Name})
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

		// Find the respective constructor (if we have symbol info for that class).
		var constructor *symbol.Definition
		targetScope := ctx.currentClass
		if ctx.currentFile != nil && ctx.currentFile.BaseClass != nil {
			if found := findClassScopeByName(ctx.currentFile.BaseClass, className); found != nil {
				targetScope = found
			}
		}
		constructor = findMatchingConstructor(targetScope, className, argumentTypes)

		// Helper function to add type arguments to a function expression
		addTypeArgs := func(funExpr ast.Expr, args []string) ast.Expr {
			if len(args) == 0 {
				return funExpr
			}
			scopeTypeParams := inScopeTypeParameters(ctx)
			typeArgExprs := make([]ast.Expr, 0, len(args))
			for _, ta := range args {
				typeArgExprs = append(typeArgExprs, javaTypeStringToGoTypeExpr(ta, scopeTypeParams))
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
			parentArgs = append(parentArgs, javaTypeStringToGoTypeExpr(a, scopeTypeParams))
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
func javaTypeStringToGoTypeExpr(typeStr string, typeParams []string) ast.Expr {
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
			return javaTypeStringToGoTypeExpr(bound, typeParams)
		}
		// ? super ... is hard to model faithfully in Go; fall back to any.
		return &ast.Ident{Name: "any"}
	}

	// Normalize qualifiers.
	base, typeArgs := parseJavaTypeString(typeStr)
	base = stripJavaQualifier(base)

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
	if prim, ok := primitive(base); ok {
		expr = prim
	} else if isTypeParam(base) {
		expr = &ast.Ident{Name: base}
	} else {
		// Reference type (including parameterized reference types) is represented as a pointer.
		baseIdent := &ast.Ident{Name: base}
		if len(typeArgs) > 0 {
			argExprs := make([]ast.Expr, 0, len(typeArgs))
			for _, arg := range typeArgs {
				argExprs = append(argExprs, javaTypeStringToGoTypeExpr(arg, typeParams))
			}
			expr = &ast.StarExpr{X: applyTypeArguments(baseIdent, argExprs)}
		} else {
			expr = &ast.StarExpr{X: baseIdent}
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
	}
	return "", false
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
		classTypeArgExprs = append(classTypeArgExprs, javaTypeStringToGoTypeExpr(arg, scopeTypeParams))
	}

	return &invocationTargetInfo{
		classScope:    classScope,
		classTypeArgs: classTypeArgExprs,
	}
}

func explicitTypeArgumentExprs(node *sitter.Node, source []byte, typeParams []string) []ast.Expr {
	typeArgsNode := node.ChildByFieldName("type_arguments")
	if typeArgsNode == nil {
		return nil
	}
	var exprs []ast.Expr
	for _, arg := range nodeutil.NamedChildrenOf(typeArgsNode) {
		exprs = append(exprs, javaTypeStringToGoTypeExpr(arg.Content(source), typeParams))
	}
	return exprs
}

func inferMethodTypeArguments(def *symbol.Definition, invocationNode *sitter.Node, ctx Ctx, source []byte) []ast.Expr {
	if len(def.TypeParameters) == 0 {
		return nil
	}

	if explicit := explicitTypeArgumentExprs(invocationNode, source, inScopeTypeParameters(ctx)); len(explicit) == len(def.TypeParameters) && len(explicit) > 0 {
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
					resolved[tp.Name] = javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx))
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

func maybeRewriteInstanceGenericMethodInvocation(objectNode *sitter.Node, objectExpr ast.Expr, methodName string, args []ast.Expr, invocationNode *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	target := resolveInvocationTarget(objectNode, ctx, source)
	return maybeRewriteInstanceGenericMethodInvocationWithTarget(target, objectExpr, methodName, args, invocationNode, ctx, source)
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

	constructorIdent := &ast.Ident{Name: "New" + helperDef.HelperName}
	helperConstructor := applyTypeArguments(constructorIdent, helperTypeArgs)
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
