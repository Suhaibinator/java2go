package transpiler

import (
	"go/ast"
	"go/token"
	"sort"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

var goKeywords = map[string]struct{}{
	"break": {}, "default": {}, "func": {}, "interface": {}, "select": {},
	"case": {}, "defer": {}, "go": {}, "map": {}, "struct": {},
	"chan": {}, "else": {}, "goto": {}, "package": {}, "switch": {},
	"const": {}, "fallthrough": {}, "if": {}, "range": {}, "type": {},
	"continue": {}, "for": {}, "import": {}, "return": {}, "var": {},
}

func parseImportDeclaration(node *sitter.Node, source []byte) *ast.ImportSpec {
	importNode := node.NamedChild(0)
	if importNode == nil {
		return &ast.ImportSpec{}
	}

	importPath := ""
	if scopeNode := importNode.ChildByFieldName("scope"); scopeNode != nil {
		importPath = scopeNode.Content(source)
	}
	if importPath == "" {
		content := strings.TrimSpace(importNode.Content(source))
		if ind := strings.LastIndex(content, "."); ind >= 0 {
			importPath = content[:ind]
		}
	}
	if importPath == "" {
		return &ast.ImportSpec{}
	}

	return &ast.ImportSpec{
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(javaPackageToGoImportPath(importPath)),
		},
	}
}

func buildUsedImportSpecs(ctx Ctx) []*ast.ImportSpec {
	if len(ctx.usedImports) == 0 {
		return nil
	}

	javaPkgs := make([]string, 0, len(ctx.usedImports))
	for javaPkg, used := range ctx.usedImports {
		if used {
			javaPkgs = append(javaPkgs, javaPkg)
		}
	}
	sort.Strings(javaPkgs)

	specs := make([]*ast.ImportSpec, 0, len(javaPkgs))
	for _, javaPkg := range javaPkgs {
		alias, ok := ctx.importAliases[javaPkg]
		if !ok || alias == "" {
			continue
		}
		specs = append(specs, &ast.ImportSpec{
			Name: &ast.Ident{Name: alias},
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(javaPackageToGoImportPath(javaPkg)),
			},
		})
	}
	return specs
}

func javaPackageToGoImportPath(javaPkg string) string {
	return strings.ReplaceAll(strings.TrimSpace(javaPkg), ".", "/")
}

func packageAliasFromJavaPackage(javaPkg string) string {
	javaPkg = strings.TrimSpace(javaPkg)
	if javaPkg == "" {
		return ""
	}
	parts := strings.Split(javaPkg, ".")
	last := parts[len(parts)-1]
	if last == "" {
		last = "pkg"
	}

	var out strings.Builder
	for ind, ch := range last {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' {
			out.WriteRune(ch)
			continue
		}
		if ch >= '0' && ch <= '9' {
			if ind == 0 {
				out.WriteRune('_')
			}
			out.WriteRune(ch)
			continue
		}
		out.WriteRune('_')
	}
	alias := out.String()
	if alias == "" {
		alias = "pkg"
	}
	if _, keyword := goKeywords[alias]; keyword {
		alias = "pkg_" + alias
	}
	return alias
}

func packageAliasTaken(ctx Ctx, alias string) bool {
	for _, existing := range ctx.importAliases {
		if existing == alias {
			return true
		}
	}
	return false
}

func markJavaPackageUsage(ctx Ctx, javaPkg string) string {
	javaPkg = strings.TrimSpace(javaPkg)
	if javaPkg == "" || ctx.currentFile == nil || javaPkg == ctx.currentFile.Package {
		return ""
	}

	alias, exists := ctx.importAliases[javaPkg]
	if !exists || alias == "" {
		baseAlias := packageAliasFromJavaPackage(javaPkg)
		alias = baseAlias
		if currentPkgAlias := packageAliasFromJavaPackage(ctx.currentFile.Package); alias == currentPkgAlias {
			alias = baseAlias + "pkg"
		}
		for ind := 2; packageAliasTaken(ctx, alias); ind++ {
			alias = baseAlias + strconv.Itoa(ind)
		}
		if ctx.importAliases != nil {
			ctx.importAliases[javaPkg] = alias
		}
	}

	if ctx.usedImports != nil {
		ctx.usedImports[javaPkg] = true
	}

	return alias
}

func qualifiedNameExpr(name string, javaPkg string, ctx Ctx) ast.Expr {
	name = strings.TrimSpace(name)
	if name == "" {
		return &ast.Ident{Name: ""}
	}
	if alias := markJavaPackageUsage(ctx, javaPkg); alias != "" {
		return &ast.SelectorExpr{
			X:   &ast.Ident{Name: alias},
			Sel: &ast.Ident{Name: name},
		}
	}
	return &ast.Ident{Name: name}
}

func classScopeContains(root *symbol.ClassScope, target *symbol.ClassScope) bool {
	if root == nil || target == nil {
		return false
	}
	if root == target {
		return true
	}
	for _, sub := range root.Subclasses {
		if classScopeContains(sub, target) {
			return true
		}
	}
	return false
}

func findJavaPackageForClassScope(scope *symbol.ClassScope) string {
	if scope == nil {
		return ""
	}
	for pkgName, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file == nil || file.BaseClass == nil {
				continue
			}
			if classScopeContains(file.BaseClass, scope) {
				return pkgName
			}
		}
	}
	return ""
}

func resolveJavaPackageForType(ctx Ctx, javaTypeBase string, scope *symbol.ClassScope) string {
	javaTypeBase = strings.TrimSpace(javaTypeBase)
	if javaTypeBase == "" {
		return ""
	}

	if ind := strings.LastIndex(javaTypeBase, "."); ind > 0 {
		pkgCandidate := strings.TrimSpace(javaTypeBase[:ind])
		if symbol.GlobalScope.FindPackage(pkgCandidate) != nil {
			return pkgCandidate
		}
	}

	simpleName := stripJavaQualifier(javaTypeBase)
	if ctx.currentFile != nil {
		if importedPkg, ok := ctx.currentFile.Imports[simpleName]; ok {
			return importedPkg
		}
		if ctx.currentFile.FindClassScope(simpleName) != nil {
			return ctx.currentFile.Package
		}
	}

	if inferredPkg := findJavaPackageForClassScope(scope); inferredPkg != "" {
		return inferredPkg
	}

	return ""
}
