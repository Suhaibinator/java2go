package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
)

// A canonical class emits an allocated public alias and a single physical
// struct. This certificate concerns that compiler declaration, not Java method
// selection or an inference from a pointer-shaped Go expression.
type canonicalGenericSignatureAlias struct {
	owner *symbol.ClassScope
	file  *symbol.FileScope
	name  string
	raw   string
	arity int
}

type canonicalGenericSignatureName struct {
	qualifier string
	name      string
}

// Only physical method/interface signatures need this normalization. Public
// aliases, constructor binders, storage selectors and bodies remain unchanged.
func canonicalGenericPhysicalSignature(signature ast.Expr, ctx Ctx) {
	switch signature.(type) {
	case *ast.FuncType, *ast.InterfaceType:
	default:
		return
	}
	if !canonicalGenericSignatureOwner(ctx.currentClass, ctx) || findFileScopeForClassScope(ctx.currentClass, ctx) != ctx.currentFile || canonicalGenericFamily(ctx.currentClass, ctx) == nil {
		return
	}
	blocked := map[string]bool{}
	for _, name := range ctx.currentClass.GoTypeParameterNames() {
		blocked[name] = true
	}
	// Most signatures contain no parameterized class pointer. Avoid inventory
	// work for their primitive/native-only descriptors.
	demand := map[canonicalGenericSignatureName]bool{}
	canonicalGenericSignatureWalk(signature, blocked, func(name canonicalGenericSignatureName, _ int) ast.Expr {
		demand[name] = true
		return nil
	})
	if len(demand) == 0 {
		return
	}
	aliases := map[canonicalGenericSignatureName]canonicalGenericSignatureAlias{}
	ambiguous := map[canonicalGenericSignatureName]bool{}
	for _, scope := range allSourceClassScopes() {
		if scope == nil || scope.Class == nil || scope.IsInterface || scope.IsEnum || len(scope.TypeParameters) == 0 {
			continue
		}
		file := findFileScopeForClassScope(scope, ctx)
		if file == nil {
			continue
		}
		name := canonicalGenericSignatureName{name: scope.Class.Name}
		if file.Package != ctx.currentFile.Package {
			name.qualifier = ctx.importAliases[file.Package]
			if name.qualifier == "" {
				continue
			}
		}
		if !demand[name] {
			continue
		}
		// A competing source declaration is authoritative even when it cannot
		// itself certify a canonical layout.
		if previous, exists := aliases[name]; exists && previous.owner != scope {
			ambiguous[name] = true
			continue
		}
		if !canonicalGenericSignatureOwner(scope, ctx) {
			ambiguous[name] = true
			continue
		}
		declaring := ctx.Clone()
		declaring.currentFile = nil
		declaring = classHeaderTypeCtx(scope, declaring)
		if declaring.currentFile != file || !canonicalGenericClass(scope, declaring) {
			ambiguous[name] = true
			continue
		}
		aliases[name] = canonicalGenericSignatureAlias{
			owner: scope, file: file, name: scope.Class.Name,
			raw: availableCanonicalGenericName(scope.Class.Name + "Java2goErased"), arity: len(scope.TypeParameters),
		}
	}
	canonicalGenericSignatureWalk(signature, blocked, func(name canonicalGenericSignatureName, arity int) ast.Expr {
		alias, exists := aliases[name]
		if !exists || ambiguous[name] || alias.arity != arity {
			return nil
		}
		if name.qualifier == "" {
			return ast.NewIdent(alias.raw)
		}
		// Preserve the already-used package binding. No import is allocated or
		// committed by a signature proof.
		return &ast.SelectorExpr{X: ast.NewIdent(name.qualifier), Sel: ast.NewIdent(alias.raw)}
	})
}

func canonicalGenericSignatureOwner(owner *symbol.ClassScope, ctx Ctx) bool {
	if owner == nil || owner.Class == nil || owner.Class.DeclarationNode == nil {
		return false
	}
	file := findFileScopeForClassScope(owner, ctx)
	if file == nil || file.ParsedRoot() == nil || ctx.currentFile == nil {
		return false
	}
	root := owner.Class.DeclarationNode
	for root.Parent() != nil {
		root = root.Parent()
	}
	return root.Equal(file.ParsedRoot())
}

// This walk accepts only type positions. A method's own binders shadow both
// unqualified aliases and possible package qualifiers in nested signatures.
func canonicalGenericSignatureWalk(expr ast.Expr, blocked map[string]bool, replace func(canonicalGenericSignatureName, int) ast.Expr) ast.Expr {
	if expr == nil {
		return nil
	}
	fields := func(list *ast.FieldList, bindings map[string]bool) {
		if list == nil {
			return
		}
		for _, field := range list.List {
			field.Type = canonicalGenericSignatureWalk(field.Type, bindings, replace)
		}
	}
	switch typ := expr.(type) {
	case *ast.FuncType:
		bindings := make(map[string]bool, len(blocked))
		for name := range blocked {
			bindings[name] = true
		}
		if typ.TypeParams != nil {
			for _, field := range typ.TypeParams.List {
				for _, name := range field.Names {
					bindings[name.Name] = true
				}
			}
		}
		fields(typ.Params, bindings)
		fields(typ.Results, bindings)
	case *ast.InterfaceType:
		fields(typ.Methods, blocked)
	case *ast.StarExpr:
		var base ast.Expr
		var arity int
		switch indexed := typ.X.(type) {
		case *ast.IndexExpr:
			base, arity = indexed.X, 1
		case *ast.IndexListExpr:
			base, arity = indexed.X, len(indexed.Indices)
		}
		var name canonicalGenericSignatureName
		switch base := base.(type) {
		case *ast.Ident:
			if !blocked[base.Name] {
				name.name = base.Name
			}
		case *ast.SelectorExpr:
			if qualifier, ok := base.X.(*ast.Ident); ok && !blocked[qualifier.Name] {
				name.qualifier, name.name = qualifier.Name, base.Sel.Name
			}
		}
		if name.name != "" {
			if physical := replace(name, arity); physical != nil {
				typ.X = physical
				return typ
			}
		}
		typ.X = canonicalGenericSignatureWalk(typ.X, blocked, replace)
	case *ast.IndexExpr:
		typ.Index = canonicalGenericSignatureWalk(typ.Index, blocked, replace)
	case *ast.IndexListExpr:
		for index := range typ.Indices {
			typ.Indices[index] = canonicalGenericSignatureWalk(typ.Indices[index], blocked, replace)
		}
	case *ast.ArrayType:
		typ.Elt = canonicalGenericSignatureWalk(typ.Elt, blocked, replace)
	case *ast.MapType:
		typ.Key = canonicalGenericSignatureWalk(typ.Key, blocked, replace)
		typ.Value = canonicalGenericSignatureWalk(typ.Value, blocked, replace)
	case *ast.ChanType:
		typ.Value = canonicalGenericSignatureWalk(typ.Value, blocked, replace)
	case *ast.ParenExpr:
		typ.X = canonicalGenericSignatureWalk(typ.X, blocked, replace)
	case *ast.Ellipsis:
		typ.Elt = canonicalGenericSignatureWalk(typ.Elt, blocked, replace)
	}
	return expr
}
