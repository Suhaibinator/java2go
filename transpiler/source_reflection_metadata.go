package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
)

// Reflective flags come from the original Java declaration. Exported Go names
// encode ABI visibility and do not distinguish public from protected members.
func sourceReflectionModifiers(def *symbol.Definition, scope *symbol.ClassScope) int32 {
	var result int32
	if def != nil && def.DeclarationNode != nil {
		bits := map[string]int32{"public": 1, "private": 2, "protected": 4, "static": 8, "final": 16, "synchronized": 32, "volatile": 64, "transient": 128, "native": 256, "abstract": 1024, "strictfp": 2048}
		for _, child := range nodeutil.NamedChildrenOf(def.DeclarationNode) {
			if child.Type() == "modifiers" {
				for _, modifier := range nodeutil.UnnamedChildrenOf(child) {
					result |= bits[modifier.Type()]
				}
			}
		}
		if def.IsStatic {
			result |= 8
		}
		if def.IsFinal {
			result |= 16
		}
	}
	if def == scope.Class {
		if scope.IsInterface {
			result |= 512 | 1024
		}
		if scope.IsAbstract {
			result |= 1024
		}
		if scope.IsEnum {
			result |= 16384
		}
		if def != nil && def.DeclarationNode != nil && def.DeclarationNode.Type() == "annotation_type_declaration" {
			result |= 8192
		}
	} else if scope.IsInterface {
		result |= 1
		if def != nil && !def.Constructor && !def.IsStatic {
			result |= 1024
		}
	}
	return result
}
func reflectionInteger(n int32) ast.Expr {
	return &ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(int64(n), 10)}
}

func sourceReflectionFieldAccessorName(scope *symbol.ClassScope, field *symbol.Definition, set bool) string {
	operation := "Get"
	if set {
		operation = "Set"
	}
	// Public, uniquely allocated selectors let the runtime project the declaring
	// view and invoke its storage accessor without unsafe or Generic[any] casts.
	return symbol.GoIdentifier(collisionSafeExecutionIdentifier("Java2goReflectField"+operation+field.Name+"Java2goExecution", scope))
}
func sourceReflectionFieldStorageType(scope *symbol.ClassScope, field *symbol.Definition, ctx Ctx) ast.Expr {
	ctx = classScopeCtx(scope, ctx)
	value := abstractClassToInterface(javaTypeStringToGoTypeExpr(field.OriginalType, scope.TypeParameterNames(), ctx), field.OriginalType, ctx)
	return directOwnerTypeParameterFieldStorageType(scope, field, value, ctx)
}

// Captured storage fields reuse the original outer/local declaration nodes.
// Only declarations inside this Java class belong to getDeclaredFields.
func sourceReflectionDeclaredField(scope *symbol.ClassScope, field *symbol.Definition) bool {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil || field == nil || field.DeclarationNode == nil {
		return false
	}
	declaration := field.DeclarationNode
	if declaration.Type() != "field_declaration" && declaration.Type() != "constant_declaration" {
		return false
	}
	owner := scope.Class.DeclarationNode
	return declaration.StartByte() >= owner.StartByte() && declaration.EndByte() <= owner.EndByte()
}

func sourceReflectionAnonymousClass(scope *symbol.ClassScope) bool {
	return scope != nil && scope.Class != nil && scope.Class.DeclarationNode != nil && scope.Class.DeclarationNode.Type() == "object_creation_expression"
}

// A generated anonymous ABI can carry enclosing variables without declaring
// them. Reflection resolves those captured identities through Enclosing, rather
// than assigning them to the anonymous class itself.
func sourceReflectionJavaDeclarationScope(scope *symbol.ClassScope) *symbol.ClassScope {
	if !sourceReflectionAnonymousClass(scope) {
		return scope
	}
	copy := *scope
	copy.DeclaredTypeParameters = []symbol.TypeParam{}
	return &copy
}

func sourceReflectionFieldAccessors(scope *symbol.ClassScope, ctx Ctx) []ast.Decl {
	if !sourceUsesReflection() || scope == nil || scope.IsInterface {
		return nil
	}
	var out []ast.Decl
	for _, field := range scope.Fields {
		if !sourceReflectionDeclaredField(scope, field) || field.IsStatic {
			continue
		}
		storage := sourceReflectionFieldStorageType(scope, field, ctx)
		receiver := ast.NewIdent("receiver")
		getter := &ast.FuncDecl{Name: ast.NewIdent(sourceReflectionFieldAccessorName(scope, field, false)), Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{receiver}, Type: classSubobjectPointerTypeExpr(scope, scope.GoTypeParameterNames(), scope, ctx)}}}, Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}, Results: &ast.FieldList{List: []*ast.Field{{Type: storage}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(symbol.GoIdentifier(field.Name))}}}}}}
		setter := &ast.FuncDecl{Name: ast.NewIdent(sourceReflectionFieldAccessorName(scope, field, true)), Recv: getter.Recv, Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx), {Names: []*ast.Ident{ast.NewIdent("value")}, Type: storage}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(symbol.GoIdentifier(field.Name))}}, Tok: token.ASSIGN, Rhs: []ast.Expr{ast.NewIdent("value")}}}}}
		if volatileFieldDefinition(field) {
			backing := &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(symbol.GoIdentifier(field.Name))}
			getter.Body.List = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{volatileFieldLoad(backing, storage, classScopeCtx(scope, ctx), field, scope)}}}
			setter.Body.List = []ast.Stmt{volatileFieldStoreStmt(field, backing, ast.NewIdent("value"), classScopeCtx(scope, ctx))}
		}
		out = append(out, getter, setter)
	}
	return out
}
func sourceReflectionGeneratedFieldCallback(scope *symbol.ClassScope, field *symbol.Definition, ctx Ctx, set bool) ast.Expr {
	params := []*ast.Field{executionParameterField("execution", ctx), {Names: []*ast.Ident{ast.NewIdent("receiver")}, Type: ast.NewIdent("any")}}
	args := []ast.Expr{ast.NewIdent("execution"), ast.NewIdent("receiver"), javaTypeIDLiteral(sourceClassRuntimeTypeID(scope, ctx), ctx), metadataString(sourceReflectionFieldAccessorName(scope, field, set))}
	service := "ReflectGeneratedFieldGetExecution"
	var results *ast.FieldList
	if set {
		service = "ReflectGeneratedFieldSetExecution"
		params = append(params, &ast.Field{Names: []*ast.Ident{ast.NewIdent("value")}, Type: ast.NewIdent("any")})
		args = append(args, ast.NewIdent("value"))
	} else {
		results = &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}
	}
	call := stdjavaCall(ctx, service, args...)
	var body ast.Stmt = &ast.ExprStmt{X: call}
	if !set {
		body = &ast.ReturnStmt{Results: []ast.Expr{call}}
	}
	return &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{List: params}, Results: results}, Body: &ast.BlockStmt{List: []ast.Stmt{body}}}
}

// Unknown finite trees must fail explicitly at the runtime Type boundary. Nil
// preserves the legacy erased fallback and would falsely describe a supported
// parameterized or variable type. ABI4 reserves no named unsupported kind;
// its closed resolver rejects every unknown kind with UnsupportedOperationException.
func sourceReflectionUnsupportedType(ctx Ctx) ast.Expr {
	return &ast.CompositeLit{Type: stdjavaQualifiedExpr("ReflectTypeDescriptor", ctx), Elts: []ast.Expr{
		metadataKey("Kind", &ast.CallExpr{Fun: stdjavaQualifiedExpr("ReflectTypeKind", ctx), Args: []ast.Expr{reflectionInteger(255)}}),
	}}
}

// Fields remain in the parser's source/declarator order; inherited fields belong
// to their own ClassDescriptor and are not flattened into this class.
func sourceReflectionFields(scope *symbol.ClassScope, ctx Ctx) []ast.Expr {
	return sourceReflectionFieldsForFields(scope, scope.Fields, ctx)
}
func sourceReflectionFieldsForFields(scope *symbol.ClassScope, fields []*symbol.Definition, ctx Ctx) []ast.Expr {
	declaring := classScopeCtx(sourceReflectionJavaDeclarationScope(scope), ctx)
	var out []ast.Expr
	for _, field := range fields {
		if !sourceReflectionDeclaredField(scope, field) {
			continue
		}
		fieldCtx := declaring.Clone()
		fieldCtx.localScope = field
		erased, ok := javaTypeDescriptorExpr(qualifyDeclaredReferenceType(symbol.JavaType{Original: field.OriginalType, TypeParameterBindings: field.TypeParameterBindings}, fieldCtx), fieldCtx)
		if !ok {
			continue
		}
		values := []ast.Expr{metadataKey("Name", metadataString(field.OriginalName)), metadataKey("GoName", metadataGoName(field.Name)), metadataKey("Type", erased), metadataKey("Final", ast.NewIdent(strconv.FormatBool(field.IsFinal))), metadataKey("NonPublic", ast.NewIdent(strconv.FormatBool(sourceReflectionModifiers(field, scope)&1 == 0))), metadataKey("Modifiers", reflectionInteger(sourceReflectionModifiers(field, scope))), metadataKey("HasModifiers", ast.NewIdent("true")), metadataKey("Annotations", sourceAnnotationDescriptors(field.DeclarationNode, scope, ctx))}
		generic, supported := reflectiveTypeDescriptorExpr(symbol.JavaType{Original: field.OriginalType, TypeParameterBindings: field.TypeParameterBindings}, fieldCtx)
		if !supported {
			generic = sourceReflectionUnsupportedType(ctx)
		}
		values = append(values, metadataKey("GenericType", &ast.UnaryExpr{Op: token.AND, X: generic}))
		if volatileFieldDefinition(field) {
			values = append(values, metadataKey("VolatileCell", volatileFieldDescriptorCallback(scope, field, ctx)))
		}
		if field.IsStatic {
			backing := ast.NewIdent(symbol.GoIdentifier(field.Name))
			value := ast.Expr(backing)
			if volatileFieldDefinition(field) {
				value = volatileFieldLoad(backing, sourceReflectionFieldStorageType(scope, field, ctx), classScopeCtx(scope, ctx), field, scope)
			}
			values = append(values, metadataKey("StaticGet", &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{value}}}}}))
			if !field.IsFinal {
				storage := sourceReflectionFieldStorageType(scope, field, ctx)

				convert := &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{ast.NewIdent("converted")}, Type: storage}}}}
				assign := &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("converted")}, Tok: token.ASSIGN, Rhs: []ast.Expr{&ast.TypeAssertExpr{X: ast.NewIdent("value"), Type: storage}}}
				callback := &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx), {Names: []*ast.Ident{ast.NewIdent("value")}, Type: ast.NewIdent("any")}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{convert, &ast.IfStmt{Cond: &ast.BinaryExpr{X: ast.NewIdent("value"), Op: token.NEQ, Y: ast.NewIdent("nil")}, Body: &ast.BlockStmt{List: []ast.Stmt{assign}}}, volatileFieldStoreStmt(field, backing, ast.NewIdent("converted"), classScopeCtx(scope, ctx))}}}
				values = append(values, metadataKey("StaticSet", callback))
			}
		} else if !scope.IsInterface {
			values = append(values, metadataKey("Get", sourceReflectionGeneratedFieldCallback(scope, field, ctx, false)), metadataKey("Set", sourceReflectionGeneratedFieldCallback(scope, field, ctx, true)))
		}
		out = append(out, &ast.CompositeLit{Elts: values})
	}
	return out
}
func sourceReflectionConstructorDescriptors(scope *symbol.ClassScope, ctx Ctx) ast.Expr {
	out := &ast.CompositeLit{Type: &ast.ArrayType{Elt: stdjavaQualifiedExpr("ConstructorDescriptor", ctx)}}
	// Extra enclosing/captured/generic ABI arguments cannot be invented by a
	// no-argument reflective constructor. Those shapes await descriptor planning.
	if scope.IsInterface || scope.IsEnum || scope.IsInner || len(scope.TypeParameters) != 0 || sourceReflectionAnonymousClass(scope) {
		return out
	}
	var noarg *symbol.Definition
	for _, method := range scope.Methods {
		if method.Constructor && len(method.Parameters) == 0 {
			noarg = method
			break
		}
	}
	name := noArgConstructorName(scope)
	if name == "" {
		return out
	}
	modifiers := sourceReflectionModifiers(scope.Class, scope) & 7
	if noarg != nil {
		modifiers = sourceReflectionModifiers(noarg, scope)
	}
	entry := &ast.CompositeLit{Elts: []ast.Expr{metadataKey("Modifiers", reflectionInteger(modifiers))}}
	if !scope.IsAbstract {
		call := &ast.CallExpr{Fun: ast.NewIdent(executionConstructorImplementationName(name, scope)), Args: []ast.Expr{ast.NewIdent("execution")}}
		callback := &ast.FuncLit{
			Type: &ast.FuncType{
				Params: &ast.FieldList{List: []*ast.Field{
					executionParameterField("execution", ctx),
					{Names: []*ast.Ident{ast.NewIdent("arguments")}, Type: &ast.ArrayType{Elt: ast.NewIdent("any")}},
				}},
				Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}},
			},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}},
		}
		entry.Elts = append(entry.Elts, metadataKey("Construct", callback))
	}
	out.Elts = append(out.Elts, entry)
	return out
}

func extendSourceReflectionMetadata(descriptor *ast.CompositeLit, scope *symbol.ClassScope, ctx Ctx) {
	descriptor.Elts = append(descriptor.Elts, metadataKey("NestHost", sourceReflectionNestHostTypeID(scope, ctx)))
	descriptor.Elts = append(descriptor.Elts, metadataKey("Modifiers", reflectionInteger(sourceReflectionModifiers(scope.Class, scope))), metadataKey("HasModifiers", ast.NewIdent("true")), metadataKey("Constructors", sourceReflectionConstructorDescriptors(scope, ctx)))
	// Anonymous classes declare no Java type variables, even when their Go ABI
	// carries captured enclosing binders. Preserve the original scope for all
	// storage/header planning and change only the reflection declaration view.
	variableScope := sourceReflectionJavaDeclarationScope(scope)
	declarationCtx := classScopeCtx(variableScope, ctx)
	variables, ok := reflectiveTypeVariableDescriptorsExpr(variableScope, declarationCtx)
	if !ok {
		// Preserve declaration arity and source names while refusing an unknown
		// bound. The unavailable bound must not silently become Object.
		entries := &ast.CompositeLit{Type: &ast.ArrayType{Elt: stdjavaQualifiedExpr("TypeVariableDescriptor", ctx)}}
		for _, parameter := range variableScope.OwnTypeParameters() {
			name := parameter.Name
			if parameter.Declaration != nil && parameter.Declaration.SourceName != "" {
				name = parameter.Declaration.SourceName
			}
			bounds := &ast.CompositeLit{Type: &ast.ArrayType{Elt: stdjavaQualifiedExpr("ReflectTypeDescriptor", ctx)}, Elts: []ast.Expr{sourceReflectionUnsupportedType(ctx)}}
			entries.Elts = append(entries.Elts, &ast.CompositeLit{Elts: []ast.Expr{metadataKey("Name", metadataString(name)), metadataKey("Bounds", bounds)}})
		}
		variables = entries
	}
	descriptor.Elts = append(descriptor.Elts, metadataKey("TypeParameters", variables))
	superclass, ok := reflectiveGenericSuperclassDescriptorExpr(variableScope, declarationCtx)
	if !ok {
		superclass = &ast.UnaryExpr{Op: token.AND, X: sourceReflectionUnsupportedType(ctx)}
	}
	descriptor.Elts = append(descriptor.Elts, metadataKey("GenericSuperclass", superclass))
}
