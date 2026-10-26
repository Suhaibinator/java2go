package symbol

import sitter "github.com/smacker/go-tree-sitter"

// RecordComponentOrigin is a read-only copy of parser-retained member identity.
// Definition pointers remain mutable for legitimate Go naming/layout passes;
// Java declaration roles and signature spellings are checked before each use.
type RecordComponentOrigin struct {
	Node     *sitter.Node
	Field    *Definition
	Accessor *Definition
}
type recordComponentOrigin struct {
	view      RecordComponentOrigin
	name, typ string
	bindings  map[string]*TypeParamDeclaration
	direct    *TypeParamDeclaration
}
type recordTypeParamOrigin struct {
	name        string
	declaration *TypeParamDeclaration
	sourceName  string
	bounds      []JavaType
}
type recordMemberOrigin struct {
	owner       *ClassScope
	declaration *sitter.Node
	parameters  []recordTypeParamOrigin
	carried     []recordTypeParamOrigin
	components  []recordComponentOrigin
}

func (scope *ClassScope) beginRecordMemberOrigin(node *sitter.Node) {
	scope.recordMemberOrigin = &recordMemberOrigin{owner: scope, declaration: node}
	scope.recordMemberOrigin.parameters = recordMemberTypeParamOrigins(scope.DeclaredTypeParameters)
	scope.recordMemberOrigin.carried = recordMemberTypeParamOrigins(scope.TypeParameters)
}

func recordMemberTypeParamOrigins(parameters []TypeParam) []recordTypeParamOrigin {
	result := make([]recordTypeParamOrigin, 0, len(parameters))
	for _, parameter := range parameters {
		snapshot := recordTypeParamOrigin{name: parameter.Name, declaration: parameter.Declaration}
		if parameter.Declaration != nil {
			snapshot.sourceName = parameter.Declaration.SourceName
		}
		for _, bound := range parameter.Bounds {
			copy := JavaType{Original: bound.Original, TypeParameterBindings: make(map[string]*TypeParamDeclaration, len(bound.TypeParameterBindings))}
			for name, binding := range bound.TypeParameterBindings {
				copy.TypeParameterBindings[name] = binding
			}
			snapshot.bounds = append(snapshot.bounds, copy)
		}
		result = append(result, snapshot)
	}
	return result
}

// Both declaration and carried parameters feed Java erasure. They have distinct
// slice values; validating only one would permit a changed bound in the other.
// Generated Go binder names are intentionally outside this Java identity check.
func recordMemberTypeParamsCurrent(parameters []TypeParam, snapshots []recordTypeParamOrigin) bool {
	if len(parameters) != len(snapshots) {
		return false
	}
	for i, snapshot := range snapshots {
		current := parameters[i]
		if current.Name != snapshot.name || current.Declaration != snapshot.declaration || (current.Declaration != nil && current.Declaration.SourceName != snapshot.sourceName) || len(current.Bounds) != len(snapshot.bounds) {
			return false
		}
		for j, bound := range current.Bounds {
			prior := snapshot.bounds[j]
			if bound.Original != prior.Original || len(bound.TypeParameterBindings) != len(prior.TypeParameterBindings) {
				return false
			}
			for name, binding := range prior.TypeParameterBindings {
				if bound.TypeParameterBindings[name] != binding {
					return false
				}
			}
		}
	}
	return true
}

func (scope *ClassScope) retainRecordMemberOrigin(node *sitter.Node, field, accessor *Definition) {
	bindings := make(map[string]*TypeParamDeclaration, len(field.TypeParameterBindings))
	for name, binding := range field.TypeParameterBindings {
		bindings[name] = binding
	}
	scope.recordMemberOrigin.components = append(scope.recordMemberOrigin.components, recordComponentOrigin{
		view: RecordComponentOrigin{Node: node, Field: field, Accessor: accessor}, name: field.OriginalName, typ: field.OriginalType, bindings: bindings, direct: field.DirectTypeParameter,
	})
}

// RecordComponentOrigins validates the original parser owner and current Java
// roles. No copied scope or donated shape can create a new origin certificate.
func (scope *ClassScope) RecordComponentOrigins() ([]RecordComponentOrigin, bool) {
	if scope == nil || scope.recordMemberOrigin == nil {
		return nil, false
	}
	origin := scope.recordMemberOrigin
	if origin.owner != scope || scope.Class == nil || scope.Class.DeclarationNode == nil || origin.declaration == nil || !origin.declaration.Equal(scope.Class.DeclarationNode) {
		return nil, false
	}
	if !recordMemberTypeParamsCurrent(scope.DeclaredTypeParameters, origin.parameters) || !recordMemberTypeParamsCurrent(scope.TypeParameters, origin.carried) {
		return nil, false
	}
	result := make([]RecordComponentOrigin, len(origin.components))
	for i, c := range origin.components {
		f := c.view.Field
		if f == nil || f.OriginalName != c.name || f.OriginalType != c.typ || f.DeclarationNode != nil || f.IsStatic || f.Constructor || f.DirectTypeParameter != c.direct || len(f.TypeParameterBindings) != len(c.bindings) {
			return nil, false
		}
		for name, binding := range c.bindings {
			if f.TypeParameterBindings[name] != binding {
				return nil, false
			}
		}
		count := 0
		for _, current := range scope.Fields {
			if current == f {
				count++
			}
		}
		if count != 1 {
			return nil, false
		}
		if m := c.view.Accessor; m != nil {
			if m.RuntimeDefault || (m.DirectTypeParameter != nil && m.DirectTypeParameter != c.direct) {
				return nil, false
			}
			if len(m.TypeParameterBindings) != 0 {
				if len(m.TypeParameterBindings) != len(c.bindings) {
					return nil, false
				}
				for name, binding := range c.bindings {
					if m.TypeParameterBindings[name] != binding {
						return nil, false
					}
				}
			}
			if m.OriginalName != c.name || m.OriginalType != c.typ || m.DeclarationNode != nil || m.IsStatic || m.IsPrivate || m.Constructor || len(m.Parameters) != 0 || len(m.TypeParameters) != 0 || m.RequiresHelper {
				return nil, false
			}
			count = 0
			for _, current := range scope.Methods {
				if current == m {
					count++
				}
			}
			if count != 1 {
				return nil, false
			}
			if scope.FindMethodByName(c.name, nil) != m {
				return nil, false
			}
		}
		result[i] = c.view
	}
	return result, true
}
