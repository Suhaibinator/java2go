package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

type recordConstructorComponent struct {
	node  *sitter.Node
	field *symbol.Definition
}

// Match the whole registered lexical chain to the actual parse declarations.
// Root membership and a simple name alone cannot distinguish A.R from B.R
// declared in the same compilation unit.
func recordConstructorLexicalChainOwned(scope *symbol.ClassScope, file *symbol.FileScope) bool {
	if scope == nil || file == nil || file.ParsedRoot() == nil {
		return false
	}
	seen := map[*symbol.ClassScope]bool{}
	for current := scope; current != nil; current = current.Enclosing {
		if seen[current] || current.Class == nil || current.Class.DeclarationNode == nil {
			return false
		}
		seen[current] = true
		declaration := current.Class.DeclarationNode
		switch declaration.Type() {
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration":
		default:
			return false
		}
		if !threadSourceNodeWithin(declaration, file.ParsedRoot()) {
			return false
		}
		name := declaration.ChildByFieldName("name")
		if name == nil || name.Content(file.Source) != current.Class.OriginalName {
			return false
		}
		var registered []*symbol.ClassScope
		var body *sitter.Node
		if current.Enclosing == nil {
			registered = file.TopLevelClasses
			body = file.ParsedRoot()
		} else {
			parent := current.Enclosing
			if parent.Class == nil || parent.Class.DeclarationNode == nil {
				return false
			}
			registered = parent.Subclasses
			body = parent.Class.DeclarationNode.ChildByFieldName("body")
		}
		count := 0
		for _, member := range registered {
			if member == current {
				count++
			}
		}
		if count != 1 || body == nil {
			return false
		}
		direct := 0
		for _, member := range nodeutil.NamedChildrenOf(body) {
			if member.Equal(declaration) {
				direct++
			}
			// Enum member declarations occur after the constant list in this exact
			// grammar container; no method, lambda or arbitrary subtree is searched.
			if body.Type() == "enum_body" && member.Type() == "enum_body_declarations" {
				for _, child := range nodeutil.NamedChildrenOf(member) {
					if child.Equal(declaration) {
						direct++
					}
				}
			}
		}
		if direct != 1 {
			return false
		}
	}
	return true
}

// Validate the source owner even when a record has no components. No empty
// loop may certify a detached record or a replaced source snapshot.
func recordConstructorSourceFile(scope *symbol.ClassScope) *symbol.FileScope {
	if !sourceReflectionRecord(scope) {
		return nil
	}
	file := findFileScopeForClassScope(scope)
	if file == nil || file.SourceInputSnapshot() == nil || file.SourceInputSnapshot().Validate() != nil || !threadSourceNodeWithin(scope.Class.DeclarationNode, file.ParsedRoot()) {
		return nil
	}
	name := scope.Class.DeclarationNode.ChildByFieldName("name")
	if name == nil || name.Content(file.Source) != scope.Class.OriginalName || !recordConstructorLexicalChainOwned(scope, file) {
		return nil
	}
	return file
}

// A written constructor must be a unique actual direct member of this record.
// A Definition whose node was donated from another owner is not a declaration.
func recordWrittenConstructorOwned(scope *symbol.ClassScope, method *symbol.Definition) bool {
	file := recordConstructorSourceFile(scope)
	if file == nil || method == nil || !method.Constructor || method.IsStatic || method.OriginalName != scope.Class.OriginalName || method.DeclarationNode == nil {
		return false
	}
	node := method.DeclarationNode
	if node.Type() != "constructor_declaration" && node.Type() != "compact_constructor_declaration" {
		return false
	}
	name := node.ChildByFieldName("name")
	if name == nil || name.Content(file.Source) != method.OriginalName {
		return false
	}
	found := 0
	for _, child := range nodeutil.NamedChildrenOf(scope.Class.DeclarationNode.ChildByFieldName("body")) {
		if child.Equal(node) {
			found++
		}
	}
	if found != 1 {
		return false
	}
	registered := 0
	for _, candidate := range scope.Methods {
		if candidate == method {
			registered++
		}
		if candidate != nil && candidate != method && candidate.DeclarationNode != nil && candidate.DeclarationNode.Equal(node) {
			return false
		}
	}
	if registered != 1 {
		return false
	}
	parameters := node.ChildByFieldName("parameters")
	if node.Type() == "compact_constructor_declaration" {
		parameters = scope.Class.DeclarationNode.ChildByFieldName("parameters")
	}
	formals := nodeutil.SemanticNamedChildrenOf(parameters)
	if len(formals) != len(method.Parameters) {
		return false
	}
	for index, formal := range formals {
		if formal.Type() != "formal_parameter" && formal.Type() != "spread_parameter" {
			return false
		}
		_, name := nodeutil.JavaParameterNodes(formal)
		if name == nil || method.Parameters[index] == nil || name.Content(file.Source) != method.Parameters[index].OriginalName || !declarationParameterMatchesDefinition(formal, method, index, file.Source) {
			return false
		}
	}
	return true
}

func recordImplicitConstructorShape(scope *symbol.ClassScope, method *symbol.Definition, components []recordConstructorComponent, requireBindings bool, ctx Ctx) bool {
	if method == nil || !method.Constructor || method.IsStatic || method.DeclarationNode != nil || method.OriginalName != scope.Class.OriginalName || len(method.TypeParameters) != 0 || len(method.Parameters) != len(components) {
		return false
	}
	for index, component := range components {
		parameter := method.Parameters[index]
		if parameter == nil || parameter.OriginalName != component.field.OriginalName || strings.TrimSpace(parameter.OriginalType) != strings.TrimSpace(component.field.OriginalType) {
			return false
		}
		if requireBindings && !recordConstructorTypeEqual(definitionDeclaredJavaType(parameter), definitionDeclaredJavaType(component.field), ctx) {
			return false
		}
	}
	return true
}

// Components come from this actual record header, never all fields or a name
// list. Existing ownership/snapshot validation also excludes donated scopes.
func recordConstructorComponents(scope *symbol.ClassScope) ([]recordConstructorComponent, bool) {
	if recordConstructorSourceFile(scope) == nil {
		return nil, false
	}
	parameters := scope.Class.DeclarationNode.ChildByFieldName("parameters")
	if parameters == nil {
		return nil, false
	}
	var result []recordConstructorComponent
	for _, parameter := range nodeutil.SemanticNamedChildrenOf(parameters) {
		// Spread components require a separate array/header metadata ABI. This
		// constructor prerequisite does not silently admit that unprovided form.
		if parameter.Type() != "formal_parameter" {
			return nil, false
		}
		var found *symbol.Definition
		for _, field := range scope.Fields {
			if node := sourceReflectionRecordComponent(scope, field); node != nil && node.Equal(parameter) {
				if found != nil {
					return nil, false
				}
				found = field
			}
		}
		if found == nil {
			return nil, false
		}
		result = append(result, recordConstructorComponent{parameter, found})
	}
	return result, true
}

// Canonical identity is declaration-bound, including class/method binders,
// nominal source origins, external owners, array rank and generic arguments.
// Equal arity or equal Go projections do not establish a Java signature.
func recordConstructorTypeEqual(left, right symbol.JavaType, ctx Ctx) bool {
	l, r := strings.TrimSpace(left.Original), strings.TrimSpace(right.Original)
	wildcard := func(value string) (string, string) {
		if value == "?" {
			return "?", ""
		}
		for _, direction := range []string{"extends", "super"} {
			if strings.HasPrefix(value, "?") {
				tail := strings.TrimSpace(strings.TrimPrefix(value, "?"))
				if strings.HasPrefix(tail, direction+" ") {
					return direction, strings.TrimSpace(strings.TrimPrefix(tail, direction))
				}
			}
		}
		return "", value
	}
	ld, lb := wildcard(l)
	rd, rb := wildcard(r)
	if ld != rd {
		return false
	}
	if ld == "?" {
		return true
	}
	if ld != "" {
		return recordConstructorTypeEqual(symbol.JavaType{Original: lb, TypeParameterBindings: left.TypeParameterBindings}, symbol.JavaType{Original: rb, TypeParameterBindings: right.TypeParameterBindings}, ctx)
	}
	lc, lr := javaArrayTypeParts(l)
	rc, rr := javaArrayTypeParts(r)
	if lr != rr {
		return false
	}
	lbase, la := parseJavaTypeString(lc)
	rbase, ra := parseJavaTypeString(rc)
	if len(la) != len(ra) {
		return false
	}
	lo := declaredJavaTypeOrigin(symbol.JavaType{Original: lbase, TypeParameterBindings: left.TypeParameterBindings}, ctx)
	ro := declaredJavaTypeOrigin(symbol.JavaType{Original: rbase, TypeParameterBindings: right.TypeParameterBindings}, ctx)
	if lo.unresolved || ro.unresolved {
		return false
	}
	same := false
	switch {
	case lo.parameter != nil || ro.parameter != nil:
		same = lo.parameter != nil && lo.parameter == ro.parameter
	case lo.nominalScope != nil || ro.nominalScope != nil:
		same = lo.nominalScope != nil && lo.nominalScope == ro.nominalScope
	case lo.externalName != "" || ro.externalName != "":
		same = lo.externalName != "" && lo.externalName == ro.externalName
	default:
		_, lp := javaPrimitiveType(lbase)
		_, rp := javaPrimitiveType(rbase)
		same = lp && rp && lbase == rbase
	}
	if !same {
		return false
	}
	for index := range la {
		if !recordConstructorTypeEqual(symbol.JavaType{Original: la[index], TypeParameterBindings: left.TypeParameterBindings}, symbol.JavaType{Original: ra[index], TypeParameterBindings: right.TypeParameterBindings}, ctx) {
			return false
		}
	}
	return true
}

func recordConstructorMatchesComponents(method *symbol.Definition, components []recordConstructorComponent, ctx Ctx) bool {
	if method == nil || !method.Constructor || len(method.TypeParameters) != 0 || len(method.Parameters) != len(components) {
		return false
	}
	for index, component := range components {
		parameter := method.Parameters[index]
		typ := definitionDeclaredJavaType(parameter)
		typ.Original = definitionParameterJavaSignatureType(method, index)
		if !recordConstructorTypeEqual(typ, definitionDeclaredJavaType(component.field), ctx) {
			return false
		}
	}
	return true
}

func recordCanonicalConstructor(scope *symbol.ClassScope, ctx Ctx) *symbol.Definition {
	components, known := recordConstructorComponents(scope)
	if !known {
		return nil
	}
	declaring := classScopeCtx(scope, ctx)
	var result *symbol.Definition
	implicitCount := 0
	for _, method := range scope.Methods {
		if method == nil || !method.Constructor {
			continue
		}
		if method.DeclarationNode == nil {
			implicitCount++
			if implicitCount != 1 || !recordImplicitConstructorShape(scope, method, components, true, declaring) {
				return nil
			}
		} else if !recordWrittenConstructorOwned(scope, method) {
			return nil
		}
		if recordConstructorMatchesComponents(method, components, declaring) {
			if result != nil {
				return nil
			}
			result = method
		}
	}
	return result
}

// Parsing can only approximate an overloaded canonical signature. Finish that
// decision against the complete source graph BEFORE overload names and call
// targets are allocated. No new parser tree or Java declaration is fabricated.
func prepareRecordConstructorDefinitions(files []parsing.SourceFile) {
	var visit func(*symbol.ClassScope, Ctx)
	visit = func(scope *symbol.ClassScope, ctx Ctx) {
		if scope == nil {
			return
		}
		if sourceReflectionRecord(scope) {
			components, known := recordConstructorComponents(scope)
			if known {
				declaring := classScopeCtx(scope, ctx)
				var written, implicit *symbol.Definition
				ambiguous := false
				implicitCount := 0
				for _, method := range scope.Methods {
					if method == nil || !method.Constructor {
						continue
					}
					if method.DeclarationNode == nil {
						implicitCount++
						if implicitCount != 1 || !recordImplicitConstructorShape(scope, method, components, false, declaring) {
							ambiguous = true
						}
						implicit = method
						continue
					}
					if !recordWrittenConstructorOwned(scope, method) {
						ambiguous = true
						continue
					}
					if recordConstructorMatchesComponents(method, components, declaring) {
						if written != nil {
							ambiguous = true
						}
						written = method
					}
				}
				if !ambiguous {
					if written != nil {
						methods := scope.Methods[:0]
						for _, method := range scope.Methods {
							if method == nil || !method.Constructor || method.DeclarationNode != nil {
								methods = append(methods, method)
							}
						}
						scope.Methods = methods
					} else {
						if implicit == nil {
							implicit = &symbol.Definition{Name: defaultConstructorName(scope.Class.Name), OriginalName: scope.Class.OriginalName, Constructor: true, Type: scope.Class.OriginalName}
							scope.Methods = append(scope.Methods, implicit)
						}
						implicit.Parameters = nil
						for _, component := range components {
							field := component.field
							implicit.Parameters = append(implicit.Parameters, &symbol.Definition{Name: field.OriginalName, OriginalName: field.OriginalName, Type: field.Type, OriginalType: field.OriginalType, DirectTypeParameter: field.DirectTypeParameter, TypeParameterBindings: field.TypeParameterBindings})
						}
					}
				}
			}
		}
		for _, nested := range scope.Subclasses {
			visit(nested, ctx)
		}
	}
	for _, file := range files {
		if file.Symbols == nil {
			continue
		}
		for _, top := range file.Symbols.TopLevelClasses {
			visit(top, Ctx{currentFile: file.Symbols, currentClass: top})
		}
	}
}
