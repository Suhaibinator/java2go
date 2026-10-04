package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
)

func reflectRuntimeTypeExpr(baseName string, ctx Ctx) (ast.Expr, bool) {
	if name := builtinReflectProtocol(baseName, ctx); name != "" {
		if name == "Type" {
			name = "ReflectType"
		}
		return stdjavaQualifiedExpr(name, ctx), true
	}
	return nil, false
}

func init() {
	for _, receiver := range []string{"Type", "Class", "ParameterizedType", "GenericArrayType", "WildcardType", "TypeVariable"} {
		registerInstanceIntrinsic(receiver, "getTypeName", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaCall(ctx, "ReflectTypeNameJavaStringExecution", intrinsicExecutionExpr(ctx), recv)
		})
		registerInstanceIntrinsicResultType(receiver, "getTypeName", "java.lang.String")
		registerInstanceIntrinsic(receiver, "getClass", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaCall(ctx, "ObjectGetClass", recv)
		})
		registerInstanceIntrinsicResultType(receiver, "getClass", "Class")
	}
}

func init() {
	type accessor struct{ receiver, java, goName, result, helper, component string }
	accessors := []accessor{
		{"ParameterizedType", "getActualTypeArguments", "GetActualTypeArguments", "Type[]", "ReflectArrayMemberExecution", "ReflectTypeTypeID"},
		{"ParameterizedType", "getRawType", "GetRawType", "Type", "ReflectTypeMemberExecution", ""},
		{"ParameterizedType", "getOwnerType", "GetOwnerType", "Type", "ReflectTypeMemberExecution", ""},
		{"GenericArrayType", "getGenericComponentType", "GetGenericComponentType", "Type", "ReflectTypeMemberExecution", ""},
		{"WildcardType", "getUpperBounds", "GetUpperBounds", "Type[]", "ReflectArrayMemberExecution", "ReflectTypeTypeID"},
		{"WildcardType", "getLowerBounds", "GetLowerBounds", "Type[]", "ReflectArrayMemberExecution", "ReflectTypeTypeID"},
		{"TypeVariable", "getBounds", "GetBounds", "Type[]", "ReflectArrayMemberExecution", "ReflectTypeTypeID"},
		{"TypeVariable", "getName", "GetName", "java.lang.String", "ReflectStringMemberJavaStringExecution", ""},
		{"TypeVariable", "getGenericDeclaration", "GetGenericDeclaration", "GenericDeclaration", "ReflectDeclarationMemberExecution", ""},
		{"GenericDeclaration", "getTypeParameters", "GetTypeParameters", "TypeVariable<?>[]", "ReflectArrayMemberExecution", "TypeVariableTypeID"},
	}
	for _, a := range accessors {
		registerInstanceIntrinsic(a.receiver, a.java, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			values := []ast.Expr{intrinsicExecutionExpr(ctx), recv, stdjavaQualifiedExpr(reflectProtocolConstants[a.receiver], ctx), &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(a.goName)}}
			if a.component != "" {
				values = append(values, stdjavaQualifiedExpr(a.component, ctx))
			}
			return stdjavaCall(ctx, a.helper, values...)
		})
		registerInstanceIntrinsicResultType(a.receiver, a.java, a.result)
	}
}
