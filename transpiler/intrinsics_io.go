package transpiler

import "go/ast"

// This file registers the java.io / java.util.Scanner intrinsics, mapping them
// onto the os/bufio-backed shims in stdjava/io.go.
//
// Constructors:
//   new File(path)                              -> stdjava.NewJavaFile(path)
//   new PrintWriter(path) / new FileWriter(path)-> stdjava.NewPrintWriter(path)
//   new BufferedReader(new FileReader(path))    -> stdjava.NewBufferedReader(path)
//   new Scanner(System.in)                      -> stdjava.NewScannerStdin()
//   new Scanner(new File(path))                 -> stdjava.NewScannerFile(path)

func init() {
	registerIOConstructors()
	registerIOInstanceMethods()
}

func registerIOConstructors() {
	registerConstructorIntrinsic("File", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "NewJavaFile", args[0])
	})

	for _, name := range []string{"PrintWriter", "FileWriter"} {
		registerConstructorIntrinsic(name, func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 1 {
				return nil
			}
			// `new PrintWriter(new File(path))` nests a File; unwrap to the path.
			return stdjavaCall(ctx, "NewPrintWriter", unwrapFileToPath(args[0]))
		})
	}

	// `new BufferedReader(new FileReader(path))` — unwrap the inner FileReader to
	// its path. A FileReader/Reader argument that is not a recognized wrapper
	// falls through.
	registerConstructorIntrinsic("BufferedReader", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		path, ok := readerArgToPath(args[0])
		if !ok {
			return nil
		}
		return stdjavaCall(ctx, "NewBufferedReader", path)
	})
	// A bare `new FileReader(path)` used directly as a reader maps to a
	// BufferedReader so readLine() is available.
	registerConstructorIntrinsic("FileReader", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "NewBufferedReader", unwrapFileToPath(args[0]))
	})

	registerConstructorIntrinsic("Scanner", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		if isSystemInExpr(args[0]) {
			return stdjavaCall(ctx, "NewScannerStdin")
		}
		// new Scanner(new File(path)) -> NewScannerFile(path).
		if path, ok := newFilePathArg(args[0]); ok {
			return stdjavaCall(ctx, "NewScannerFile", path)
		}
		return nil
	})
}

func registerIOInstanceMethods() {
	// java.io.File
	registerInstanceIntrinsic("File", "exists", ioMethod("Exists", 0))
	registerInstanceIntrinsic("File", "getName", ioMethod("GetName", 0))
	registerInstanceIntrinsic("File", "getPath", ioMethod("GetPath", 0))
	registerInstanceIntrinsic("File", "isDirectory", ioMethod("IsDirectory", 0))
	registerInstanceIntrinsic("File", "length", ioMethod("Length", 0))
	registerInstanceIntrinsic("File", "delete", ioMethod("Delete", 0))

	// PrintWriter / FileWriter
	for _, t := range []string{"PrintWriter", "FileWriter"} {
		registerInstanceIntrinsic(t, "print", ioMethod("Print", 1))
		registerInstanceIntrinsic(t, "println", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			switch len(args) {
			case 0:
				return methodCall(recv, "PrintlnEmpty")
			case 1:
				return methodCall(recv, "Println", args[0])
			}
			return nil
		})
		registerInstanceIntrinsic(t, "write", ioMethod("Print", 1))
		registerInstanceIntrinsic(t, "flush", ioMethod("Flush", 0))
		registerInstanceIntrinsic(t, "close", ioMethod("Close", 0))
	}

	// BufferedReader / FileReader
	for _, t := range []string{"BufferedReader", "FileReader"} {
		registerInstanceIntrinsic(t, "close", ioMethod("Close", 0))
		// readLine returns (string, bool); transpiled code that assigns it expects a
		// single value. Emitting the method call keeps the two-value shape, which
		// only fits the `(line, ok)` idiom; documented as an approximation and left
		// for the call site. For the common `while ((line = r.readLine()) != null)`
		// pattern the transpiler would need a loop rewrite, which is out of scope
		// here, so readLine is emitted as a method call returning the line.
		registerInstanceIntrinsic(t, "readLine", ioMethod("ReadLine", 0))
	}

	// java.util.Scanner
	registerInstanceIntrinsic("Scanner", "nextInt", ioMethod("NextInt", 0))
	registerInstanceIntrinsic("Scanner", "nextLong", ioMethod("NextLong", 0))
	registerInstanceIntrinsic("Scanner", "nextDouble", ioMethod("NextDouble", 0))
	registerInstanceIntrinsic("Scanner", "next", ioMethod("Next", 0))
	registerInstanceIntrinsic("Scanner", "nextLine", ioMethod("NextLine", 0))
	registerInstanceIntrinsic("Scanner", "hasNext", ioMethod("HasNext", 0))
	registerInstanceIntrinsic("Scanner", "hasNextInt", ioMethod("HasNext", 0))
	registerInstanceIntrinsic("Scanner", "hasNextLine", ioMethod("HasNext", 0))
	registerInstanceIntrinsic("Scanner", "close", ioMethod("Close", 0))
}

// ioMethod builds a generator emitting a method call on the receiver when the
// argument count matches.
func ioMethod(goName string, argc int) intrinsicGenerator {
	return func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, argc) {
			return nil
		}
		return methodCall(recv, goName, args...)
	}
}

// isSystemInExpr reports whether a parsed expression is System.in.
func isSystemInExpr(arg ast.Expr) bool {
	sel, ok := arg.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "in" {
		return false
	}
	base, ok := sel.X.(*ast.Ident)
	return ok && base.Name == "System"
}

// unwrapFileToPath returns the inner path expression if arg is a
// stdjava.NewJavaFile(path) call (i.e. a `new File(path)`), else arg unchanged.
func unwrapFileToPath(arg ast.Expr) ast.Expr {
	if path, ok := newFilePathArg(arg); ok {
		return path
	}
	return arg
}

// newFilePathArg returns the path argument if arg is a stdjava.NewJavaFile(path)
// call.
func newFilePathArg(arg ast.Expr) (ast.Expr, bool) {
	call, ok := arg.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return nil, false
	}
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil && sel.Sel.Name == "NewJavaFile" {
		return call.Args[0], true
	}
	return nil, false
}

// readerArgToPath extracts the path from a reader argument that is a
// stdjava.NewBufferedReader(path) call (i.e. a `new FileReader(path)` already
// rewritten) or a `new File(path)`.
func readerArgToPath(arg ast.Expr) (ast.Expr, bool) {
	call, ok := arg.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return nil, false
	}
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
		switch sel.Sel.Name {
		case "NewBufferedReader", "NewJavaFile":
			return call.Args[0], true
		}
	}
	return nil, false
}
