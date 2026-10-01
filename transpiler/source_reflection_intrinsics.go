package transpiler

import "go/ast"

// These registrations pass through the ordinary canonical-owner admission;
// source classes and lexical binders with the same names retain source calls.
func init() {
	registerIntrinsicOwner("java.lang.Class", true)
	registerIntrinsicOwner("java.lang.reflect.Field", true)
	registerIntrinsicOwner("java.lang.reflect.Constructor", true)
	registerIntrinsicOwner("java.lang.reflect.Method", true)
	for receiver, methods := range map[string]map[string]string{
		"Class":       {"getDeclaredFields": "java.lang.reflect.Field[]", "getDeclaredConstructor": "java.lang.reflect.Constructor", "getModifiers": "int", "getGenericSuperclass": "java.lang.reflect.Type", "getTypeParameters": "java.lang.reflect.TypeVariable<?>[]", "getAnnotation": "java.lang.Object"},
		"Field":       {"getModifiers": "int", "getType": "java.lang.Class", "getDeclaringClass": "java.lang.Class", "getGenericType": "java.lang.reflect.Type", "setAccessible": "void", "canAccess": "boolean", "getAnnotation": "java.lang.Object"},
		"Constructor": {"getModifiers": "int", "getDeclaringClass": "java.lang.Class", "setAccessible": "void", "canAccess": "boolean"},
	} {
		for name, result := range methods {
			receiver, name := receiver, name
			registerInstanceIntrinsic(receiver, name, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
				goName := map[string]string{"getDeclaredFields": "GetDeclaredFields", "getDeclaredConstructor": "GetDeclaredConstructor", "getModifiers": "GetModifiers", "getGenericSuperclass": "GetGenericSuperclass", "getTypeParameters": "GetTypeParameters", "getAnnotation": "GetAnnotationExecution", "getType": "GetType", "getDeclaringClass": "GetDeclaringClass", "getGenericType": "GetGenericType", "setAccessible": "SetAccessible", "canAccess": "CanAccess"}[name]
				if name == "getAnnotation" {
					args = append([]ast.Expr{intrinsicExecutionExpr(ctx)}, args...)
				}
				return selectorCall(recv, goName, args)
			})
			if name == "getAnnotation" {
				registerInstanceNodeIntrinsic(receiver, name, reflectionAnnotationIntrinsic)
			}
			if receiver == "Class" && name == "getDeclaredConstructor" {
				registerInstanceNodeIntrinsic(receiver, name, reflectionVarargsIntrinsic(receiver, name))
			}
			registerInstanceIntrinsicResultType(receiver, name, result)
		}
	}
	// The existing get path already threads Execution. Mutation must retain it too,
	// including private field access and declaring-class static initialization.
	registerInstanceIntrinsic("Field", "set", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		return selectorCall(recv, "SetExecution", append([]ast.Expr{intrinsicExecutionExpr(ctx)}, args...))
	})
	registerInstanceIntrinsicResultType("Field", "set", "void")
	for name, result := range map[string]string{"getLong": "long", "setLong": "void"} {
		goName := map[string]string{"getLong": "GetLongExecution", "setLong": "SetLongExecution"}[name]
		registerInstanceIntrinsic("Field", name, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			return selectorCall(recv, goName, append([]ast.Expr{intrinsicExecutionExpr(ctx)}, args...))
		})
		registerInstanceIntrinsicResultType("Field", name, result)
	}
}
