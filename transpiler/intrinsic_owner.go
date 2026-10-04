package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

// Only migrated intrinsic families participate. Other legacy registrations keep
// their existing resolution until they acquire explicit canonical owners.
var intrinsicOwners = map[string]map[string]bool{}
var intrinsicOwnerKeys = map[string]string{}
var intrinsicDefaultOwners = map[string]string{}

func registerIntrinsicOwner(owner string, supported bool) {
	name := stripJavaQualifier(owner)
	if intrinsicOwners[name] == nil {
		intrinsicOwners[name] = map[string]bool{}
	}
	intrinsicOwners[name][owner] = supported
	if supported && intrinsicDefaultOwners[name] == "" {
		intrinsicDefaultOwners[name] = owner
	}
}
func intrinsicOwnerKey(owner string) string {
	if key := intrinsicOwnerKeys[owner]; key != "" {
		return key
	}
	return stripJavaQualifier(owner)
}

// canonicalIntrinsicOwner resolves a registered external family before its
// simple name is used as a dispatch key. Source declarations always win for
// unqualified names; exact qualified external names never borrow a source name.
func canonicalIntrinsicOwner(javaType string, ctx Ctx) (string, bool) {
	base, _ := parseJavaTypeString(strings.TrimSpace(javaType))
	if owner := canonicalMapEntryOwner(base, ctx); owner != "" {
		return owner, true
	}
	owners, registered := intrinsicOwners[stripJavaQualifier(base)]
	if !registered || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return "", false
	}
	if strings.Contains(base, ".") {
		// An exact registered declaration is already canonical. In particular,
		// a source class with the outer simple name cannot shadow this owner.
		if _, exact := owners[base]; exact {
			return base, true
		}
		// A partially qualified nested declaration inherits the identity of
		// its resolved outer owner. Keep source/value bindings in the caller's
		// qualifier checks; do not guess a nested owner from its simple name.
		separator := strings.LastIndexByte(base, '.')
		outer, recognized := canonicalIntrinsicOwner(base[:separator], ctx)
		if recognized && intrinsicOwnerSupported(outer) {
			nested := outer + base[separator:]
			if _, known := owners[nested]; known {
				return nested, true
			}
		}
		return base, true
	}
	if ctx.currentFile != nil {
		if pkg, ok := ctx.currentFile.Imports[base]; ok {
			return pkg + "." + base, true
		}
	}
	candidates := map[string]bool{}
	for _, pkg := range intrinsicOnDemandImports(ctx) {
		owner := pkg + "." + base
		if _, known := owners[owner]; known {
			candidates[owner] = true
		}
	}
	// java.lang is implicitly imported; this also detects ambiguity with an
	// on-demand import when both canonical declarations are registered.
	if _, known := owners["java.lang."+base]; known {
		candidates["java.lang."+base] = true
	}
	if len(candidates) == 1 {
		for owner := range candidates {
			return owner, true
		}
	}
	if len(candidates) > 1 {
		return "ambiguous owner for " + base, true
	}
	// Historical isolated transpiler inputs often omit import declarations. Keep
	// that convention only when no competing canonical owner has been selected.
	if owner := intrinsicDefaultOwners[base]; owner != "" {
		return owner, true
	}
	return base, true
}

func intrinsicOnDemandImports(ctx Ctx) []string {
	if ctx.currentFile == nil {
		return nil
	}
	scope := ctx.currentClass
	if scope == nil {
		scope = ctx.currentFile.BaseClass
	}
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil {
		return nil
	}
	root := scope.Class.DeclarationNode
	for root.Parent() != nil {
		root = root.Parent()
	}
	var imports []string
	for _, node := range nodeutil.NamedChildrenOf(root) {
		if node.Type() != "import_declaration" {
			continue
		}
		wildcard, static := false, false
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			wildcard = wildcard || child.Content(ctx.currentFile.Source) == "*"
			static = static || child.Type() == "static"
		}
		if wildcard && !static && node.NamedChildCount() > 0 {
			imports = append(imports, node.NamedChild(0).Content(ctx.currentFile.Source))
		}
	}
	return imports
}

func intrinsicOwnerSupported(owner string) bool {
	return intrinsicOwners[stripJavaQualifier(owner)][owner]
}

func unsupportedIntrinsicOwnerValue(owner string, node *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	diagnostic := reportUnsupported("JDK owner "+owner, node, source, ctx)
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: callIdent("panic", &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(diagnostic.String())})}}},
	}}
}
