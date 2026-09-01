package symbol

import (
	"fmt"
	"strings"
)

// JavaType is a lightweight representation of a Java type as it appears in source.
// For now this is kept as an original string; it can be extended later to support
// richer constraint translation (e.g. bounds -> Go interfaces).
type JavaType struct {
	Original              string
	TypeParameterBindings map[string]*TypeParamDeclaration
}

// TypeParamDeclaration is the stable identity of one Java type-parameter
// declaration. SourceName participates in Java lexical lookup; GoName is the
// collision-free binder emitted after hoisting or helper synthesis. The pointer
// identity and both names survive ordinary TypeParam value copies.
type TypeParamDeclaration struct {
	SourceName string
	GoName     string
}

// TypeParam represents a declared type parameter (class or method), including
// any upper bounds (e.g. `T extends Number & Comparable<T>`).
type TypeParam struct {
	Name        string
	Bounds      []JavaType
	Declaration *TypeParamDeclaration
}

// NewTypeParam creates a parameter with declaration identity. Legacy and test
// callers may still use a struct literal; helpers below provide deterministic
// fallbacks for those zero-identity values.
func NewTypeParam(name string, bounds []JavaType) TypeParam {
	name = strings.TrimSpace(name)
	return TypeParam{
		Name:   name,
		Bounds: append([]JavaType(nil), bounds...),
		Declaration: &TypeParamDeclaration{
			SourceName: name,
			GoName:     name,
		},
	}
}

func (p TypeParam) EmittedName() string {
	if p.Declaration != nil && strings.TrimSpace(p.Declaration.GoName) != "" {
		return p.Declaration.GoName
	}
	return p.Name
}

func TypeParamNames(params []TypeParam) []string {
	if len(params) == 0 {
		return nil
	}
	names := make([]string, 0, len(params))
	for _, p := range params {
		names = append(names, p.Name)
	}
	return names
}

// GoTypeParamNames returns the generated binder spellings. It is intentionally
// separate from TypeParamNames, which remains the Java source-name view used by
// lexical lookup and therefore preserves compatibility for existing callers.
func GoTypeParamNames(params []TypeParam) []string {
	if len(params) == 0 {
		return nil
	}
	names := make([]string, 0, len(params))
	for _, parameter := range params {
		names = append(names, parameter.EmittedName())
	}
	return names
}

// AppendTypeParamsByDeclaration builds a generated ABI list without applying
// Java name shadowing. Distinct declarations with the same source spelling are
// retained. Legacy parameters without identity use a deterministic structural
// key so copied synthetic parameters still deduplicate predictably.
func AppendTypeParamsByDeclaration(groups ...[]TypeParam) []TypeParam {
	var result []TypeParam
	seenDeclarations := map[*TypeParamDeclaration]struct{}{}
	seenLegacy := map[string]struct{}{}
	for _, group := range groups {
		for _, parameter := range group {
			if parameter.Declaration != nil {
				if _, duplicate := seenDeclarations[parameter.Declaration]; duplicate {
					continue
				}
				seenDeclarations[parameter.Declaration] = struct{}{}
				result = append(result, parameter)
				continue
			}
			key := legacyTypeParamKey(parameter)
			if _, duplicate := seenLegacy[key]; duplicate {
				continue
			}
			seenLegacy[key] = struct{}{}
			result = append(result, parameter)
		}
	}
	return result
}

func legacyTypeParamKey(parameter TypeParam) string {
	parts := make([]string, 0, len(parameter.Bounds))
	for _, bound := range parameter.Bounds {
		parts = append(parts, strings.TrimSpace(bound.Original))
	}
	return parameter.Name + "\x00" + strings.Join(parts, "\x00")
}

// DisambiguateTypeParamGoNames assigns deterministic generated names while
// preserving names already allocated in an outer generation context. The first
// declaration of a source spelling keeps that spelling; later declarations get
// a numeric suffix that cannot collide with another source spelling or a
// previously allocated emitted name. This matters because an outer declaration
// may already have been copied into generated AST before a local class adds a
// same-named declaration.
func DisambiguateTypeParamGoNames(parameters []TypeParam) {
	// Older synthetic call sites may still construct TypeParam values directly.
	// Give each retained legacy value an identity here so collisions can be
	// resolved just like parser-backed declarations. AppendTypeParamsByDeclaration
	// has already removed structurally identical legacy copies.
	for index := range parameters {
		if parameters[index].Declaration != nil {
			continue
		}
		parameters[index].Declaration = &TypeParamDeclaration{
			SourceName: parameters[index].Name,
			GoName:     parameters[index].Name,
		}
	}

	reserved := make(map[string]struct{}, len(parameters))
	occupied := make(map[string]*TypeParamDeclaration, len(parameters))
	for _, parameter := range parameters {
		declaration := parameter.Declaration
		source := declaration.SourceName
		if source == "" {
			source = parameter.Name
			declaration.SourceName = source
		}
		reserved[source] = struct{}{}
		if emitted := strings.TrimSpace(declaration.GoName); emitted != "" && emitted != source {
			// A non-source spelling was allocated by an earlier/outer context and
			// is frozen: later lowering must never invalidate already-built AST.
			if _, exists := occupied[emitted]; !exists {
				occupied[emitted] = declaration
			}
		}
	}

	for _, parameter := range parameters {
		declaration := parameter.Declaration
		source := declaration.SourceName
		if emitted := strings.TrimSpace(declaration.GoName); emitted != "" && emitted != source {
			continue
		}
		if owner := occupied[source]; owner == nil || owner == declaration {
			declaration.GoName = source
			occupied[source] = declaration
			continue
		}

		for suffix := 2; ; suffix++ {
			candidate := fmt.Sprintf("%s%d", source, suffix)
			if _, collision := reserved[candidate]; collision {
				continue
			}
			if _, collision := occupied[candidate]; collision {
				continue
			}
			declaration.GoName = candidate
			occupied[candidate] = declaration
			break
		}
	}
}

func FindVisibleTypeParam(parameters []TypeParam, sourceName string) *TypeParamDeclaration {
	for index := len(parameters) - 1; index >= 0; index-- {
		parameter := parameters[index]
		if parameter.Name == sourceName {
			return parameter.Declaration
		}
	}
	return nil
}

// DirectTypeParamForJavaType resolves a bare (optionally array-qualified) Java
// type-parameter use against the supplied lexical view. Nested parameterized
// types retain their string representation for the existing recursive type
// converter.
func DirectTypeParamForJavaType(javaType string, parameters []TypeParam) *TypeParamDeclaration {
	javaType = strings.TrimSpace(javaType)
	for strings.HasSuffix(javaType, "[]") {
		javaType = strings.TrimSpace(strings.TrimSuffix(javaType, "[]"))
	}
	if javaType == "" || strings.ContainsAny(javaType, "<>,.? ") {
		return nil
	}
	return FindVisibleTypeParam(parameters, javaType)
}

// VisibleTypeParamBindings returns Java's innermost declaration for each source
// spelling in an ordered lexical view. The map may safely travel with a copied
// Definition because declarations themselves have stable pointer identity.
func VisibleTypeParamBindings(parameters []TypeParam) map[string]*TypeParamDeclaration {
	bindings := map[string]*TypeParamDeclaration{}
	for _, parameter := range parameters {
		if parameter.Declaration != nil {
			bindings[parameter.Name] = parameter.Declaration
		}
	}
	if len(bindings) == 0 {
		return nil
	}
	return bindings
}

// BindTypeParameterBounds captures the lexical declaration selected by names
// inside each bound. Call it only for the newly declared parameters; carried
// outer parameters already retain bindings from their own declaration scope.
func BindTypeParameterBounds(parameters []TypeParam, visible []TypeParam) {
	bindings := VisibleTypeParamBindings(visible)
	for parameterIndex := range parameters {
		for boundIndex := range parameters[parameterIndex].Bounds {
			parameters[parameterIndex].Bounds[boundIndex].TypeParameterBindings = bindings
		}
	}
}

// MergeTypeParams merges outer and inner type parameters, applying Java-style
// shadowing: if an inner type parameter has the same name as an outer one, the
// inner one replaces it.
func MergeTypeParams(outer, inner []TypeParam) []TypeParam {
	if len(outer) == 0 {
		return append([]TypeParam{}, inner...)
	}
	if len(inner) == 0 {
		return append([]TypeParam{}, outer...)
	}

	shadowed := make(map[string]struct{}, len(inner))
	for _, p := range inner {
		shadowed[p.Name] = struct{}{}
	}

	merged := make([]TypeParam, 0, len(outer)+len(inner))
	for _, p := range outer {
		if _, ok := shadowed[p.Name]; ok {
			continue
		}
		merged = append(merged, p)
	}
	merged = append(merged, inner...)
	return merged
}
