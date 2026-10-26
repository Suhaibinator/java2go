package transpiler

import "go/ast"

func characterIORuntimeTypeExpr(baseName string, ctx Ctx) (ast.Expr, bool) {
	if resolveClassScopeByQualifiedName(ctx, baseName) != nil {
		return nil, false
	}
	switch stripJavaQualifier(baseName) {
	case "Reader", "Writer", "Appendable", "Closeable", "Flushable":
		return stdjavaQualifiedExpr(stripJavaQualifier(baseName), ctx), true
	}
	return nil, false
}
func registerCharacterIOIntrinsics() {
	registerStaticIntrinsic("Thread", "holdsLock", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "ThreadHoldsLockExecution", intrinsicExecutionExpr(ctx), args[0])
	})
	registerStaticIntrinsicResultType("Thread", "holdsLock", "boolean")
	registerCharacterWriterIntrinsics()
	registerInstanceIntrinsic("Reader", "read", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		name := ""
		switch len(args) {
		case 0:
			name = "ReaderReadCharExecution"
		case 1:
			name = "ReaderReadArrayExecution"
		case 3:
			name = "ReaderReadCharsExecution"
		}
		if name == "" {
			return nil
		}
		return stdjavaCall(ctx, name, append([]ast.Expr{intrinsicExecutionExpr(ctx), recv}, args...)...)
	})
	registerInstanceIntrinsicResultType("Reader", "read", "int")
	for _, name := range []string{"Reader", "Writer", "Closeable"} {
		registerInstanceIntrinsic(name, "close", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaCall(ctx, "CloseableCloseExecution", intrinsicExecutionExpr(ctx), recv)
		})
	}
}

func registerCharacterWriterIntrinsics() {
	for _, name := range []string{"Writer"} {
		registerInstanceIntrinsic(name, "write", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 1 && len(args) != 3 {
				return nil
			}
			return stdjavaCall(ctx, "WriterWriteExecution", append([]ast.Expr{intrinsicExecutionExpr(ctx), recv}, args...)...)
		})
		registerInstanceIntrinsic(name, "append", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			return stdjavaCall(ctx, "WriterAppendExecution", append([]ast.Expr{intrinsicExecutionExpr(ctx), recv}, args...)...)
		})
		registerInstanceIntrinsicResultType(name, "append", name)
	}
	registerInstanceIntrinsic("Appendable", "append", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		return stdjavaCall(ctx, "AppendableAppendExecution", append([]ast.Expr{intrinsicExecutionExpr(ctx), recv}, args...)...)
	})
	registerInstanceIntrinsicResultType("Appendable", "append", "Appendable")
	for _, name := range []string{"Writer", "Flushable"} {
		registerInstanceIntrinsic(name, "flush", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			return stdjavaCall(ctx, "FlushableFlushExecution", intrinsicExecutionExpr(ctx), recv)
		})
	}
}
