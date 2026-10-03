package transpiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// A Go anonymous field occupies the same selector namespace as promoted Java
// methods and fields. Rename the storage edge, rather than the Java member, by
// embedding an alias of the exact parent instantiation only when necessary.
func superclassEmbeddingNeedsAlias(scope *symbol.ClassScope, ctx Ctx) bool {
	parent := resolveSuperclassScopeInDeclaringContext(ctx, scope)
	return parent != nil && parent.Class != nil && !parent.IsInterface &&
		superclassEmbeddingMemberExists(parent.Class.Name, scope, ctx)
}

// This checks physical selectors without asking bridge planners to resolve
// signatures: their covariance paths themselves consume superclass selectors.
// Execution and exact bridge names use the same collision allocator directly.
func superclassEmbeddingMemberExists(name string, scope *symbol.ClassScope, ctx Ctx) bool {
	if name == "" || scope == nil {
		return false
	}
	if referenceIdentityReservedSelector(name) || name == fieldInitMethodName {
		return true
	}
	queue := []*symbol.ClassScope{scope}
	seen := make(map[*symbol.ClassScope]bool)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == nil || seen[current] {
			continue
		}
		seen[current] = true
		for _, field := range current.Fields {
			if field != nil && !field.IsStatic && field.Name == name {
				return true
			}
		}
		for _, method := range current.Methods {
			if method == nil || method.Constructor || method.IsStatic {
				continue
			}
			if method.Name == name || method.HelperName == name ||
				(strings.HasPrefix(name, method.Name+executionMethodSuffix) && collisionSafeExecutionIdentifier(method.Name+executionMethodSuffix, current) == name) ||
				(strings.HasPrefix(name, method.Name+"Java2goExactExecution") && collisionSafeExecutionIdentifier(method.Name+"Java2goExactExecution", current) == name) {
				return true
			}
		}
		if current.EnclosingFieldName() == name || classDispatchFieldName(current) == name ||
			classSelfSetterName(current) == name ||
			(strings.HasPrefix(name, "Java2goInstall") && classSubobjectInstallerName(current) == name) {
			return true
		}
		for _, view := range current.AffineArrayViews {
			if view != nil && view.HelperName == name {
				return true
			}
		}
		interfaces := resolveImplementedInterfaceScopesInDeclaringContext(ctx, current)
		for _, iface := range interfaces {
			if iface != nil && iface.Class != nil &&
				(iface.Class.Name == name || interfaceDefaultCarrierName(iface) == name) {
				return true
			}
		}
		queue = append(queue, interfaces...)
		queue = append(queue, resolveSuperclassScopeInDeclaringContext(ctx, current))
	}
	return false
}

// Alias names are exported because another generated package can project a
// child expression to this parent subobject. The binary child identity makes
// edge names distinct across packages, nesting, and hoisted synthetic classes.
// A nonhex delimiter separates numeric collision suffixes from the hex identity.
func superclassEmbeddingAliasName(scope *symbol.ClassScope, ctx Ctx) string {
	identity := javaClassBinaryName(scope)
	if findFileScopeForClassScope(scope) == nil && scope != nil && scope.Class != nil {
		// Two methods can declare different local classes with the same Java
		// name. Their hoisted Go names carry the unique declaration identity.
		identity += "\x00" + scope.Class.Name
	}
	base := "Java2goSuperFor" + fmt.Sprintf("%X", []byte(identity))
	for suffix := 0; ; suffix++ {
		candidate := base
		if suffix > 0 {
			candidate += "_" + strconv.Itoa(suffix)
		}
		// Import aliases can be allocated later in another consuming file.
		// Reserve their complete spelling family using the stable source graph,
		// rather than the caller's mutable importAliases map.
		for superclassEmbeddingImportIdentifierExists(candidate) {
			candidate += "_"
		}
		if !generatedIdentifierExists(candidate, scope) &&
			!superclassEmbeddingGeneratedIdentifierExists(candidate, scope) &&
			!superclassEmbeddingMemberExists(candidate, scope, ctx) &&
			!superclassEmbeddingSourceIdentifierExists(candidate, ctx) {
			return candidate
		}
	}
}

// Global Definitions cover ordinary declarations. Source identifier scanning
// additionally covers local/anonymous declarations before they are hoisted,
// helper spellings derived from those methods, and possible import aliases.
func superclassEmbeddingSourceIdentifierExists(name string, ctx Ctx) bool {
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			var visit func(*sitter.Node) bool
			visit = func(node *sitter.Node) bool {
				if node == nil {
					return false
				}
				if node.Type() == "identifier" || node.Type() == "type_identifier" {
					identifier := sanitizeGoIdent(node.Content(file.Source))
					if identifier == name || symbol.Uppercase(identifier) == name || symbol.Lowercase(identifier) == name {
						return true
					}
				}
				if node.Type() == "method_declaration" {
					method := synthAnonClassMethodDefinition(node, file.Source, false)
					if method != nil && (method.Name == name ||
						(strings.HasPrefix(name, method.Name+executionMethodSuffix) && collisionSafeExecutionIdentifier(method.Name+executionMethodSuffix, nil) == name) ||
						(strings.HasPrefix(name, method.Name+"Java2goExactExecution") && collisionSafeExecutionIdentifier(method.Name+"Java2goExactExecution", nil) == name)) {
						return true
					}
				}
				for _, child := range nodeutil.NamedChildrenOf(node) {
					if visit(child) {
						return true
					}
				}
				return false
			}
			for _, top := range file.TopLevelClasses {
				if top != nil && top.Class != nil && visit(top.Class.DeclarationNode) {
					return true
				}
			}
		}
	}
	return false
}

func superclassEmbeddingImportIdentifierExists(name string) bool {
	matches := func(javaPackage string) bool {
		base := packageAliasFromJavaPackage(javaPackage)
		if name == base || name == base+"pkg" {
			return true
		}
		if strings.HasPrefix(name, base) {
			suffix, err := strconv.Atoi(strings.TrimPrefix(name, base))
			return err == nil && suffix >= 2
		}
		return false
	}
	for packageName, pkg := range symbol.GlobalScope.Packages {
		if matches(packageName) {
			return true
		}
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			for _, imported := range file.Imports {
				if matches(imported) {
					return true
				}
			}
		}
	}
	return false
}

func superclassEmbeddingGeneratedIdentifierExists(name string, owner *symbol.ClassScope) bool {
	matches := func(scope *symbol.ClassScope) bool {
		if scope == nil || scope.Class == nil {
			return false
		}
		constructor := defaultConstructorName(scope.Class.Name)
		if name == constructor || name == constructorWithSelfName(constructor) ||
			name == collisionSafeExecutionIdentifier(constructor+executionMethodSuffix, scope) ||
			name == collisionSafeExecutionIdentifier(constructorWithSelfName(constructor)+executionMethodSuffix, scope) ||
			name == classInitializationStateName(scope) || name == classInitializationEnsureName(scope) {
			return true
		}
		for _, method := range scope.Methods {
			if method != nil && (method.HelperName == name ||
				(strings.HasPrefix(name, method.Name+executionMethodSuffix) && collisionSafeExecutionIdentifier(method.Name+executionMethodSuffix, scope) == name) ||
				(strings.HasPrefix(name, method.Name+"Java2goExactExecution") && collisionSafeExecutionIdentifier(method.Name+"Java2goExactExecution", scope) == name)) {
				return true
			}
		}
		return superclassEmbeddingMemberExists(name, scope, Ctx{})
	}
	if matches(owner) {
		return true
	}
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			for _, top := range file.TopLevelClasses {
				if visitClassScopes(top, matches) {
					return true
				}
			}
		}
	}
	return false
}

// superclassEmbeddedSelectorName always names the immediate edge belonging to
// scope. Every hierarchy walk must call it again for the next child edge.
func superclassEmbeddedSelectorName(scope *symbol.ClassScope, ctx Ctx) string {
	parent := resolveSuperclassScopeInDeclaringContext(ctx, scope)
	if parent == nil || parent.Class == nil {
		return ""
	}
	if superclassEmbeddingNeedsAlias(scope, ctx) {
		return superclassEmbeddingAliasName(scope, ctx)
	}
	return parent.Class.Name
}

func superclassEmbeddingTypeExpr(scope *symbol.ClassScope, javaType string, typeParams []string, ctx Ctx) ast.Expr {
	if !superclassEmbeddingNeedsAlias(scope, ctx) {
		return javaTypeStringToGoTypeExpr(javaType, typeParams, classHeaderTypeCtx(scope, ctx))
	}
	alias := instantiateGenericType(superclassEmbeddingAliasName(scope, ctx), typeParamExprs(scope.GoTypeParameterNames()))
	return &ast.StarExpr{X: alias}
}

func superclassEmbeddingAliasDecl(scope *symbol.ClassScope, ctx Ctx) ast.Decl {
	if !superclassEmbeddingNeedsAlias(scope, ctx) {
		return nil
	}
	declarationCtx := classHeaderTypeCtx(scope, ctx)
	parentType := javaTypeStringToGoTypeExpr(scope.Superclass, scope.TypeParameterNames(), declarationCtx)
	if pointer, ok := parentType.(*ast.StarExpr); ok {
		parentType = pointer.X
	}
	alias := &ast.TypeSpec{
		Name:   ast.NewIdent(superclassEmbeddingAliasName(scope, ctx)),
		Assign: token.Pos(1),
		Type:   parentType,
	}
	if len(scope.TypeParameters) > 0 {
		alias.TypeParams = &ast.FieldList{List: makeTypeParamFieldsInContext(scope.TypeParameters, declarationCtx)}
	}
	return &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{alias}}
}

// A superclass reference projection must retain null and evaluate its operand
// once. Using the same edge walker preserves both aliases and multilevel views.
func nullableSuperclassEdgeView(value ast.Expr, actualType, expectedType string, actual, expected *symbol.ClassScope, ctx Ctx) ast.Expr {
	parameter := ast.NewIdent("__java2goBaseValue")
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{parameter}, Type: javaTypeStringToGoTypeExpr(actualType, inScopeTypeParameters(ctx), ctx)}}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(expectedType, inScopeTypeParameters(ctx), ctx)}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.IfStmt{Cond: &ast.BinaryExpr{X: parameter, Op: token.EQL, Y: ast.NewIdent("nil")},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("nil")}}}}},
			&ast.ReturnStmt{Results: []ast.Expr{sourceClassViewExpr(actual, expected, parameter, ctx)}},
		}},
	}, Args: []ast.Expr{value}}
}
