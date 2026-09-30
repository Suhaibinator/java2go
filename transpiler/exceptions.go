package transpiler

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// builtinExceptionTypes is the set of java.lang / java.io exception classes that
// the stdjava runtime models directly. Constructing one of these with `new`
// produces the corresponding stdjava value, and the names double as the parent
// links recognised by stdjava.CaughtAs.
var builtinExceptionTypes = map[string]string{
	"Throwable":                       "",
	"ParseException":                  "Exception",
	"ExecutionException":              "Exception",
	"TimeoutException":                "Exception",
	"CancellationException":           "IllegalStateException",
	"RejectedExecutionException":      "RuntimeException",
	"IllegalThreadStateException":     "IllegalArgumentException",
	"Error":                           "Throwable",
	"AssertionError":                  "Error",
	"LinkageError":                    "Error",
	"ExceptionInInitializerError":     "LinkageError",
	"NoClassDefFoundError":            "LinkageError",
	"Exception":                       "Throwable",
	"RuntimeException":                "Exception",
	"IllegalArgumentException":        "RuntimeException",
	"IllegalStateException":           "RuntimeException",
	"IllegalMonitorStateException":    "RuntimeException",
	"NullPointerException":            "RuntimeException",
	"NegativeArraySizeException":      "RuntimeException",
	"IndexOutOfBoundsException":       "RuntimeException",
	"ArrayIndexOutOfBoundsException":  "IndexOutOfBoundsException",
	"StringIndexOutOfBoundsException": "IndexOutOfBoundsException",
	"ArrayStoreException":             "RuntimeException",
	"NumberFormatException":           "IllegalArgumentException",
	"ArithmeticException":             "RuntimeException",
	"ClassCastException":              "RuntimeException",
	"UnsupportedOperationException":   "RuntimeException",
	"IOException":                     "Exception",
	"UnsupportedEncodingException":    "IOException",
	"NoSuchAlgorithmException":        "Exception",
	"NoSuchElementException":          "RuntimeException",
	"ConcurrentModificationException": "RuntimeException",
}

// isBuiltinExceptionType reports whether className (qualified or not) names one
// of the stdjava-modelled exception types.
func isBuiltinExceptionType(className string) bool {
	base, _ := parseJavaTypeString(className)
	_, ok := builtinExceptionTypes[stripJavaQualifier(base)]
	return ok
}

// builtinExceptionConstructorExpr builds a call to the stdjava constructor for a
// built-in exception type, e.g. stdjava.NewIllegalArgumentException(args). The
// Types with modeled cause overloads preserve the entire Java argument list.
func builtinExceptionConstructorExpr(className string, arguments *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
	if call := canonicalThrowableConstructorExpr(className, arguments, args, ctx, source); call != nil {
		return call
	}
	if owner, ok := canonicalIntrinsicOwner(className, ctx); ok && owner == "java.lang.AssertionError" {
		return stdjavaCall(ctx, "NewJavaAssertionErrorExecution", append([]ast.Expr{intrinsicExecutionExpr(ctx)}, args...)...)
	}

	name := stripJavaQualifier(className)
	switch name {
	case "Throwable", "Exception", "RuntimeException", "IllegalArgumentException", "IllegalStateException", "UnsupportedEncodingException":
		return stdjavaCall(ctx, "New"+name+"Execution", append([]ast.Expr{intrinsicExecutionExpr(ctx)}, args...)...)
	case "ParseException":
		return &ast.CallExpr{Fun: stdjavaQualifiedExpr("New"+name, ctx), Args: args}
	}
	var message ast.Expr = &ast.BasicLit{Kind: token.STRING, Value: `""`}
	if len(args) > 0 {
		message = args[0]
	}
	return &ast.CallExpr{
		Fun:  stdjavaQualifiedExpr("New"+name, ctx),
		Args: []ast.Expr{message},
	}
}

// canonicalThrowableConstructorExpr selects modeled Java overloads from their
// static argument types before values are erased. In particular, String-null
// leaves initCause available while Throwable-null initializes the cause slot.
// Additional builtin constructor families are separate runtime ABI migrations.
func canonicalThrowableConstructorExpr(className string, arguments *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
	name := stripJavaQualifier(className)
	if name != "Exception" && name != "RuntimeException" && name != "IllegalStateException" && name != "IllegalArgumentException" {
		return nil
	}
	if _, builtin := builtinExceptionStorageTypeName(className, ctx); !builtin {
		return nil
	}
	if len(args) == 0 {
		return stdjavaCall(ctx, "NewJava"+name+"Message", ast.NewIdent("nil"))
	}
	if arguments == nil || int(arguments.NamedChildCount()) != len(args) {
		return nil
	}
	if (name == "Exception" || name == "IllegalArgumentException") && len(args) == 2 {
		// The two-argument declaration is (String, Throwable); coerce a null literal
		// according to that declaration instead of interpreting its runtime value.
		message := stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{javaStringReferenceType(ctx)}, []ast.Expr{args[0], stdjavaQualifiedExpr("StringTypeID", ctx)})
		return stdjavaCall(ctx, "NewJava"+name+"MessageCause", message, args[1])
	}
	actual, known := inferExprJavaType(arguments.NamedChild(0), ctx, source)
	if !known {
		return nil
	}
	if len(args) == 1 {
		if throwableConstructorMessageType(symbol.JavaType{Original: actual}, ctx, map[typeParameterIdentityKey]bool{}) {
			message := args[0]
			if !isBuiltinJavaString(actual, ctx) {
				// Bound parameters can use an erased Go representation. The
				// nominal view preserves both the original reference and null.
				message = stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{javaStringReferenceType(ctx)}, []ast.Expr{message, stdjavaQualifiedExpr("StringTypeID", ctx)})
			}
			return stdjavaCall(ctx, "NewJava"+name+"Message", message)
		}
		if name != "IllegalStateException" && javaExceptionReferenceAssignable(actual, "java.lang.Throwable", ctx) {
			return stdjavaCall(ctx, "NewJava"+name+"CauseExecution", intrinsicExecutionExpr(ctx), args[0])
		}
	}
	return nil
}

// A String-bounded parameter selects the String constructor declaration.
// Reuse declaration identities and each bound's lexical context; a source type
// or unrelated binder named String must never borrow the builtin overload.
func throwableConstructorMessageType(actual symbol.JavaType, ctx Ctx, visiting map[typeParameterIdentityKey]bool) bool {
	base, rank := javaArrayTypeParts(strings.TrimSpace(actual.Original))
	if rank != 0 {
		return false
	}
	if binding, found := resolveReferenceTypeParameter(actual, ctx); found {
		identity := identityKeyForTypeParameter(binding.parameter)
		if visiting[identity] {
			return false
		}
		visiting[identity] = true
		defer delete(visiting, identity)
		for _, bound := range binding.parameter.Bounds {
			if throwableConstructorMessageType(bound, binding.context, visiting) {
				return true
			}
		}
		return false
	}
	if actual.TypeParameterBindings[base] != nil {
		return false
	}
	return isBuiltinJavaString(base, ctx)
}

// exceptionSuperclassName returns the simple name of scope's superclass if that
// superclass is itself an exception type (either a stdjava built-in or another
// user-defined exception that transitively extends one). It returns "" when the
// class does not participate in the exception hierarchy.
func exceptionSuperclassName(ctx Ctx, scope *symbol.ClassScope) string {
	if scope == nil {
		return ""
	}
	super, _ := parseJavaTypeString(strings.TrimSpace(scope.Superclass))
	if super == "" {
		return ""
	}
	base := stripJavaQualifier(super)
	if isBuiltinExceptionType(base) {
		return base
	}
	if parentScope := resolveClassScopeByQualifiedName(ctx, super); parentScope != nil {
		if exceptionSuperclassName(ctx, parentScope) != "" {
			return base
		}
	}
	return ""
}

// isUserDefinedExceptionClass reports whether scope is a user class that extends
// (transitively) one of the modelled exception types.
func isUserDefinedExceptionClass(ctx Ctx, scope *symbol.ClassScope) bool {
	return exceptionSuperclassName(ctx, scope) != ""
}

// isExceptionJavaType reports whether the Java type named javaType is an
// exception type the stdjava runtime understands: a built-in exception or a
// user-defined class that extends one. It is used to route getMessage() /
// printStackTrace() through the runtime.
func isExceptionJavaType(ctx Ctx, javaType string) bool {
	base, _ := parseJavaTypeString(strings.TrimSpace(javaType))
	switch stripJavaQualifier(base) {
	case "ReflectiveOperationException", "ClassNotFoundException", "NoSuchMethodException", "NoSuchFieldException", "IllegalAccessException", "InvocationTargetException":
		return true
	}
	if isBuiltinExceptionType(base) {
		return true
	}
	if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil {
		return isUserDefinedExceptionClass(ctx, scope)
	}
	return false
}

// buildExceptionRegistrationDecl emits an init() function that registers a
// user-defined exception class with the stdjava hierarchy so catch-by-supertype
// dispatch recognises it:
//
//	func init() { stdjava.RegisterException("MyException", "RuntimeException") }
func buildExceptionRegistrationDecl(childName, parentName string, ctx Ctx) ast.Decl {
	declaration := &ast.FuncDecl{
		Name: &ast.Ident{Name: "init"},
		Type: &ast.FuncType{Params: &ast.FieldList{}},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: stdjavaQualifiedExpr("RegisterException", ctx),
						Args: []ast.Expr{
							&ast.BasicLit{Kind: token.STRING, Value: `"` + childName + `"`},
							&ast.BasicLit{Kind: token.STRING, Value: `"` + parentName + `"`},
						},
					},
				},
			},
		},
	}
	if registration := throwableMessageRegistration(ctx); registration != nil {
		declaration.Body.List = append(declaration.Body.List, registration)
	}
	for _, textMethod := range [][2]string{
		{"toString", "RegisterJavaThrowableToStringOverride"},
		{"getLocalizedMessage", "RegisterJavaThrowableLocalizedMessageOverride"},
	} {
		if registration := throwableTextOverrideRegistration(textMethod[0], textMethod[1], ctx); registration != nil {
			declaration.Body.List = append(declaration.Body.List, registration)
		}
	}
	if registration := throwableInitCauseRegistration(ctx); registration != nil {
		declaration.Body.List = append(declaration.Body.List, registration)
	}
	return declaration

}

// buildThrowableTypeNameMethod generates a ThrowableTypeName() method on the
// transpiled exception struct that returns the class's own Java name. This
// overrides the implementation promoted from the embedded stdjava base (which
// reports the parent type) so hierarchy matching identifies the concrete type.
func buildThrowableTypeNameMethod(ctx Ctx, javaName string) ast.Decl {
	recv := ShortName(ctx.className)
	return &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{{Name: recv}},
					Type:  &ast.StarExpr{X: &ast.Ident{Name: ctx.className}},
				},
			},
		},
		Name: &ast.Ident{Name: "ThrowableTypeName"},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{{Type: &ast.Ident{Name: "string"}}},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.BasicLit{Kind: token.STRING, Value: `"` + javaName + `"`},
					},
				},
			},
		},
	}
}

// throwsClauseComment parses a method's `throws` clause and renders it as a Go
// doc comment line. Full error-return translation is out of scope; preserving
// the clause as documentation keeps the original contract visible. It returns ""
// when the method declares no checked exceptions.
func throwsClauseComment(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	var throwsNode *sitter.Node
	for _, child := range nodeutil.NamedChildrenOf(node) {
		if child.Type() == "throws" {
			throwsNode = child
			break
		}
	}
	if throwsNode == nil {
		return ""
	}
	types := []string{}
	for _, child := range nodeutil.NamedChildrenOf(throwsNode) {
		switch child.Type() {
		case "type_identifier", "scoped_type_identifier", "generic_type":
			types = append(types, child.Content(source))
		}
	}
	if len(types) == 0 {
		return ""
	}
	return "// throws " + strings.Join(types, ", ")
}

// javaExceptionReferenceAssignable supplies the external superclass edges that
// are absent from source symbols. Source subclasses walk their declared parents;
// a source type shadowing a JDK name is never treated as that JDK class.
func javaExceptionReferenceAssignable(actual, expected string, ctx Ctx) bool {
	expectedName, known := builtinThrowableReferenceName(expected, ctx)
	if !known {
		return false
	}
	return throwableBoundReferenceAssignable(symbol.JavaType{Original: actual}, expectedName, ctx, map[typeParameterIdentityKey]bool{}, map[*symbol.ClassScope]bool{})
}

func isThrowableInitCauseOverride(method *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) bool {
	if method == nil || method.IsStatic || method.OriginalName != "initCause" || len(method.Parameters) != 1 {
		return false
	}
	declaring := classScopeCtx(owner, ctx)
	parameter := method.Parameters[0].OriginalType
	return (parameter == "Throwable" || parameter == "java.lang.Throwable") && resolveClassScopeByQualifiedName(declaring, parameter) == nil
}

func throwableInitCauseRegistration(ctx Ctx) ast.Stmt {
	for scope := ctx.currentClass; scope != nil; scope = resolveSuperclassScopeInDeclaringContext(ctx, scope) {
		for _, method := range scope.Methods {
			if !isThrowableInitCauseOverride(method, scope, ctx) {
				continue
			}
			receiverType := &ast.StarExpr{X: ast.NewIdent(ctx.className)}
			invoke := &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{
				executionParameterField("execution", ctx),
				{Names: []*ast.Ident{ast.NewIdent("receiver")}, Type: ast.NewIdent("any")},
				{Names: []*ast.Ident{ast.NewIdent("cause")}, Type: stdjavaQualifiedExpr("Throwable", ctx)},
			}}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.ReturnStmt{Results: []ast.Expr{&ast.CallExpr{Fun: &ast.SelectorExpr{X: &ast.TypeAssertExpr{X: ast.NewIdent("receiver"), Type: receiverType}, Sel: ast.NewIdent(executionImplementationName(method, scope, ctx))}, Args: []ast.Expr{ast.NewIdent("execution"), ast.NewIdent("cause")}}}},
			}}}
			return &ast.ExprStmt{X: stdjavaCall(ctx, "RegisterThrowableInitCause", &ast.CallExpr{Fun: receiverType, Args: []ast.Expr{ast.NewIdent("nil")}}, invoke)}
		}
	}
	return nil
}
