package transpiler

import "strings"

// Primitive argument captures retain the selected Java formal's physical Go
// type. A selector constant can be an untyped Go value even though Java already
// knows its primitive type; := would otherwise infer Go int or float64 here.
// Reference, unknown and type-parameter formals retain their existing guards.
func invocationArgumentHasPrimitiveFormal(javaType string) bool {
	_, primitive := javaPrimitiveType(strings.TrimSpace(javaType))
	return primitive
}
