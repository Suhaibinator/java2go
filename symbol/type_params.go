package symbol

// JavaType is a lightweight representation of a Java type as it appears in source.
// For now this is kept as an original string; it can be extended later to support
// richer constraint translation (e.g. bounds -> Go interfaces).
type JavaType struct {
	Original string
}

// TypeParam represents a declared type parameter (class or method), including
// any upper bounds (e.g. `T extends Number & Comparable<T>`).
type TypeParam struct {
	Name   string
	Bounds []JavaType
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
