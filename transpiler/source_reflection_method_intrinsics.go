package transpiler

import "go/ast"

func init() {
	for name, result := range map[string]string{"isBridge": "boolean", "isSynthetic": "boolean", "getReturnType": "java.lang.Class<?>", "getDeclaringClass": "java.lang.Class<?>", "getModifiers": "int", "setAccessible": "void", "canAccess": "boolean"} {
		name := name
		goName := map[string]string{"isBridge": "IsBridge", "isSynthetic": "IsSynthetic", "getReturnType": "GetReturnType", "getDeclaringClass": "GetDeclaringClass", "getModifiers": "GetModifiers", "setAccessible": "SetAccessible", "canAccess": "CanAccess"}[name]
		registerInstanceIntrinsic("Method", name, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr { return selectorCall(recv, goName, args) })
		registerInstanceIntrinsicResultType("Method", name, result)
	}
	registerInstanceNodeIntrinsic("Class", "getDeclaredMethod", reflectionVarargsIntrinsic("Class", "getDeclaredMethod"))
	registerInstanceIntrinsic("Class", "getDeclaredMethod", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		return selectorCall(recv, "GetDeclaredMethodJavaString", args)
	})
	registerInstanceIntrinsicResultType("Class", "getDeclaredMethod", "java.lang.reflect.Method")
	registerInstanceIntrinsic("Class", "getDeclaredMethods", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		return selectorCall(recv, "GetDeclaredMethods", args)
	})
	registerInstanceIntrinsicResultType("Class", "getDeclaredMethods", "java.lang.reflect.Method[]")
}
