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

// sanitizeGoIdent renames a Java identifier that collides with a Go reserved
// keyword (e.g. a variable named `map` or `type`) to a safe form by appending an
// underscore. Applied consistently at every point a Java identifier becomes a Go
// identifier so a declaration and its references rename identically.
func sanitizeGoIdent(name string) string {
	if _, isKeyword := goKeywords[name]; isKeyword {
		return name + "_"
	}
	return name
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
		// JDK packages have no Go import path; never emit them.
		if used && !isJavaStdlibPackage(javaPkg) {
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

// stdjavaImportPath is the Go import path of the runtime support package that
// generated code uses for Java behavior with no direct Go analogue. It is keyed
// in the import maps by its full path; javaPackageToGoImportPath leaves a path
// that already contains slashes untouched.
const stdjavaImportPath = "github.com/NickyBoy89/java2go/stdjava"

func javaPackageToGoImportPath(javaPkg string) string {
	javaPkg = strings.TrimSpace(javaPkg)
	// A key that already looks like a Go import path (contains a slash) is used
	// verbatim. This covers the stdjava runtime package, whose path does not
	// follow Java's dotted-package convention.
	if strings.Contains(javaPkg, "/") {
		return javaPkg
	}
	return strings.ReplaceAll(javaPkg, ".", "/")
}

// stdjavaQualifiedExpr returns a selector expression referencing name within the
// stdjava runtime package (e.g. stdjava.StringCharAt) and registers the import.
func stdjavaQualifiedExpr(name string, ctx Ctx) ast.Expr {
	markStdjavaUsage(ctx)
	return &ast.SelectorExpr{
		X:   &ast.Ident{Name: "stdjava"},
		Sel: &ast.Ident{Name: name},
	}
}

// markStdjavaUsage registers the stdjava runtime package import under a fixed
// "stdjava" alias so generated references resolve.
func markStdjavaUsage(ctx Ctx) {
	if ctx.importAliases != nil {
		ctx.importAliases[stdjavaImportPath] = "stdjava"
	}
	if ctx.usedImports != nil {
		ctx.usedImports[stdjavaImportPath] = true
	}
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

// definitionDeclaresGoIdent reports whether a symbol definition or anything in
// its lexical subtree declares name after Java-to-Go identifier sanitization.
// Import aliases live in the file block and can be shadowed by parameters or
// locals at a later qualified use, so package alias allocation must reserve all
// identifiers in the source file rather than only other import aliases.
func definitionDeclaresGoIdent(def *symbol.Definition, name string) bool {
	if def == nil {
		return false
	}
	if sanitizeGoIdent(def.OriginalName) == name || sanitizeGoIdent(def.Name) == name {
		return true
	}
	for _, param := range def.Parameters {
		if definitionDeclaresGoIdent(param, name) {
			return true
		}
	}
	for _, child := range def.Children {
		if definitionDeclaresGoIdent(child, name) {
			return true
		}
	}
	return false
}

func classScopeDeclaresGoIdent(scope *symbol.ClassScope, name string) bool {
	if scope == nil {
		return false
	}
	if definitionDeclaresGoIdent(scope.Class, name) {
		return true
	}
	for _, field := range scope.Fields {
		if definitionDeclaresGoIdent(field, name) {
			return true
		}
	}
	for _, method := range scope.Methods {
		if definitionDeclaresGoIdent(method, name) {
			return true
		}
	}
	for _, nested := range scope.Subclasses {
		if classScopeDeclaresGoIdent(nested, name) {
			return true
		}
	}
	return false
}

func fileScopeDeclaresGoIdent(file *symbol.FileScope, name string) bool {
	if file == nil || name == "" {
		return false
	}
	if len(file.TopLevelClasses) == 0 {
		return classScopeDeclaresGoIdent(file.BaseClass, name)
	}
	for _, class := range file.TopLevelClasses {
		if classScopeDeclaresGoIdent(class, name) {
			return true
		}
	}
	return false
}

func packageAliasUnavailable(ctx Ctx, alias string) bool {
	return packageAliasTaken(ctx, alias) || fileScopeDeclaresGoIdent(ctx.currentFile, alias)
}

// isJavaStdlibPackage reports whether a Java package belongs to the JDK
// (java.* / javax.*). These have no Go import path: their types are either
// modelled by the stdjava runtime / intrinsics table or mapped to Go builtins,
// so they must never be emitted as Go imports (an `import "java/util"` is an
// invalid path that fails to compile).
func isJavaStdlibPackage(javaPkg string) bool {
	javaPkg = strings.TrimSpace(javaPkg)
	return javaPkg == "java" || javaPkg == "javax" ||
		strings.HasPrefix(javaPkg, "java.") || strings.HasPrefix(javaPkg, "javax.")
}

func markJavaPackageUsage(ctx Ctx, javaPkg string) string {
	javaPkg = strings.TrimSpace(javaPkg)
	if javaPkg == "" || ctx.currentFile == nil || javaPkg == ctx.currentFile.Package {
		return ""
	}
	// JDK packages are never emitted as Go imports.
	if isJavaStdlibPackage(javaPkg) {
		return ""
	}

	alias, exists := ctx.importAliases[javaPkg]
	if !exists || alias == "" {
		baseAlias := packageAliasFromJavaPackage(javaPkg)
		alias = baseAlias
		if currentPkgAlias := packageAliasFromJavaPackage(ctx.currentFile.Package); alias == currentPkgAlias || fileScopeDeclaresGoIdent(ctx.currentFile, alias) {
			alias = baseAlias + "pkg"
		}
		for ind := 2; packageAliasUnavailable(ctx, alias); ind++ {
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

func findFileScopeForClassScope(scope *symbol.ClassScope) *symbol.FileScope {
	if scope == nil {
		return nil
	}
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file != nil && file.BaseClass != nil && classScopeContains(file.BaseClass, scope) {
				return file
			}
		}
	}
	return nil
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
