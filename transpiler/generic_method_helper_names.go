package transpiler

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// Generic instance helpers occupy package scope, unlike Java overloads. Allocate
// their type and constructor names together after ordinary member resolution.
// Rebuilding from declaration identity makes repeated resolution and source-file
// traversal order irrelevant; callers and declarations share Definition.HelperName.
func resolveGenericMethodHelperNames() {
	type entry struct {
		owner  *symbol.ClassScope
		method *symbol.Definition
		key    string
	}
	entries := []entry{}
	reserved := map[string]map[string]bool{}
	reserve := func(pkg, name string) {
		if reserved[pkg] == nil {
			reserved[pkg] = map[string]bool{}
		}
		reserved[pkg][name] = true
	}
	for _, scope := range allSourceClassScopes() {
		pkg := findJavaPackageForClassScope(scope)
		if file := findFileScopeForClassScope(scope); file != nil && scope.Class != nil {
			for name := range affineLoopUsedNames(scope.Class.DeclarationNode, file.Source, Ctx{}) {
				reserve(pkg, name)
			}
		}
		var reserveDefinition func(*symbol.Definition)
		reserveDefinition = func(def *symbol.Definition) {
			if def == nil {
				return
			}
			reserve(pkg, def.Name)
			for _, parameter := range def.TypeParameters {
				reserve(pkg, parameter.EmittedName())
			}
			for _, parameter := range def.Parameters {
				reserveDefinition(parameter)
			}
			for _, child := range def.Children {
				reserveDefinition(child)
			}
		}
		for _, parameter := range scope.TypeParameters {
			reserve(pkg, parameter.EmittedName())
		}
		if scope.Class != nil {
			reserve(pkg, scope.Class.Name)
			reserve(pkg, "New"+scope.Class.Name)
			reserve(pkg, classDispatchTypeName(scope))
			reserve(pkg, interfaceDefaultCarrierName(scope))
		}
		for _, field := range scope.Fields {
			if field != nil {
				reserve(pkg, field.Name)
			}
		}
		for _, method := range scope.Methods {
			if method == nil {
				continue
			}
			reserveDefinition(method)
			if method.RequiresHelper {
				entries = append(entries, entry{scope, method, genericMethodHelperIdentity(scope, method)})
			}
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].key < entries[j].key })
	for _, entry := range entries {
		pkg := findJavaPackageForClassScope(entry.owner)
		methodName := symbol.HandleExportStatus(ast.IsExported(entry.method.Name), entry.method.OriginalName)
		stem := entry.owner.Class.Name + methodName + "Helper"
		for suffix := 0; ; suffix++ {
			candidate := stem
			if suffix > 0 {
				candidate += strconv.Itoa(suffix)
			}
			if reserved[pkg][candidate] || reserved[pkg]["New"+candidate] {
				continue
			}
			entry.method.HelperName = candidate
			reserve(pkg, candidate)
			reserve(pkg, "New"+candidate)
			break
		}
	}
}

// Binder ordinals describe declarations, not emitted Go spellings or allocation
// addresses. In particular a method's T and its owner's T remain distinct even
// when source spelling is identical; method bounds distinguish legal overloads
// whose type-variable formals have different erased descriptors.
func genericMethodHelperIdentity(owner *symbol.ClassScope, method *symbol.Definition) string {
	declarations := map[*symbol.TypeParamDeclaration]string{}
	fallback := map[string]string{}
	var chain []*symbol.ClassScope
	for scope := owner; scope != nil; scope = scope.Enclosing {
		chain = append([]*symbol.ClassScope{scope}, chain...)
	}
	for _, scope := range chain {
		for index, param := range scope.OwnTypeParameters() {
			marker := "@class:" + qualifiedSourceClassName(scope) + ":" + strconv.Itoa(index)
			if param.Declaration != nil {
				declarations[param.Declaration] = marker
			}
			fallback[param.Name] = marker
		}
	}
	for index, param := range method.TypeParameters {
		marker := "@method:" + strconv.Itoa(index)
		if param.Declaration != nil {
			declarations[param.Declaration] = marker
		}
		fallback[param.Name] = marker
	}
	canonical := func(javaType string, bindings map[string]*symbol.TypeParamDeclaration) string {
		replacements := map[string]string{}
		for name, marker := range fallback {
			replacements[name] = marker
		}
		for name, decl := range bindings {
			if marker, ok := declarations[decl]; ok {
				replacements[name] = marker
			}
		}
		return qualifyJavaTypeInDeclaringContext(substituteJavaTypeParameters(javaType, replacements), owner)
	}
	parts := []string{qualifiedSourceClassName(owner), method.OriginalName}
	for _, param := range method.TypeParameters {
		bounds := []string{}
		for _, bound := range param.Bounds {
			bounds = append(bounds, canonical(bound.Original, bound.TypeParameterBindings))
		}
		parts = append(parts, "<"+strings.Join(bounds, "&")+">")
	}
	for index, param := range method.Parameters {
		parts = append(parts, canonical(definitionParameterJavaSignatureType(method, index), param.TypeParameterBindings))
	}
	return strings.Join(parts, ";")
}
