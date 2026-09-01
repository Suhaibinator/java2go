package transpiler

import "go/ast"

// This file registers the java.io / java.nio.file / java.util.Scanner
// intrinsics, mapping them onto the os/bufio-backed shims in stdjava/io.go and
// stdjava/nio.go.
//
// Constructors:
//   new File(path)                              -> stdjava.NewJavaFile(path)
//   new PrintWriter(path) / new FileWriter(path)-> stdjava.NewPrintWriter(path)
//   new FileWriter(path, append)                -> stdjava.NewPrintWriterAppend(path, append)
//   new BufferedWriter(new FileWriter(path))    -> stdjava.NewBufferedWriter(path)
//   new BufferedReader(new FileReader(path))    -> stdjava.NewBufferedReader(path)
//   new BufferedReader(new InputStreamReader(System.in))
//                                               -> stdjava.NewBufferedReader(stdjava.NewInputStreamReaderStdin())
//   new Scanner(System.in)                      -> stdjava.NewScannerStdin()
//   new Scanner(new File(path))                 -> stdjava.NewScannerFile(path)
//   new FileInputStream / FileOutputStream / ByteArrayInputStream /
//   ByteArrayOutputStream / StringReader / StringWriter / PrintStream /
//   InputStreamReader / OutputStreamWriter      -> the matching stdjava shim
//
// Statics:
//   File.createTempFile(prefix, suffix)         -> stdjava.CreateTempFile(...)
//   Paths.get(...) / Path.of(...)               -> stdjava.PathsGet(...)
//   Files.<op>(...)                             -> stdjava.Files<Op>(...)
//
// The stdjava Files entry points and Path methods accept either a *JavaPath or a
// plain path string, so no coercion is needed at a call site whose argument is a
// String rather than a Path.

func init() {
	registerIOConstructors()
	registerIOInstanceMethods()
	registerIOStatics()
	registerNioIntrinsics()
}

func registerIOStatics() {
	// File.createTempFile(prefix, suffix) -> stdjava.CreateTempFile(prefix, suffix).
	registerStaticIntrinsic("File", "createTempFile", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 2 {
			return nil
		}
		return stdjavaCall(ctx, "CreateTempFile", args[0], args[1])
	})
	registerStaticIntrinsicResultType("File", "createTempFile", "File")
}

func registerIOConstructors() {
	registerConstructorIntrinsic("File", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "NewJavaFile", args[0])
	})

	// PrintWriter takes a path, a File, or a nested writer. The stdjava
	// constructor accepts a path string, a File/Path, or another stdjava writer,
	// so unwrap one writer layer (new PrintWriter(new FileWriter(x)) -> x) and pass
	// the result through.
	registerConstructorIntrinsic("PrintWriter", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "NewPrintWriter", unwrapWriterArg(args[0]))
	})

	// FileWriter shares PrintWriter's shim, plus Java's two-argument append form.
	registerConstructorIntrinsic("FileWriter", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		switch len(args) {
		case 1:
			return stdjavaCall(ctx, "NewPrintWriter", unwrapWriterArg(args[0]))
		case 2:
			return stdjavaCall(ctx, "NewPrintWriterAppend", unwrapWriterArg(args[0]), args[1])
		}
		return nil
	})

	registerConstructorIntrinsic("BufferedWriter", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "NewBufferedWriter", unwrapWriterArg(args[0]))
	})

	// OutputStreamWriter/PrintStream over System.out have no path to open, so they
	// bind to the stdout-specific shim; every other destination goes through the
	// polymorphic constructor.
	registerConstructorIntrinsic("OutputStreamWriter", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		if isSystemStreamExpr(args[0], "out") {
			return stdjavaCall(ctx, "NewOutputStreamWriterStdout")
		}
		return stdjavaCall(ctx, "NewOutputStreamWriter", unwrapWriterArg(args[0]))
	})

	registerConstructorIntrinsic("StringWriter", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "NewStringWriter")
	})

	registerConstructorIntrinsic("PrintStream", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "NewPrintStream", unwrapWriterArg(args[0]))
	})

	registerConstructorIntrinsic("FileOutputStream", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		switch len(args) {
		case 1:
			return stdjavaCall(ctx, "NewFileOutputStream", unwrapFileArg(args[0]))
		case 2:
			return stdjavaCall(ctx, "NewFileOutputStreamAppend", unwrapFileArg(args[0]), args[1])
		}
		return nil
	})

	registerConstructorIntrinsic("ByteArrayOutputStream", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "NewByteArrayOutputStream")
	})

	registerConstructorIntrinsic("FileInputStream", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "NewFileInputStream", unwrapFileArg(args[0]))
	})

	registerConstructorIntrinsic("ByteArrayInputStream", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "NewByteArrayInputStream", args[0])
	})

	registerConstructorIntrinsic("StringReader", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "NewStringReader", args[0])
	})

	// InputStreamReader over System.in is the console-reading idiom and has no
	// path to open, so it binds to the stdin-specific shim.
	registerConstructorIntrinsic("InputStreamReader", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		if isSystemStreamExpr(args[0], "in") {
			return stdjavaCall(ctx, "NewInputStreamReaderStdin")
		}
		return stdjavaCall(ctx, "NewInputStreamReader", unwrapFileArg(args[0]))
	})

	// BufferedReader/FileReader take a path, a File, or a nested reader. Unwrap one
	// reader layer (new BufferedReader(new FileReader(x)) -> x) and pass through;
	// the stdjava constructor accepts a path string, a File/Path, or another
	// stdjava reader such as the InputStreamReader shim.
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
		if isSystemStreamExpr(args[0], "in") {
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
	registerInstanceIntrinsic("File", "getAbsolutePath", ioMethod("GetAbsolutePath", 0))
	registerInstanceIntrinsic("File", "isDirectory", ioMethod("IsDirectory", 0))
	registerInstanceIntrinsic("File", "isFile", ioMethod("IsFile", 0))
	registerInstanceIntrinsic("File", "length", ioMethod("Length", 0))
	registerInstanceIntrinsic("File", "delete", ioMethod("Delete", 0))
	registerInstanceIntrinsic("File", "mkdir", ioMethod("Mkdir", 0))
	registerInstanceIntrinsic("File", "mkdirs", ioMethod("Mkdirs", 0))
	registerInstanceIntrinsic("File", "createNewFile", ioMethod("CreateNewFile", 0))
	registerInstanceIntrinsic("File", "toPath", ioMethod("ToPath", 0))
	registerInstanceIntrinsicResultType("File", "toPath", "Path")

	// PrintWriter / FileWriter / PrintStream share the print/println shim.
	for _, t := range []string{"PrintWriter", "FileWriter", "PrintStream"} {
		registerInstanceIntrinsic(t, "print", ioMethod("Print", 1))
		registerInstanceIntrinsic(t, "println", printlnMethod)
		registerInstanceIntrinsic(t, "flush", ioMethod("Flush", 0))
		registerInstanceIntrinsic(t, "close", ioMethod("Close", 0))
	}
	// Writer.write(String) prints its argument, but PrintStream.write(int) writes a
	// single byte; that overload is left unmodeled rather than lowered to Print,
	// which would emit the number instead.
	registerInstanceIntrinsic("PrintWriter", "write", ioMethod("Print", 1))
	registerInstanceIntrinsic("FileWriter", "write", ioMethod("Print", 1))

	// BufferedWriter / OutputStreamWriter / StringWriter are Writers without
	// print/println; Java's write(String) is the only text entry point.
	for _, t := range []string{"BufferedWriter", "OutputStreamWriter", "StringWriter"} {
		registerInstanceIntrinsic(t, "write", ioMethod("WriteString", 1))
		registerInstanceIntrinsic(t, "flush", ioMethod("Flush", 0))
		registerInstanceIntrinsic(t, "close", ioMethod("Close", 0))
	}
	registerInstanceIntrinsic("BufferedWriter", "newLine", ioMethod("NewLine", 0))
	registerInstanceIntrinsic("StringWriter", "toString", ioMethod("String", 0))
	registerInstanceIntrinsicResultType("StringWriter", "toString", "String")

	// Byte streams
	for _, t := range []string{"FileOutputStream", "ByteArrayOutputStream"} {
		registerInstanceIntrinsic(t, "write", ioMethod("WriteBytes", 1))
		registerInstanceIntrinsic(t, "flush", ioMethod("Flush", 0))
		registerInstanceIntrinsic(t, "close", ioMethod("Close", 0))
	}
	registerInstanceIntrinsic("ByteArrayOutputStream", "toByteArray", ioMethod("ToByteArray", 0))
	registerInstanceIntrinsic("ByteArrayOutputStream", "toString", ioMethod("String", 0))
	registerInstanceIntrinsic("ByteArrayOutputStream", "size", ioMethod("Size", 0))
	registerInstanceIntrinsic("ByteArrayOutputStream", "reset", ioMethod("Reset", 0))
	registerInstanceIntrinsicResultType("ByteArrayOutputStream", "toString", "String")

	for _, t := range []string{"FileInputStream", "ByteArrayInputStream"} {
		registerInstanceIntrinsic(t, "read", ioMethod("ReadByteValue", 0))
		registerInstanceIntrinsic(t, "readAllBytes", ioMethod("ReadAllBytes", 0))
		registerInstanceIntrinsic(t, "available", ioMethod("Available", 0))
		registerInstanceIntrinsic(t, "close", ioMethod("Close", 0))
	}

	// Character readers that are only ever wrapped; close is the one method
	// transpiled code calls on them directly.
	registerInstanceIntrinsic("StringReader", "read", ioMethod("ReadChar", 0))
	registerInstanceIntrinsic("StringReader", "close", ioMethod("Close", 0))
	registerInstanceIntrinsic("InputStreamReader", "close", ioMethod("Close", 0))

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
		registerInstanceIntrinsic(t, "ready", ioMethod("Ready", 0))
		registerInstanceIntrinsic(t, "lines", ioMethod("Lines", 0))
		registerInstanceIntrinsicResultType(t, "lines", "Stream<String>")
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

// --- java.nio.file ----------------------------------------------------------

// nioFilesMethods maps each modeled Files static onto its stdjava function name,
// the Java argument count it accepts, and the Java type of its result (empty
// when the result is void or a primitive the transpiler already infers).
var nioFilesMethods = []struct {
	javaName   string
	goName     string
	argc       int
	resultType string
}{
	{"readAllLines", "FilesReadAllLines", 1, "List<String>"},
	{"lines", "FilesLines", 1, "Stream<String>"},
	{"readString", "FilesReadString", 1, "String"},
	{"writeString", "FilesWriteString", 2, "Path"},
	{"write", "FilesWrite", 2, "Path"},
	{"exists", "FilesExists", 1, ""},
	{"createDirectories", "FilesCreateDirectories", 1, "Path"},
	{"createFile", "FilesCreateFile", 1, "Path"},
	{"delete", "FilesDelete", 1, ""},
	{"deleteIfExists", "FilesDeleteIfExists", 1, ""},
	{"size", "FilesSize", 1, "long"},
	{"isDirectory", "FilesIsDirectory", 1, ""},
	{"isRegularFile", "FilesIsRegularFile", 1, ""},
	{"copy", "FilesCopy", 2, "Path"},
	{"move", "FilesMove", 2, "Path"},
}

// nioPathMethods maps each modeled Path instance method onto its stdjava method
// name, argument count, and Java result type.
var nioPathMethods = []struct {
	javaName   string
	goName     string
	argc       int
	resultType string
}{
	{"getFileName", "GetFileName", 0, "Path"},
	{"getParent", "GetParent", 0, "Path"},
	{"toString", "ToString", 0, "String"},
	{"resolve", "Resolve", 1, "Path"},
	{"toAbsolutePath", "ToAbsolutePath", 0, "Path"},
	{"normalize", "Normalize", 0, "Path"},
	{"getNameCount", "GetNameCount", 0, ""},
	{"startsWith", "StartsWith", 1, ""},
	{"endsWith", "EndsWith", 1, ""},
	{"toFile", "ToFile", 0, "File"},
}

func registerNioIntrinsics() {
	// Paths.get(a, b, ...) / Path.of(a, b, ...) -> stdjava.PathsGet(a, b, ...).
	// Java's overload is varargs, so any non-empty argument list is accepted.
	pathsGet := func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) == 0 {
			return nil
		}
		return stdjavaCall(ctx, "PathsGet", args...)
	}
	registerStaticIntrinsic("Paths", "get", pathsGet)
	registerStaticIntrinsicResultType("Paths", "get", "Path")
	registerStaticIntrinsic("Path", "of", pathsGet)
	registerStaticIntrinsicResultType("Path", "of", "Path")

	for _, method := range nioFilesMethods {
		registerStaticIntrinsic("Files", method.javaName, filesFunction(method.goName, method.argc))
		if method.resultType != "" {
			registerStaticIntrinsicResultType("Files", method.javaName, method.resultType)
		}
	}

	for _, method := range nioPathMethods {
		registerInstanceIntrinsic("Path", method.javaName, ioMethod(method.goName, method.argc))
		if method.resultType != "" {
			registerInstanceIntrinsicResultType("Path", method.javaName, method.resultType)
		}
	}
}

// filesFunction builds a generator emitting stdjava.<goName>(args) for a Files
// static taking exactly argc Java arguments. The overloads that take a Charset
// or OpenOption/CopyOption varargs carry more arguments and so fall through
// rather than compiling to code that would silently ignore those options.
func filesFunction(goName string, argc int) intrinsicGenerator {
	return func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, argc) {
			return nil
		}
		return stdjavaCall(ctx, goName, args...)
	}
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

// printlnMethod handles both println() and println(value) on the print shims.
func printlnMethod(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
	switch len(args) {
	case 0:
		return methodCall(recv, "PrintlnEmpty")
	case 1:
		return methodCall(recv, "Println", args[0])
	}
	return nil
}

// isSystemStreamExpr reports whether a parsed expression is System.<field>, e.g.
// System.in or System.out.
func isSystemStreamExpr(arg ast.Expr, field string) bool {
	sel, ok := arg.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != field {
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
