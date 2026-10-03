package transpiler

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

const generatedObjectCloneMethod = "Java2goCloneSubobject"

func externalCloneableType(javaType string, ctx Ctx) bool {
	return qualifyDeclaredReferenceType(symbol.JavaType{Original: javaType}, ctx) == "java.lang.Cloneable"
}
func sourceDirectCloneable(scope *symbol.ClassScope, ctx Ctx) bool {
	if scope == nil {
		return false
	}
	declaring := classHeaderTypeCtx(scope, ctx)
	for _, implemented := range scope.ImplementedInterfaces {
		if externalCloneableType(implemented, declaring) {
			return true
		}
	}
	return false
}

func arrayCloneResultType(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	name := node.ChildByFieldName("name")
	object := node.ChildByFieldName("object")
	arguments := node.ChildByFieldName("arguments")
	if name == nil || name.Content(source) != "clone" || object == nil || (arguments != nil && arguments.NamedChildCount() != 0) {
		return "", false
	}
	javaType, known := inferExprJavaType(object, ctx, source)
	if !known {
		return "", false
	}
	_, rank := javaArrayTypeParts(javaType)
	return javaType, rank > 0
}

// Resolve the selected zero-argument declaration, including source ancestors
// that inherit Object.clone and overloads that do not override its signature.
func sourceSuperSelectsObjectClone(scope *symbol.ClassScope, ctx Ctx) bool {
	seen := map[*symbol.ClassScope]bool{}
	for current := scope; current != nil && !seen[current]; {
		seen[current] = true
		// Enum's implicit superclass is java.lang.Enum, whose final clone
		// always throws even when the declaration implements Cloneable.
		// Its omitted source superclass must never select Object's body.
		if current.IsEnum {
			return false
		}
		declaring := classHeaderTypeCtx(current, ctx)
		parent := resolveSuperclassScopeInDeclaringContext(declaring, current)
		if parent == nil {
			written := strings.TrimSpace(current.Superclass)
			return written == "" || qualifyDeclaredReferenceType(symbol.JavaType{Original: written}, declaring) == "java.lang.Object"
		}
		for _, method := range parent.Methods {
			if method != nil && method.OriginalName == "clone" && !method.IsStatic && len(method.Parameters) == 0 {
				return false
			}
		}
		current = parent
	}
	return false
}

// Select Enum's final platform body by declaration ancestry. A source class
// named Enum and a selected source clone override retain ordinary dispatch.
func sourceSuperSelectsEnumClone(scope *symbol.ClassScope, ctx Ctx) bool {
	seen := map[*symbol.ClassScope]bool{}
	for current := scope; current != nil && !seen[current]; {
		seen[current] = true
		if current.IsEnum {
			return true
		}
		parent := resolveSuperclassScopeInDeclaringContext(classHeaderTypeCtx(current, ctx), current)
		if parent == nil {
			return false
		}
		for _, method := range parent.Methods {
			if method != nil && method.OriginalName == "clone" && !method.IsStatic && len(method.Parameters) == 0 {
				return false
			}
		}
		current = parent
	}
	return false
}

func objectCloneInvocation(object *sitter.Node, method string, arguments *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	if method != "clone" || object == nil || (arguments != nil && arguments.NamedChildCount() != 0) {
		return nil
	}
	if object.Type() == "super" && ctx.currentClass != nil {
		if sourceSuperSelectsEnumClone(ctx.currentClass, ctx) {
			return stdjavaCall(ctx, "EnumCloneExecution", intrinsicExecutionExpr(ctx))
		}
		if sourceSuperSelectsObjectClone(ctx.currentClass, ctx) {
			return stdjavaCall(ctx, "ObjectCloneExecution", intrinsicExecutionExpr(ctx), ast.NewIdent(ShortName(ctx.className)))
		}
		return nil // A source super.clone override retains ordinary source dispatch.
	}
	if javaType, ok := inferExprJavaType(object, ctx, source); ok {
		_, rank := javaArrayTypeParts(javaType)
		if rank > 0 {
			return stdjavaGenericCall(ctx, "ArrayCloneExecution", []ast.Expr{javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx)}, []ast.Expr{intrinsicExecutionExpr(ctx), ParseExpr(object, source, ctx)})
		}
	}
	return nil
}

// Each source declaration copies only its own Java fields. Ancestor copiers
// execute in their declaring package, so private and final state remains shallow
// without using exported accessors or running constructors/initializers.
func generateObjectCloneDecls(ctx Ctx) []ast.Decl {
	scope := ctx.currentClass
	if scope == nil || scope.Class == nil || scope.IsInterface || scope.IsEnum || !sourceInheritsObjectTextDefault(scope, ctx) {
		return nil
	}
	receiver := ShortName(scope.Class.Name)
	used := map[string]struct{}{receiver: {}}
	for _, name := range scope.GoTypeParameterNames() {
		used[name] = struct{}{}
	}
	cloneName := synchronizedUniqueLocalName("__java2goClone", used)
	selfName := synchronizedUniqueLocalName("__java2goCloneSelf", used)
	typ := classSubobjectPointerTypeExpr(scope, scope.GoTypeParameterNames(), scope, ctx)
	body := []ast.Stmt{
		&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(cloneName)}, Tok: token.DEFINE, Rhs: []ast.Expr{&ast.CallExpr{Fun: ast.NewIdent("new"), Args: []ast.Expr{typ.(*ast.StarExpr).X}}}},
		&ast.IfStmt{Cond: &ast.BinaryExpr{X: ast.NewIdent(selfName), Op: token.EQL, Y: ast.NewIdent("nil")}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(selfName)}, Tok: token.ASSIGN, Rhs: []ast.Expr{ast.NewIdent(cloneName)}}}}},
	}
	selector := func(recv, name string) ast.Expr {
		return &ast.SelectorExpr{X: ast.NewIdent(recv), Sel: ast.NewIdent(name)}
	}
	for _, field := range scope.Fields {
		if field == nil || field.IsStatic {
			continue
		}
		value := selector(receiver, field.Name)
		if volatileFieldDefinition(field) {
			value = volatileFieldLoad(value, sourceReflectionFieldStorageType(scope, field, ctx), ctx, field, scope)
		}
		body = append(body, volatileFieldStoreStmt(field, selector(cloneName, field.Name), value, ctx))
	}
	if enclosingInstanceType(scope) != nil {
		name := scope.EnclosingFieldName()
		body = append(body, &ast.AssignStmt{Lhs: []ast.Expr{selector(cloneName, name)}, Tok: token.ASSIGN, Rhs: []ast.Expr{selector(receiver, name)}})
	}
	if paths := classSubobjectAncestorPaths(scope, ctx); len(paths) > 0 {
		parent := paths[0]
		field := parent.selectors[0]
		copied := &ast.CallExpr{Fun: &ast.SelectorExpr{X: selector(receiver, field), Sel: ast.NewIdent(generatedObjectCloneMethod)}, Args: []ast.Expr{ast.NewIdent(selfName)}}
		body = append(body, &ast.AssignStmt{Lhs: []ast.Expr{selector(cloneName, field)}, Tok: token.ASSIGN, Rhs: []ast.Expr{&ast.TypeAssertExpr{X: copied, Type: classSubobjectPointerTypeWithGoArguments(parent.scope, parent.goTypeArguments, ctx)}}})
	}
	if classNeedsReferenceObjectInfo(scope, ctx) && sourceHierarchyRoot(scope, ctx) {
		body = append(body, &ast.AssignStmt{Lhs: []ast.Expr{selector(cloneName, "ObjectInfo")}, Tok: token.ASSIGN, Rhs: []ast.Expr{stdjavaCall(ctx, "NewGeneratedObjectInfo", ast.NewIdent(selfName))}})
	}
	if setter := classSelfSetterCallStmtWithValue(ctx, cloneName, ast.NewIdent(selfName)); setter != nil {
		body = append(body, setter)
	}
	body = append(body, &ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent(cloneName)}})
	return []ast.Decl{&ast.FuncDecl{Name: ast.NewIdent(generatedObjectCloneMethod), Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent(receiver)}, Type: typ}}}, Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent(selfName)}, Type: &ast.InterfaceType{Methods: &ast.FieldList{}}}}}, Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.InterfaceType{Methods: &ast.FieldList{}}}}}}, Body: &ast.BlockStmt{List: body}}}
}
