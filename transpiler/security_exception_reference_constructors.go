package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

// Security exception constructors select their JDK declarations before reference
// erasure. Keep their strict policy separate from the frozen CME family.
func canonicalSecurityExceptionConstructorExpr(className string, arguments *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
	name := stripJavaQualifier(className)
	if name != "NoSuchAlgorithmException" && name != "GeneralSecurityException" {
		return nil
	}
	if _, builtin := builtinExceptionStorageTypeName(className, ctx); !builtin {
		return nil
	}
	reject := func() ast.Expr { reportUnsupported("expression", arguments, source, ctx); return &ast.BadExpr{} }
	if (arguments == nil && len(args) != 0) || (arguments != nil && nodeutil.SemanticNamedChildCount(arguments) != len(args)) {
		return reject()
	}
	if len(args) == 0 {
		return stdjavaCall(ctx, "NewJava"+name+"Message", ast.NewIdent("nil"))
	}
	typeAt := func(index int) (string, bool) {
		return inferExprJavaType(nodeutil.SemanticNamedChild(arguments, index), ctx, source)
	}
	messageType := func(actual string) bool {
		return actual == "null" || throwableConstructorMessageType(symbol.JavaType{Original: actual}, ctx, map[typeParameterIdentityKey]bool{})
	}
	causeType := func(actual string) bool {
		_, rank := javaArrayTypeParts(strings.TrimSpace(actual))
		return rank == 0 && (actual == "null" || javaExceptionReferenceAssignable(actual, "java.lang.Throwable", ctx))
	}
	messageView := func(value ast.Expr) ast.Expr {
		return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{javaStringReferenceType(ctx)}, []ast.Expr{value, stdjavaQualifiedExpr("StringTypeID", ctx)})
	}
	if len(args) == 2 {
		message, messageKnown := typeAt(0)
		cause, causeKnown := typeAt(1)
		if !messageKnown || !causeKnown || !messageType(message) || !causeType(cause) {
			return reject()
		}
		return stdjavaCall(ctx, "NewJava"+name+"MessageCause", messageView(args[0]), args[1])
	}
	if len(args) != 1 {
		return reject()
	}
	actual, known := typeAt(0)
	// Both one-argument declarations accept bare null, so javac rejects it as
	// ambiguous. Typed null selects its static String or Throwable declaration.
	if !known || actual == "null" {
		return reject()
	}
	if messageType(actual) {
		message := args[0]
		if !isBuiltinJavaString(actual, ctx) {
			message = messageView(message)
		}
		return stdjavaCall(ctx, "NewJava"+name+"Message", message)
	}
	if causeType(actual) {
		return stdjavaCall(ctx, "NewJava"+name+"CauseExecution", intrinsicExecutionExpr(ctx), args[0])
	}
	return reject()
}
