package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"

	sitter "github.com/smacker/go-tree-sitter"
)

func systemPropertyInvocationOwned(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil {
		return false
	}
	if object := node.ChildByFieldName("object"); object != nil {
		return platformTypeOwner(object.Content(source), object, ctx, source) == "java.lang.System"
	}
	imported := resolveStaticImportedMethod(node, ctx, source)
	return imported.source == nil && imported.intrinsic == "System"
}

func systemPropertyProfileRefusal(node *sitter.Node, ctx Ctx, source []byte, failure error) ast.Expr {
	diagnostic := Diagnostic{Kind: "runtime profile", ClassName: ctx.className, Message: failure.Error()}
	if node != nil {
		diagnostic.NodeType = node.Type()
		physical, offset := source, node.StartByte()
		if file := ctx.currentFile; file != nil && file.OriginalSource != nil && int(offset) < len(file.SourceOffsets) {
			physical, offset = file.OriginalSource, file.SourceOffsets[offset]
		}
		diagnostic.Line = physicalSourceLine(physical, offset)
	}
	diagnostic = recordUnsupportedDiagnostic(diagnostic, ctx)
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: javaStringReferenceType(ctx)}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: callIdent("panic", &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(diagnostic.String())})}}}}}
}

func init() {
	registerIntrinsicOwner("java.lang.System", true)
	for _, method := range []struct {
		name, helper string
		arities      []int
	}{
		{"getProperty", "SystemGetProperty", []int{1, 2}},
		{"setProperty", "SystemSetProperty", []int{2}},
		{"clearProperty", "SystemClearProperty", []int{1}},
	} {
		for _, arity := range method.arities {
			parameters := make([]string, arity)
			for i := range parameters {
				parameters[i] = "java.lang.String"
			}
			registerStaticIntrinsicImportSignature("System", method.name, parameters...)
		}
		registerStaticIntrinsicResultType("System", method.name, "java.lang.String")
		registerStaticIntrinsicExpectedArguments("System", method.name, func(node *sitter.Node, _ Ctx, _ []byte) []string {
			count := invocationArgumentCount(node)
			valid := false
			for _, arity := range method.arities {
				valid = valid || count == arity
			}
			if !valid {
				return nil
			}
			parameters := make([]string, count)
			for i := range parameters {
				parameters[i] = "java.lang.String"
			}
			return parameters
		})
		registerStaticNodeIntrinsic("System", method.name, func(node *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
			if !systemPropertyInvocationOwned(node, ctx, source) {
				return nil
			}
			count := len(args)
			valid := false
			for _, arity := range method.arities {
				valid = valid || count == arity
			}
			if !valid || invocationArgumentCount(node) != count {
				return unsupportedIntrinsicValue(node, "java.lang.String", source, ctx)
			}
			for i := range args {
				if !intrinsicInvocationConversionApplicable(invocationArgumentNode(node, i), "java.lang.String", ctx, source) {
					return unsupportedIntrinsicValue(node, "java.lang.String", source, ctx)
				}
			}
			profile, err := systemPropertyProfileExpr(ctx)
			if err != nil {
				return systemPropertyProfileRefusal(node, ctx, source, err)
			}
			lowered := []ast.Expr{profile}
			for i, arg := range args {
				lowered = append(lowered, coerceArgumentToExpectedType(arg, invocationArgumentNode(node, i), "java.lang.String", ctx, source))
			}
			helper := method.helper
			if method.name == "getProperty" && count == 2 {
				helper = "SystemGetPropertyDefault"
			}
			return stdjavaCall(ctx, helper, lowered...)
		})
	}
}
