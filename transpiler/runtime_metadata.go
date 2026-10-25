package transpiler

import (
	"bytes"
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// A dynamically supplied binary name can select any source class in the
// compilation. Enable descriptors for the complete source set in that case.
var reflectionUsePattern = regexp.MustCompile(`\b(newUpdater|getClass|getSimpleName|super\s*\.\s*toString|forName|getConstructor|getField|getDeclaredField|getDeclaredFields|getDeclaredConstructor|getGenericSuperclass|getGenericType|getTypeParameters|getAnnotation|getMethod|getDeclaredMethod|getDeclaredMethods|isEnum|isEnumConstant|isAnnotationPresent|getSuperclass|isAssignableFrom|isInstance)\b`)

func sourceUsesReflection() bool {
	seen := map[*symbol.FileScope]bool{}
	for _, scope := range allSourceClassScopes() {
		file := findFileScopeForClassScope(scope)
		// Exclude only source that cannot contain a volatile modifier. Escaped
		// keywords, comments, strings, and file-less declarations retain the
		// exact AST check, including local and anonymous class fields.
		if scope.Class != nil && (file == nil || len(file.Source) == 0 || bytes.Contains(file.Source, []byte("volatile")) || bytes.Contains(file.Source, []byte("\\u"))) && sourceNodeDeclaresVolatileField(scope.Class.DeclarationNode) {
			return true
		}
		if file == nil || seen[file] {
			continue
		}
		seen[file] = true
		if reflectionUsePattern.Match(file.Source) {
			return true
		}
	}
	return false
}

func reflectionPublic(def *symbol.Definition, scope *symbol.ClassScope) bool {
	if def == nil || def.DeclarationNode == nil {
		return false
	}
	file := findFileScopeForClassScope(scope)
	if file == nil {
		return false
	}
	for _, child := range nodeutil.NamedChildrenOf(def.DeclarationNode) {
		if child.Type() == "modifiers" {
			for _, modifier := range strings.Fields(child.Content(file.Source)) {
				if modifier == "public" {
					return true
				}
			}
		}
	}
	return false
}
func metadataString(value string) ast.Expr {
	return &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(value)}
}

// Encode only literals explicitly declared to be generated Go selectors.
// Ordinary Java Name/SimpleName/TypeID/string values keep their exact text.
func metadataGoName(value string) ast.Expr {
	return metadataString(symbol.GoIdentifier(value))
}
func metadataKey(name string, value ast.Expr) ast.Expr {
	return &ast.KeyValueExpr{Key: ast.NewIdent(name), Value: value}
}
func sourceClassMetadataStmt(scope *symbol.ClassScope, ctx Ctx) ast.Stmt {
	return sourceClassMetadataForTypeIDStmt(scope, sourceClassRuntimeTypeID(scope, ctx), ctx)
}
func sourceClassMetadataForTypeIDStmt(scope *symbol.ClassScope, id string, ctx Ctx) ast.Stmt {
	if scope == nil || scope.Class == nil || (!sourceUsesReflection() && !scope.IsEnum) {
		return nil
	}
	simpleName := ""
	if scope.Class != nil && scope.Class.DeclarationNode != nil {
		if name := scope.Class.DeclarationNode.ChildByFieldName("name"); name != nil {
			if file := findFileScopeForClassScope(scope); file != nil {
				simpleName = name.Content(file.Source)
			}
		}
	}
	descriptor := &ast.CompositeLit{Type: stdjavaQualifiedExpr("ClassDescriptor", ctx), Elts: []ast.Expr{
		metadataKey("Type", javaTypeIDLiteral(id, ctx)),
		metadataKey("SimpleName", metadataString(simpleName)),
		metadataKey("HasSimpleName", ast.NewIdent("true")),
		metadataKey("Interface", ast.NewIdent(strconv.FormatBool(scope.IsInterface))),
		metadataKey("Enum", ast.NewIdent(strconv.FormatBool(scope.IsEnum))),
	}}
	extendSourceReflectionMetadata(descriptor, scope, ctx)
	if sourceReflectionRecord(scope) {
		descriptor.Elts = append(descriptor.Elts, metadataKey("PartialMemberMetadata", ast.NewIdent("true")))
		for _, entry := range descriptor.Elts {
			if key, ok := entry.(*ast.KeyValueExpr); ok {
				if name, ok := key.Key.(*ast.Ident); ok && name.Name == "GenericSuperclass" {
					key.Value = &ast.UnaryExpr{Op: token.AND, X: &ast.CompositeLit{Type: stdjavaQualifiedExpr("ReflectTypeDescriptor", ctx), Elts: []ast.Expr{metadataKey("Kind", stdjavaQualifiedExpr("ReflectClassKind", ctx)), metadataKey("Raw", stdjavaQualifiedExpr("RecordTypeID", ctx))}}}
				}
			}
		}
	}
	execution := ast.NewIdent("execution")
	if initialize := classInitializationEnsureCall(scope, execution, ctx); initialize != nil {
		target := classInitializationTarget(scope, ctx)
		descriptor.Elts = append(descriptor.Elts, metadataKey("Initialization", qualifiedNameExpr(classInitializationStateName(target), findJavaPackageForClassScope(target), ctx)))
		descriptor.Elts = append(descriptor.Elts, metadataKey("Initialize", &ast.FuncLit{
			Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: initialize}}},
		}))
	}
	// Inner/local/generic classes require extra ABI arguments and are outside
	// this public no-argument reflection surface.
	if !sourceReflectionRecord(scope) && !scope.IsInterface && !scope.IsAbstract && !scope.IsEnum && !scope.IsInner && len(scope.TypeParameters) == 0 && !sourceReflectionAnonymousClass(scope) {
		constructor := noArgConstructorName(scope)
		var definition *symbol.Definition
		public := reflectionPublic(scope.Class, scope)
		for _, method := range scope.Methods {
			if method.Constructor && len(method.Parameters) == 0 {
				definition = method
				public = reflectionPublic(method, scope)
			}
		}
		if constructor != "" && public && len(sourceReflectionConstructorCaptures(scope, ctx)) == 0 {
			call := &ast.CallExpr{Fun: ast.NewIdent(executionConstructorImplementationName(constructor, scope)), Args: []ast.Expr{execution}}
			if definition != nil && len(definition.TypeParameters) > 0 {
				// A noarg source constructor can still have generic ABI arguments.
				// Reuse the declared-constructor erasure/witness callback and keep
				// this legacy unary metadata boundary on the caller's execution.
				call = &ast.CallExpr{Fun: sourceReflectionConstructorCallback(scope, definition, constructor, nil, ctx), Args: []ast.Expr{execution, ast.NewIdent("nil")}}
			}
			descriptor.Elts = append(descriptor.Elts, metadataKey("Construct", &ast.FuncLit{
				Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}},
			}))
		}
	}
	fields := &ast.CompositeLit{Type: &ast.ArrayType{Elt: stdjavaQualifiedExpr("FieldDescriptor", ctx)}}
	methods := &ast.CompositeLit{Type: &ast.ArrayType{Elt: stdjavaQualifiedExpr("MethodDescriptor", ctx)}}
	if scope.IsEnum {
		fields.Elts = append(fields.Elts, enumReflectionFieldDescriptors(scope, ctx)...)
	}
	if !scope.IsEnum {
		fields.Elts = append(fields.Elts, sourceReflectionFields(scope, ctx)...)
	}
	methods.Elts = append(methods.Elts, sourceReflectionMethods(scope, ctx)...)

	descriptor.Elts = append(descriptor.Elts, metadataKey("Fields", fields), metadataKey("Methods", methods))
	annotations := &ast.CompositeLit{Type: &ast.ArrayType{Elt: stdjavaQualifiedExpr("TypeID", ctx)}}
	annotationValues := sourceAnnotationDescriptors(scope.Class.DeclarationNode, scope, ctx)
	if values, ok := annotationValues.(*ast.CompositeLit); ok {
		for _, value := range values.Elts {
			if descriptor, ok := value.(*ast.CompositeLit); ok {
				for _, element := range descriptor.Elts {
					if pair, ok := element.(*ast.KeyValueExpr); ok {
						if key, ok := pair.Key.(*ast.Ident); ok && key.Name == "Type" {
							annotations.Elts = append(annotations.Elts, pair.Value)
						}
					}
				}
			}
		}
	}
	descriptor.Elts = append(descriptor.Elts, metadataKey("Annotations", annotations))
	descriptor.Elts = append(descriptor.Elts, metadataKey("AnnotationValues", annotationValues))
	if _, inherited := runtimeMarkerPolicy(scope); inherited {
		descriptor.Elts = append(descriptor.Elts, metadataKey("InheritedAnnotation", ast.NewIdent("true")))
	}

	return &ast.ExprStmt{X: stdjavaCall(ctx, "RegisterClassDescriptor", descriptor)}
}

func init() {
	registerStaticIntrinsic("Class", "forName", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "ClassForNameJavaString", intrinsicExecutionExpr(ctx), args[0])
	})
	registerStaticIntrinsicResultType("Class", "forName", "Class")
	for receiver, methods := range map[string]map[string]string{
		"Class":       {"getName": "java.lang.String", "getSimpleName": "java.lang.String", "getSuperclass": "Class", "isAssignableFrom": "boolean", "isInstance": "boolean", "isAnnotationPresent": "boolean", "getConstructor": "Constructor", "getField": "Field", "getDeclaredField": "Field", "isEnum": "boolean", "getMethod": "Method"},
		"Field":       {"get": "Object", "set": "void", "getName": "java.lang.String", "isEnumConstant": "boolean"},
		"Method":      {"invoke": "Object", "getName": "java.lang.String", "isVarArgs": "boolean"},
		"Constructor": {"newInstance": "Object"},
	} {
		for name, result := range methods {
			receiver, name := receiver, name
			if receiver == "Class" && name == "getSimpleName" {
				registerInstanceIntrinsic(receiver, name, classJavaSimpleNameIntrinsic)
				registerInstanceIntrinsicResultType(receiver, name, result)
				continue
			}
			if receiver == "Class" && name == "getName" {
				registerInstanceIntrinsic(receiver, name, classJavaNameIntrinsic)
				registerInstanceIntrinsicResultType(receiver, name, result)
				continue
			}
			registerInstanceIntrinsic(receiver, name, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
				if (receiver == "Field" || receiver == "Method") && name == "getName" {
					return selectorCall(recv, "GetNameJavaString", args)
				}
				if receiver == "Class" && (name == "getField" || name == "getDeclaredField" || name == "getMethod") {
					return selectorCall(recv, strings.ToUpper(name[:1])+name[1:]+"JavaString", args)
				}
				if receiver == "Field" && name == "get" {
					return &ast.CallExpr{Fun: &ast.SelectorExpr{X: recv, Sel: ast.NewIdent("GetExecution")}, Args: append([]ast.Expr{intrinsicExecutionExpr(ctx)}, args...)}
				}
				if (receiver == "Method" && name == "invoke") || (receiver == "Constructor" && name == "newInstance") {
					args = append([]ast.Expr{intrinsicExecutionExpr(ctx)}, args...)
				}
				return &ast.CallExpr{Fun: &ast.SelectorExpr{X: recv, Sel: ast.NewIdent(strings.ToUpper(name[:1]) + name[1:])}, Args: args}
			})
			if (receiver == "Class" && (name == "getConstructor" || name == "getMethod")) || (receiver == "Method" && name == "invoke") || (receiver == "Constructor" && name == "newInstance") {
				registerInstanceNodeIntrinsic(receiver, name, reflectionVarargsIntrinsic(receiver, name))
			}
			registerInstanceIntrinsicResultType(receiver, name, result)
		}
	}
}

// Java reflection methods accept an explicit varargs array as well as expanded
// arguments. Preserve that distinction, including (Object)null versus
// (Object[])null, before the two expressions erase to the same Go value.
func reflectionVarargsIntrinsic(receiver, name string) nodeIntrinsicGenerator {
	return func(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		object := invocation.ChildByFieldName("object")
		if object == nil {
			return nil
		}
		nodes := nodeutil.SemanticNamedChildrenOf(invocation.ChildByFieldName("arguments"))
		fixed := 0
		if name == "getMethod" || name == "getDeclaredMethod" || name == "invoke" {
			fixed = 1
		}
		var args []ast.Expr
		if receiver == "Method" || receiver == "Constructor" {
			// Expanded reflection arguments have the canonical Object formal, so
			// primitive expressions must be boxed before Method/Constructor runtime
			// conversion. An explicit varargs array retains its array boundary.
			expected := make([]string, len(nodes))
			for index := range expected {
				expected[index] = "java.lang.Object"
			}
			if len(nodes) == fixed+1 {
				last := nodes[fixed]
				javaType, _ := inferExprJavaType(last, ctx, source)
				if last.Type() == "null_literal" || strings.HasSuffix(strings.TrimSpace(javaType), "[]") {
					expected[fixed] = "java.lang.Object[]"
				}
			}
			args = parseArgumentListWithExpectedTypes(invocation.ChildByFieldName("arguments"), source, ctx, expected)
		} else {
			args = intrinsicArgs(object, name, source, ctx)
		}
		spread := false
		if len(nodes) == fixed+1 {
			last := nodes[fixed]
			// A null value cast to Object is a single argument; do not send
			// it through Go's non-null interface assertion path.
			if last.Type() == "cast_expression" {
				value := last.ChildByFieldName("value")
				if value != nil && value.Type() == "null_literal" {
					args[fixed] = ast.NewIdent("nil")
				}
			}
			javaType, _ := inferExprJavaType(last, ctx, source)
			if last.Type() == "null_literal" || strings.HasSuffix(strings.TrimSpace(javaType), "[]") {
				helper := "ReflectionArguments"
				if receiver == "Class" {
					helper = "ReflectionClassArguments"
				}
				args[fixed] = stdjavaCall(ctx, helper, args[fixed])
				spread = true
			}
		}
		if receiver == "Method" || receiver == "Constructor" {
			args = append([]ast.Expr{intrinsicExecutionExpr(ctx)}, args...)
		}
		selector := strings.ToUpper(name[:1]) + name[1:]
		if (receiver == "Method" && name == "invoke") || (receiver == "Constructor" && name == "newInstance") {
			if caller, owned := reflectionCallerExpression(invocation, ctx); owned {
				args = append([]ast.Expr{args[0], caller}, args[1:]...)
				if receiver == "Method" {
					selector = "InvokeWithCallerExecution"
				} else {
					selector = "NewInstanceWithCallerExecution"
				}
			} else if ctx.runtimeLocationProfile == sourceRuntimeLocationClasspath {
				return unsupportedIntrinsicValue(invocation, "java.lang.Object", source, ctx)
			}
		}
		if receiver == "Class" && (name == "getMethod" || name == "getDeclaredMethod") {
			selector += "JavaString"
		}
		call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: recv, Sel: ast.NewIdent(selector)}, Args: args}
		if spread {
			call.Ellipsis = 1
		}
		if receiver == "Constructor" && name == "newInstance" {
			return reflectionTypedResultExpr(call, invocation, ctx, source)
		}
		return call
	}
}

func declarationAnnotations(scope *symbol.ClassScope) []*sitter.Node {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil {
		return nil
	}
	var annotations []*sitter.Node
	for _, child := range nodeutil.NamedChildrenOf(scope.Class.DeclarationNode) {
		if child.Type() != "modifiers" {
			continue
		}
		for _, node := range nodeutil.NamedChildrenOf(child) {
			if node.Type() == "marker_annotation" || node.Type() == "annotation" {
				annotations = append(annotations, node)
			}
		}
	}
	return annotations
}

// CLASS retention and declaration targets come from the actual annotation
// declaration's owned TYPE namespace, never from simple names or enum suffixes.
func runtimeMarkerPolicy(scope *symbol.ClassScope) (runtime, inherited bool) {
	policy, known := reflectionSourceAnnotationPolicy(scope)
	return known && policy.retention == "RUNTIME", known && policy.inherited
}
