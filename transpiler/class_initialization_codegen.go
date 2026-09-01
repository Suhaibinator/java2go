package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func classInitializationStateName(scope *symbol.ClassScope) string {
	if scope == nil || scope.Class == nil {
		return ""
	}
	return collisionSafeExecutionIdentifier("__java2goClassInitialization"+scope.Class.Name, scope)
}

func classInitializationEnsureName(scope *symbol.ClassScope) string {
	if scope == nil || scope.Class == nil {
		return ""
	}
	return collisionSafeExecutionIdentifier(scope.Class.Name+"Java2goEnsureInitialized", scope)
}

func javaClassBinaryName(scope *symbol.ClassScope) string {
	if scope == nil || scope.Class == nil {
		return ""
	}
	parts := []string{scope.Class.OriginalName}
	for enclosing := scope.Enclosing; enclosing != nil && enclosing.Class != nil; enclosing = enclosing.Enclosing {
		parts = append([]string{enclosing.Class.OriginalName}, parts...)
	}
	name := strings.Join(parts, "$")
	if file := findFileScopeForClassScope(scope); file != nil && strings.TrimSpace(file.Package) != "" {
		name = file.Package + "." + name
	}
	return name
}

func classScopeHasInitializationWork(scope *symbol.ClassScope) bool {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil {
		return false
	}
	body := scope.Class.DeclarationNode.ChildByFieldName("body")
	var source []byte
	if file := findFileScopeForClassScope(scope); file != nil {
		source = file.Source
	}
	return classBodyNeedsOrderedStaticInitialization(body, scope, source)
}

// classInitializationNeedsCoordinator includes otherwise-empty subclasses whose
// initialization transitively initializes a superclass. Each class needs its
// own erroneous/initialized state: if Base initialization fails while Sub is
// being initialized, later uses of Sub and Base must report their respective
// class identities rather than sharing Base's coordinator.
func classInitializationNeedsCoordinator(scope *symbol.ClassScope, ctx Ctx) bool {
	seen := map[*symbol.ClassScope]struct{}{}
	for current := scope; current != nil; current = resolveSuperclassScopeInDeclaringContext(ctx, current) {
		if _, duplicate := seen[current]; duplicate {
			return false
		}
		seen[current] = struct{}{}
		if classScopeHasInitializationWork(current) {
			return true
		}
		if current.IsInterface {
			return false
		}
	}
	return false
}

// classInitializationTarget returns the class whose lazy coordinator must run
// for an active use of scope. A class without its own initialization statements
// still owns a coordinator when a superclass has work, because Java records
// success/failure independently for every class in that chain.
func classInitializationTarget(scope *symbol.ClassScope, ctx Ctx) *symbol.ClassScope {
	if classInitializationNeedsCoordinator(scope, ctx) {
		return scope
	}
	return nil
}

func classInitializationEnsureCall(scope *symbol.ClassScope, execution ast.Expr, ctx Ctx) ast.Expr {
	target := classInitializationTarget(scope, ctx)
	if target == nil || execution == nil {
		return nil
	}
	return &ast.CallExpr{
		Fun: qualifiedNameExpr(
			classInitializationEnsureName(target),
			findJavaPackageForClassScope(target),
			ctx,
		),
		Args: []ast.Expr{execution},
	}
}

func classInitializationEnsureStmt(scope *symbol.ClassScope, execution ast.Expr, ctx Ctx) ast.Stmt {
	call := classInitializationEnsureCall(scope, execution, ctx)
	if call == nil {
		return nil
	}
	return &ast.ExprStmt{X: call}
}

// guardClassInitializationBeforeExpr evaluates the active-use initialization
// before evaluating value itself. This ordering is required for direct class
// instance creation: the JVM initializes C before it evaluates the arguments in
// `new C(arguments)`. Calls through a constructor reference do not use this
// helper because their SAM arguments have already been evaluated; the hidden
// constructor implementation's entry guard handles that boundary instead.
func guardClassInitializationBeforeExpr(scope *symbol.ClassScope, value, resultType ast.Expr, ctx Ctx) ast.Expr {
	execution := executionExpr(ctx)
	ensure := classInitializationEnsureStmt(scope, execution, ctx)
	if ensure == nil || value == nil || resultType == nil {
		return value
	}
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: resultType}}}},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			ensure,
			&ast.ReturnStmt{Results: []ast.Expr{value}},
		}},
	}}
}

func orderedStaticInitializationStatements(body *sitter.Node, source []byte, ctx Ctx, executionName string) []ast.Stmt {
	var statements []ast.Stmt
	for _, child := range nodeutil.NamedChildrenOf(body) {
		switch child.Type() {
		case "field_declaration":
			for _, declarator := range nodeutil.VariableDeclarators(child) {
				valueNode := declarator.ChildByFieldName("value")
				nameNode := declarator.ChildByFieldName("name")
				if valueNode == nil || nameNode == nil {
					continue
				}
				fieldDefinitions := ctx.currentClass.FindField().ByOriginalName(nameNode.Content(source))
				if len(fieldDefinitions) == 0 {
					continue
				}
				fieldDefinition := fieldDefinitions[0]
				if !fieldDefinition.IsStatic {
					continue
				}
				if fieldDefinition.IsCompileTimeConstant {
					continue
				}
				valueCtx := ctx.Clone()
				valueCtx.localScope = &symbol.Definition{IsStatic: true}
				valueCtx.executionContextName = executionName
				valueCtx.expectedType = fieldDefinition.OriginalType
				valueCtx.expectedTypeRoot = valueNode
				value := ParseExpr(valueNode, source, valueCtx)
				value = coerceArgumentToExpectedType(value, valueNode, fieldDefinition.OriginalType, valueCtx, source)
				statements = append(statements, &ast.AssignStmt{
					Lhs: []ast.Expr{&ast.Ident{Name: fieldDefinition.Name}},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{value},
				})
			}
		case "static_initializer":
			staticCtx := ctx.Clone()
			staticCtx.localScope = &symbol.Definition{IsStatic: true}
			staticCtx.executionContextName = executionName
			block, ok := ParseStmt(child.NamedChild(0), source, staticCtx).(*ast.BlockStmt)
			if ok && block != nil {
				statements = append(statements, block.List...)
			}
		}
	}
	return statements
}

func buildLazyClassInitializationDecls(body *sitter.Node, source []byte, ctx Ctx) []ast.Decl {
	if ctx.currentClass == nil || !classInitializationNeedsCoordinator(ctx.currentClass, ctx) {
		return nil
	}

	executionName := executionParameterName(body, source, ctx)
	statements := orderedStaticInitializationStatements(body, source, ctx, executionName)
	stateName := classInitializationStateName(ctx.currentClass)
	ensureName := classInitializationEnsureName(ctx.currentClass)
	stateDeclaration := &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names: []*ast.Ident{{Name: stateName}},
			Values: []ast.Expr{stdjavaCall(ctx, "NewClassInitialization", &ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(javaClassBinaryName(ctx.currentClass)),
			})},
		}},
	}

	initializerBody := []ast.Stmt{&ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{&ast.Ident{Name: executionName}},
	}}
	if parent := resolveSuperclassScopeInDeclaringContext(ctx, ctx.currentClass); parent != nil {
		if ensure := classInitializationEnsureStmt(parent, &ast.Ident{Name: executionName}, ctx); ensure != nil {
			initializerBody = append(initializerBody, ensure)
		}
	}
	initializerBody = append(initializerBody, statements...)

	ensureDeclaration := &ast.FuncDecl{
		Name: &ast.Ident{Name: ensureName},
		Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{
			executionParameterField(executionName, ctx),
		}}},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: stateName},
				Sel: &ast.Ident{Name: "Ensure"},
			},
			Args: []ast.Expr{
				&ast.Ident{Name: executionName},
				&ast.FuncLit{
					Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{
						executionParameterField(executionName, ctx),
					}}},
					Body: &ast.BlockStmt{List: initializerBody},
				},
			},
		}}}},
	}
	return []ast.Decl{stateDeclaration, ensureDeclaration}
}
