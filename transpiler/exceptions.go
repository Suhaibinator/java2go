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
func builtinExceptionConstructorExpr(className string, args []ast.Expr, ctx Ctx) ast.Expr {
	name := stripJavaQualifier(className)
	switch name {
	case "Exception", "RuntimeException", "IllegalArgumentException", "IllegalStateException", "UnsupportedEncodingException":
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
	return &ast.FuncDecl{
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
	if resolveClassScopeByQualifiedName(ctx, expected) != nil || !isBuiltinExceptionType(expected) {
		return false
	}
	expected = stripJavaQualifier(expected)
	seen := map[string]bool{}
	for actual != "" && !seen[actual] {
		seen[actual] = true
		if scope := resolveClassScopeByQualifiedName(ctx, actual); scope != nil {
			ctx = classScopeCtx(scope, ctx)
			actual = strings.TrimSpace(scope.Superclass)
			continue
		}
		actual = stripJavaQualifier(actual)
		parent, known := builtinExceptionTypes[actual]
		if !known {
			return false
		}
		if actual == expected {
			return true
		}
		actual = parent
	}
	return false
}
