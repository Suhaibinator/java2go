package transpiler

import "go/ast"

// digestIORuntimeTypeExpr reuses the existing IO runtime at declared parameter
// and result boundaries, not only at constructor call sites. Source classes
// with these simple names retain their own representation.
func digestIORuntimeTypeExpr(baseName string, ctx Ctx) (ast.Expr, bool) {
	if resolveClassScopeByQualifiedName(ctx, baseName) != nil {
		return nil, false
	}
	base := stripJavaQualifier(baseName)
	switch base {
	case "InputStream", "OpenOption", "StandardOpenOption":
		return stdjavaQualifiedExpr(base, ctx), true
	}
	name := ""
	switch base {
	case "File":
		name = "JavaFile"
	case "Path":
		name = "JavaPath"
	case "FileInputStream", "ByteArrayInputStream", "BufferedInputStream", "RandomAccessFile", "FileChannel":
		name = base
	}
	if name == "" {
		return nil, false
	}
	return &ast.StarExpr{X: stdjavaQualifiedExpr(name, ctx)}, true
}
func digestIOAssignable(actual, expected string) bool {
	actual, expected = stripJavaQualifier(actual), stripJavaQualifier(expected)
	if expected == "InputStream" {
		switch actual {
		case "BufferedInputStream", "FileInputStream", "ByteArrayInputStream":
			return true
		}
	}
	return expected == "OpenOption" && actual == "StandardOpenOption"
}

// Called after the existing IO registrations so the read dispatcher extends
// their no-argument overload without replacing any unrelated writer behavior.
func registerDigestIOIntrinsics() {
	registerConstructorIntrinsic("BufferedInputStream", func(_ []ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 && len(args) != 2 {
			return nil
		}
		return stdjavaCall(ctx, "NewBufferedInputStream", args...)
	})
	registerConstructorIntrinsic("RandomAccessFile", func(_ []ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 2 {
			return nil
		}
		return stdjavaCall(ctx, "NewRandomAccessFile", args...)
	})
	for _, name := range []string{"InputStream", "FileInputStream", "ByteArrayInputStream", "BufferedInputStream", "RandomAccessFile"} {
		registerInstanceIntrinsic(name, "read", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			all := append([]ast.Expr{recv}, args...)
			switch len(args) {
			case 0:
				if name != "InputStream" {
					return methodCall(recv, "ReadByteValue")
				}
				return stdjavaCall(ctx, "InputStreamReadByte", all...)
			case 1, 3:
				return stdjavaCall(ctx, "InputStreamReadInto", all...)
			}
			return nil
		})
		registerInstanceIntrinsicResultType(name, "read", "int")
		registerInstanceIntrinsic(name, "close", ioMethod("Close", 0))
		if name != "RandomAccessFile" {
			registerInstanceIntrinsic(name, "readAllBytes", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
				if len(args) != 0 {
					return nil
				}
				if name != "InputStream" {
					return methodCall(recv, "ReadAllBytes")
				}
				return stdjavaCall(ctx, "InputStreamReadAllBytes", recv)
			})
			registerInstanceIntrinsicResultType(name, "readAllBytes", "byte[]")
		}
	}
	registerStaticIntrinsic("Files", "newInputStream", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) < 1 {
			return nil
		}
		return stdjavaCall(ctx, "FilesNewInputStream", args...)
	})
	registerStaticIntrinsicResultType("Files", "newInputStream", "InputStream")
	for _, entry := range []struct{ java, runtime string }{{"READ", "Read"}, {"WRITE", "Write"}, {"APPEND", "Append"}, {"CREATE", "Create"}, {"CREATE_NEW", "CreateNew"}, {"TRUNCATE_EXISTING", "TruncateExisting"}, {"DELETE_ON_CLOSE", "DeleteOnClose"}, {"SPARSE", "Sparse"}, {"SYNC", "Sync"}, {"DSYNC", "Dsync"}} {
		name := entry.runtime
		registerStaticFieldIntrinsic("StandardOpenOption", entry.java, func(ctx Ctx) ast.Expr { return stdjavaQualifiedExpr("StandardOpenOption"+name, ctx) })
		registerStaticFieldIntrinsicResultType("StandardOpenOption", entry.java, "StandardOpenOption")
	}
	for _, entry := range []struct {
		class, java, runtime, result string
		argc                         int
	}{
		{"RandomAccessFile", "getChannel", "GetChannel", "FileChannel", 0},
		{"RandomAccessFile", "seek", "Seek", "", 1},
		{"RandomAccessFile", "getFilePointer", "GetFilePointer", "long", 0},
		{"RandomAccessFile", "length", "Length", "long", 0},
		{"FileChannel", "read", "Read", "int", 1},
		{"FileChannel", "close", "Close", "", 0},
		{"FileChannel", "isOpen", "IsOpen", "boolean", 0},
		{"ByteBuffer", "flip", "Flip", "ByteBuffer", 0},
		{"ByteBuffer", "clear", "Clear", "ByteBuffer", 0},
	} {
		registerInstanceIntrinsic(entry.class, entry.java, ioMethod(entry.runtime, entry.argc))
		if entry.result != "" {
			registerInstanceIntrinsicResultType(entry.class, entry.java, entry.result)
		}
	}
}
