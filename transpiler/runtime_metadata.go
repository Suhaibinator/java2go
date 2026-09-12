package transpiler

import (
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
var reflectionUsePattern = regexp.MustCompile(`\b(forName|getConstructor|getField|getMethod|isAnnotationPresent|getSuperclass|isAssignableFrom)\b`)

func sourceUsesReflection() bool {
	seen := map[*symbol.FileScope]bool{}
	for _, scope := range allSourceClassScopes() {
		file := findFileScopeForClassScope(scope)
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
func metadataKey(name string, value ast.Expr) ast.Expr {
	return &ast.KeyValueExpr{Key: ast.NewIdent(name), Value: value}
}
func sourceClassMetadataStmt(scope *symbol.ClassScope, ctx Ctx) ast.Stmt {
	if !sourceUsesReflection() {
		return nil
	}
	descriptor := &ast.CompositeLit{Type: stdjavaQualifiedExpr("ClassDescriptor", ctx), Elts: []ast.Expr{
		metadataKey("Type", javaTypeIDLiteral(javaClassBinaryName(scope), ctx)),
		metadataKey("Interface", ast.NewIdent(strconv.FormatBool(scope.IsInterface))),
	}}
	execution := ast.NewIdent("execution")
	if initialize := classInitializationEnsureCall(scope, execution, ctx); initialize != nil {
		descriptor.Elts = append(descriptor.Elts, metadataKey("Initialize", &ast.FuncLit{
			Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: initialize}}},
		}))
	}
	// Inner/local/generic classes require extra ABI arguments and are outside
	// this public no-argument reflection surface.
	if !scope.IsInterface && !scope.IsAbstract && !scope.IsEnum && !scope.IsInner && len(scope.TypeParameters) == 0 {
		constructor := noArgConstructorName(scope)
		public := reflectionPublic(scope.Class, scope)
		for _, method := range scope.Methods {
			if method.Constructor && len(method.Parameters) == 0 {
				public = reflectionPublic(method, scope)
			}
		}
		if constructor != "" && public {
			descriptor.Elts = append(descriptor.Elts, metadataKey("Construct", &ast.FuncLit{
				Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.CallExpr{Fun: ast.NewIdent(executionConstructorImplementationName(constructor, scope)), Args: []ast.Expr{execution}}}}}},
			}))
		}
	}
	fields := &ast.CompositeLit{Type: &ast.ArrayType{Elt: stdjavaQualifiedExpr("FieldDescriptor", ctx)}}
	methods := &ast.CompositeLit{Type: &ast.ArrayType{Elt: stdjavaQualifiedExpr("MethodDescriptor", ctx)}}
	if len(scope.TypeParameters) == 0 && !scope.IsInterface && !scope.IsEnum {
		for _, field := range scope.Fields {
			if !reflectionPublic(field, scope) || field.IsStatic {
				continue
			}
			id, ok := javaTypeDescriptorExpr(field.OriginalType, ctx)
			if !ok {
				continue
			}
			fields.Elts = append(fields.Elts, &ast.CompositeLit{Elts: []ast.Expr{metadataKey("Name", metadataString(field.OriginalName)), metadataKey("GoName", metadataString(field.Name)), metadataKey("Type", id), metadataKey("Final", ast.NewIdent(strconv.FormatBool(field.IsFinal)))}})
		}
		for _, method := range scope.Methods {
			if method.Constructor || method.IsStatic || method.RequiresHelper || len(method.Parameters) != 0 || !reflectionPublic(method, scope) {
				continue
			}
			id, _ := javaTypeDescriptorExpr(method.OriginalType, ctx)
			if id == nil {
				id = javaTypeIDLiteral("void", ctx)
			}
			methods.Elts = append(methods.Elts, &ast.CompositeLit{Elts: []ast.Expr{metadataKey("Name", metadataString(method.OriginalName)), metadataKey("GoName", metadataString(executionImplementationName(method, scope))), metadataKey("Return", id)}})
		}
	}
	descriptor.Elts = append(descriptor.Elts, metadataKey("Fields", fields), metadataKey("Methods", methods))
	annotations := &ast.CompositeLit{Type: &ast.ArrayType{Elt: stdjavaQualifiedExpr("TypeID", ctx)}}
	for _, annotation := range declarationAnnotations(scope) {
		nameNode := annotation.ChildByFieldName("name")
		if nameNode == nil {
			continue
		}
		file := findFileScopeForClassScope(scope)
		name := nameNode.Content(file.Source)
		annotationScope := resolveClassScopeByQualifiedName(ctx, name)
		if annotationScope != nil {
			runtime, _ := runtimeMarkerPolicy(annotationScope)
			if runtime {
				annotations.Elts = append(annotations.Elts, javaTypeIDLiteral(javaClassBinaryName(annotationScope), ctx))
			}
		} else if annotationNameIs(name, "java.lang.Deprecated", scope) {
			annotations.Elts = append(annotations.Elts, javaTypeIDLiteral("java.lang.Deprecated", ctx))
		}
	}
	descriptor.Elts = append(descriptor.Elts, metadataKey("Annotations", annotations))
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
		return stdjavaCall(ctx, "ClassForName", intrinsicExecutionExpr(ctx), args[0])
	})
	registerStaticIntrinsicResultType("Class", "forName", "Class")
	for receiver, methods := range map[string]map[string]string{
		"Class":       {"getName": "String", "getSuperclass": "Class", "isAssignableFrom": "boolean", "isAnnotationPresent": "boolean", "getConstructor": "Constructor", "getField": "Field", "getMethod": "Method"},
		"Field":       {"get": "Object", "set": "void", "getName": "String"},
		"Method":      {"invoke": "Object", "getName": "String"},
		"Constructor": {"newInstance": "Object"},
	} {
		for name, result := range methods {
			receiver, name := receiver, name
			registerInstanceIntrinsic(receiver, name, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
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
		args := intrinsicArgs(object, name, source, ctx)
		nodes := nodeutil.NamedChildrenOf(invocation.ChildByFieldName("arguments"))
		fixed := 0
		if name == "getMethod" || name == "invoke" {
			fixed = 1
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
		call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: recv, Sel: ast.NewIdent(strings.ToUpper(name[:1]) + name[1:])}, Args: args}
		if spread {
			call.Ellipsis = 1
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
func annotationNameIs(name, qualified string, scope *symbol.ClassScope) bool {
	if name == qualified {
		return true
	}
	simple := stripJavaQualifier(qualified)
	if name != simple {
		return false
	}
	if file := findFileScopeForClassScope(scope); file != nil {
		if imported, ok := file.Imports[simple]; ok {
			return imported+"."+simple == qualified
		}
	}
	return true
}

// Only marker annotations are represented. CLASS is the Java default retention;
// RUNTIME must be explicitly declared, and @Inherited applies to class ancestry.
func runtimeMarkerPolicy(scope *symbol.ClassScope) (runtime, inherited bool) {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil || scope.Class.DeclarationNode.Type() != "annotation_type_declaration" {
		return false, false
	}
	for _, member := range nodeutil.NamedChildrenOf(scope.Class.DeclarationNode.ChildByFieldName("body")) {
		if member.Type() == "annotation_type_element_declaration" {
			return false, false
		}
	}
	file := findFileScopeForClassScope(scope)
	if file == nil {
		return false, false
	}
	for _, annotation := range declarationAnnotations(scope) {
		nameNode := annotation.ChildByFieldName("name")
		if nameNode == nil {
			continue
		}
		name := nameNode.Content(file.Source)
		if annotationNameIs(name, "java.lang.annotation.Inherited", scope) {
			inherited = true
		}
		if annotationNameIs(name, "java.lang.annotation.Retention", scope) {
			arguments := annotation.ChildByFieldName("arguments")
			if arguments != nil {
				value := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(arguments.Content(file.Source), "("), ")"))
				runtime = value == "RUNTIME" || strings.HasSuffix(value, ".RUNTIME")
			}
		}
	}
	return runtime, inherited
}
