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
	registerIOStatics()
}

func registerIOStatics() {
	// File.createTempFile(prefix, suffix) -> stdjava.CreateTempFile(prefix, suffix).
	registerStaticIntrinsic("File", "createTempFile", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 2 {
			return nil
		}
		return stdjavaCall(ctx, "CreateTempFile", args[0], args[1])
	})
}

func registerIOConstructors() {
	registerConstructorIntrinsic("File", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "NewJavaFile", args[0])
	})

	// PrintWriter/FileWriter take a path, a File, or a nested writer. The stdjava
	// constructor accepts a path string or *JavaFile, so unwrap one writer layer
	// (new PrintWriter(new FileWriter(x)) -> x) and pass the result through.
	for _, name := range []string{"PrintWriter", "FileWriter"} {
		registerConstructorIntrinsic(name, func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 1 {
				return nil
			}
			return stdjavaCall(ctx, "NewPrintWriter", unwrapWriterArg(args[0]))
		})
	}

	// BufferedReader/FileReader take a path, a File, or a nested reader. Unwrap one
	// reader layer (new BufferedReader(new FileReader(x)) -> x) and pass through;
	// the stdjava constructor accepts a path string or *JavaFile.
	for _, name := range []string{"BufferedReader", "FileReader"} {
		registerConstructorIntrinsic(name, func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 1 {
				return nil
			}
			return stdjavaCall(ctx, "NewBufferedReader", unwrapReaderArg(args[0]))
		})
	}

	registerConstructorIntrinsic("Scanner", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		if isSystemInExpr(args[0]) {
			return stdjavaCall(ctx, "NewScannerStdin")
		}
		// new Scanner(new File(path)) -> NewScannerFile(path): unwrap the rewritten
		// File layer (NewScannerFile accepts a path string or *JavaFile).
		return stdjavaCall(ctx, "NewScannerFile", unwrapFileArg(args[0]))
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

// unwrapWriterArg unwraps one writer layer: if arg is a rewritten FileWriter
// (stdjava.NewPrintWriter(x)), return its inner x; otherwise return arg (a path
// string or a *JavaFile expression, both accepted by NewPrintWriter).
func unwrapWriterArg(arg ast.Expr) ast.Expr {
	if inner, ok := stdjavaCallArg(arg, "NewPrintWriter"); ok {
		return inner
	}
	return arg
}

// unwrapReaderArg unwraps one reader layer: if arg is a rewritten FileReader
// (stdjava.NewBufferedReader(x)), return its inner x; otherwise return arg.
func unwrapReaderArg(arg ast.Expr) ast.Expr {
	if inner, ok := stdjavaCallArg(arg, "NewBufferedReader"); ok {
		return inner
	}
	return arg
}

// unwrapFileArg unwraps one File layer: if arg is a rewritten File
// (stdjava.NewJavaFile(x)), return its inner x; otherwise return arg (a path
// string or a *JavaFile expression, both accepted by NewScannerFile).
func unwrapFileArg(arg ast.Expr) ast.Expr {
	if inner, ok := stdjavaCallArg(arg, "NewJavaFile"); ok {
		return inner
	}
	return arg
}

// stdjavaCallArg returns the single argument of arg if it is a call to
// stdjava.<name>(x).
func stdjavaCallArg(arg ast.Expr, name string) (ast.Expr, bool) {
	call, ok := arg.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return nil, false
	}
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil && sel.Sel.Name == name {
		return call.Args[0], true
	}
	return nil, false
}
