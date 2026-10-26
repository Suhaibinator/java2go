package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

func init() {
	for _, receiver := range []string{"Class", "Field", "Method", "Constructor", "AccessibleObject"} {
		registerInstanceNodeIntrinsic(receiver, "getAnnotation", reflectionAnnotationIntrinsic)
		registerInstanceIntrinsicResultType(receiver, "getAnnotation", "java.lang.annotation.Annotation")
		for _, method := range []string{"getAnnotations", "getDeclaredAnnotations"} {
			name := method
			registerInstanceIntrinsic(receiver, name, func(value ast.Expr, arguments []ast.Expr, ctx Ctx) ast.Expr {
				if len(arguments) != 0 {
					return nil
				}
				selected := "GetAnnotationsExecution"
				if receiver == "Class" && name == "getDeclaredAnnotations" {
					selected = "GetDeclaredAnnotationsExecution"
				}
				return selectorCall(value, selected, []ast.Expr{intrinsicExecutionExpr(ctx)})
			})
			registerInstanceIntrinsicResultType(receiver, name, "java.lang.annotation.Annotation[]")
		}
	}
	registerInstanceIntrinsic("Annotation", "annotationType", func(value ast.Expr, arguments []ast.Expr, ctx Ctx) ast.Expr {
		if len(arguments) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "AnnotationTypeExecution", intrinsicExecutionExpr(ctx), value)
	})
	registerInstanceIntrinsicResultType("Annotation", "annotationType", "java.lang.Class<? extends java.lang.annotation.Annotation>")
	registerInstanceIntrinsic("AccessibleObject", "isAccessible", func(value ast.Expr, arguments []ast.Expr, ctx Ctx) ast.Expr {
		if len(arguments) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "AccessibleObjectIsAccessible", value)
	})
	registerInstanceIntrinsicResultType("AccessibleObject", "isAccessible", "boolean")
	// Every admitted value getter/setter uses the same owned lexical caller
	// proof. The typed Long pair retains the existing primitive conversion and
	// volatile storage path; other typed primitive APIs remain unimplemented.
	for _, operation := range []struct{ name, legacy, caller, result string }{
		{"get", "GetExecution", "GetWithCallerExecution", "java.lang.Object"},
		{"set", "SetExecution", "SetWithCallerExecution", "void"},
		{"getLong", "GetLongExecution", "GetLongWithCallerExecution", "long"},
		{"setLong", "SetLongExecution", "SetLongWithCallerExecution", "void"},
	} {
		operation := operation
		registerInstanceNodeIntrinsic("Field", operation.name, func(value ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			args := intrinsicArgs(node.ChildByFieldName("object"), operation.name, source, ctx)
			caller, owned := reflectionCallerExpression(node, ctx)
			if !owned {
				if ctx.runtimeLocationProfile == sourceRuntimeLocationClasspath {
					return unsupportedIntrinsicValue(node, operation.result, source, ctx)
				}
				return selectorCall(value, operation.legacy, append([]ast.Expr{intrinsicExecutionExpr(ctx)}, args...))
			}
			return selectorCall(value, operation.caller, append([]ast.Expr{intrinsicExecutionExpr(ctx), caller}, args...))
		})
	}

	registerInstanceIntrinsic("Field", "toString", func(value ast.Expr, arguments []ast.Expr, ctx Ctx) ast.Expr {
		if len(arguments) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "FieldStringExecution", intrinsicExecutionExpr(ctx), value)
	})
	registerInstanceIntrinsicResultType("Field", "toString", "java.lang.String")

	registerReflectionAccessIntrinsics()

}
