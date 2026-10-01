package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/token"
)

func characterIOBaseTypeExpr(javaType string, ctx Ctx) ast.Expr {
	if resolveClassScopeByQualifiedName(ctx, javaType) != nil {
		return nil
	}
	base := stripJavaQualifier(javaType)
	if base == "Reader" || base == "Writer" {
		return &ast.StarExpr{X: stdjavaQualifiedExpr(base+"Base", ctx)}
	}
	return nil
}
func characterIOBuiltinProtocols(name string) []string {
	switch stripJavaQualifier(name) {
	case "Reader", "StringReader", "InputStreamReader", "BufferedReader", "FileReader":
		return []string{"Reader", "Closeable", "AutoCloseable"}
	case "Writer", "StringWriter", "BufferedWriter", "OutputStreamWriter", "PrintWriter", "FileWriter":
		return []string{"Writer", "Appendable", "Closeable", "AutoCloseable", "Flushable"}
	case "StringBuilder", "StringBuffer":
		return []string{"Appendable"}
	case "Closeable":
		return []string{"Closeable", "AutoCloseable"}
	case "Appendable", "Flushable":
		return []string{stripJavaQualifier(name)}
	}
	return nil
}
func sourceCharacterIOProtocols(scope *symbol.ClassScope, ctx Ctx) []string {
	seen := map[*symbol.ClassScope]bool{}
	protocols := map[string]bool{}
	var visit func(*symbol.ClassScope)
	visit = func(scope *symbol.ClassScope) {
		if scope == nil || seen[scope] {
			return
		}
		seen[scope] = true
		for _, name := range append([]string{scope.Superclass}, scope.ImplementedInterfaces...) {
			base, _ := parseJavaTypeString(name)
			if parent := resolveClassScopeByQualifiedName(ctx, base); parent != nil {
				visit(parent)
			} else {
				for _, protocol := range characterIOBuiltinProtocols(base) {
					protocols[protocol] = true
				}
			}
		}
	}
	visit(scope)
	var result []string
	for _, name := range []string{"Reader", "Writer", "Appendable", "Closeable", "Flushable", "AutoCloseable"} {
		if protocols[name] {
			result = append(result, name)
		}
	}
	return result
}
func sourceCharacterIOBase(scope *symbol.ClassScope, ctx Ctx) string {
	for _, name := range sourceCharacterIOProtocols(scope, ctx) {
		if name == "Reader" || name == "Writer" {
			return name
		}
	}
	return ""
}
func characterIOReferenceAssignable(actual, expected string, ctx Ctx) bool {
	if resolveClassScopeByQualifiedName(ctx, expected) != nil {
		return false
	}
	base, _ := parseJavaTypeString(actual)
	protocols := characterIOBuiltinProtocols(base)
	if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil {
		protocols = sourceCharacterIOProtocols(scope, ctx)
	}
	for _, name := range protocols {
		if name == stripJavaQualifier(expected) {
			return true
		}
	}
	return false
}
func characterIOSuperConstructor(args []ast.Expr, ctx Ctx) ast.Stmt {
	if ctx.currentClass == nil || characterIOBaseTypeExpr(ctx.currentClass.Superclass, ctx) == nil {
		return nil
	}
	return &ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: ast.NewIdent(ShortName(ctx.className)), Sel: ast.NewIdent(stripJavaQualifier(ctx.currentClass.Superclass) + "Base")}}, Tok: token.ASSIGN, Rhs: []ast.Expr{stdjavaCall(ctx, "New"+stripJavaQualifier(ctx.currentClass.Superclass)+"Base", args...)}}
}
func characterIOInvocation(object *sitter.Node, name string, argsNode *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	if object == nil {
		return nil
	}
	isSuper := object.Type() == "super" && ctx.currentClass != nil && characterIOBaseTypeExpr(ctx.currentClass.Superclass, ctx) != nil
	javaType, ok := inferExprJavaType(object, ctx, source)
	if !isSuper && (!ok || len(sourceCharacterIOProtocols(resolveClassScopeByQualifiedName(ctx, javaType), ctx)) == 0) {
		return nil
	}
	if argsNode != nil && nodeutil.SemanticNamedChildCount(argsNode) > 0 {
		first := nodeutil.SemanticNamedChild(argsNode, 0)
		argType, _ := inferExprJavaType(first, ctx, source)
		if name == "read" && argType != "char[]" && first.Type() != "null_literal" {
			return nil
		}
		if name == "write" {
			switch stripJavaQualifier(argType) {
			case "int", "char", "byte", "short", "String", "char[]":
			default:
				if first.Type() != "null_literal" {
					return nil
				}
			}
		}
	}
	args := parseArgumentListWithExpectedTypes(argsNode, source, ctx, nil)
	var receiver ast.Expr
	if isSuper {
		receiver = ast.NewIdent(ShortName(ctx.className))
	} else {
		receiver = ParseExpr(object, source, ctx)
	}
	runtime := ""
	switch name {
	case "read":
		switch len(args) {
		case 0:
			runtime = "ReaderReadChar"
		case 1:
			runtime = "ReaderReadArray"
		case 3:
			runtime = "ReaderReadChars"
		}
	case "write":
		runtime = "WriterWrite"
	case "append":
		if sourceCharacterIOBase(resolveClassScopeByQualifiedName(ctx, javaType), ctx) == "Writer" || isSuper {
			runtime = "WriterAppend"
		} else {
			runtime = "AppendableAppend"
		}
	case "flush":
		if len(args) == 0 {
			runtime = "FlushableFlush"
		}
	case "close":
		if len(args) == 0 {
			runtime = "CloseableClose"
		}
	}
	if runtime == "" {
		return nil
	}
	if isSuper {
		runtime += "Default"
	}
	return stdjavaCall(ctx, runtime+"Execution", append([]ast.Expr{intrinsicExecutionExpr(ctx), receiver}, args...)...)
}
func generateCharacterIOBridgeDecls(ctx Ctx) []ast.Decl {
	protocols := sourceCharacterIOProtocols(ctx.currentClass, ctx)
	if len(protocols) == 0 {
		return nil
	}
	has := func(name string) bool {
		for _, p := range protocols {
			if p == name {
				return true
			}
		}
		return false
	}
	type signature struct {
		java, bridge string
		params       []string
		result       string
	}
	var signatures []signature
	if has("Reader") {
		signatures = append(signatures, signature{"read", "JavaReaderReadChar", nil, "int"}, signature{"read", "JavaReaderReadArray", []string{"char[]"}, "int"}, signature{"read", "JavaReaderReadChars", []string{"char[]", "int", "int"}, "int"})
	}
	if has("Writer") {
		signatures = append(signatures, signature{"write", "JavaWriterWriteInt", []string{"int"}, "void"}, signature{"write", "JavaWriterWriteString", []string{"String"}, "void"}, signature{"write", "JavaWriterWriteStringRange", []string{"String", "int", "int"}, "void"}, signature{"write", "JavaWriterWriteArray", []string{"char[]"}, "void"}, signature{"write", "JavaWriterWriteChars", []string{"char[]", "int", "int"}, "void"})
	}
	if has("Appendable") {
		signatures = append(signatures, signature{"append", "JavaAppendChar", []string{"char"}, "Appendable"}, signature{"append", "JavaAppendSequence", []string{"CharSequence"}, "Appendable"}, signature{"append", "JavaAppendRange", []string{"CharSequence", "int", "int"}, "Appendable"})
	}
	if has("Closeable") {
		signatures = append(signatures, signature{"close", "JavaClose", nil, "void"})
	}
	if has("Flushable") {
		signatures = append(signatures, signature{"flush", "JavaFlush", nil, "void"})
	}
	var declarations []ast.Decl
	if has("Writer") {
		recv := ast.NewIdent(ShortName(ctx.className))
		stateCall := stdjavaCall(ctx, "EnsureWriterBase", &ast.UnaryExpr{Op: token.AND, X: &ast.SelectorExpr{X: recv, Sel: ast.NewIdent("WriterBase")}}, recv)
		declarations = append(declarations, &ast.FuncDecl{Name: ast.NewIdent("JavaWriterState"), Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{recv}, Type: &ast.StarExpr{X: instantiateGenericType(ctx.className, typeParamExprs(ctx.currentClass.GoTypeParameterNames()))}}}}, Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("WriterBase", ctx)}}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{stateCall}}}}})
	}
	for _, protocol := range protocols {
		if protocol == "AutoCloseable" {
			continue
		}
		declarations = append(declarations, &ast.FuncDecl{Name: ast.NewIdent("Java" + protocol + "Marker"), Recv: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: instantiateGenericType(ctx.className, typeParamExprs(ctx.currentClass.GoTypeParameterNames()))}}}}, Type: &ast.FuncType{Params: &ast.FieldList{}}, Body: &ast.BlockStmt{}})
	}
	for _, sig := range signatures {
		var method *symbol.Definition
		var owner *symbol.ClassScope
		for scope := ctx.currentClass; scope != nil; scope = resolveClassScopeByQualifiedName(ctx, scope.Superclass) {
			for _, candidate := range scope.Methods {
				if candidate.OriginalName != sig.java || candidate.IsStatic || len(candidate.Parameters) != len(sig.params) {
					continue
				}
				matches := true
				for i, p := range candidate.Parameters {
					if stripJavaQualifier(p.OriginalType) != sig.params[i] {
						matches = false
					}
				}
				if matches {
					method, owner = candidate, scope
					break
				}
			}
			if method != nil {
				break
			}
		}
		if method == nil {
			continue
		}
		recv := ast.NewIdent(ShortName(ctx.className))
		params := []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("execution")}, Type: &ast.StarExpr{X: stdjavaQualifiedExpr("Execution", ctx)}}}
		args := []ast.Expr{ast.NewIdent("execution")}
		for i, p := range sig.params {
			name := []string{"buffer", "offset", "length"}[i]
			params = append(params, &ast.Field{Names: []*ast.Ident{ast.NewIdent(name)}, Type: javaTypeStringToGoTypeExpr(p, nil, ctx)})
			args = append(args, ast.NewIdent(name))
		}
		call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: recv, Sel: ast.NewIdent(executionImplementationName(method, owner, ctx))}, Args: args}
		var results *ast.FieldList
		var body ast.Stmt = &ast.ExprStmt{X: call}
		if sig.result != "void" {
			results = &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(sig.result, nil, ctx)}}}
			body = &ast.ReturnStmt{Results: []ast.Expr{call}}
		}
		declarations = append(declarations, &ast.FuncDecl{Name: ast.NewIdent(sig.bridge), Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{recv}, Type: &ast.StarExpr{X: instantiateGenericType(ctx.className, typeParamExprs(ctx.currentClass.GoTypeParameterNames()))}}}}, Type: &ast.FuncType{Params: &ast.FieldList{List: params}, Results: results}, Body: &ast.BlockStmt{List: []ast.Stmt{body}}})
	}
	return declarations
}

func inheritedCharacterIOLock(name string, ctx Ctx) ast.Expr {
	if name != "lock" || sourceCharacterIOBase(ctx.currentClass, ctx) != "Writer" {
		return nil
	}
	if ctx.localScope != nil && (ctx.localScope.ParameterByName(name) != nil || ctx.localScope.FindVariable(name) != nil) {
		return nil
	}
	if findFieldInHierarchy(ctx.currentClass, name, ctx) != nil {
		return nil
	}
	return &ast.SelectorExpr{X: stdjavaCall(ctx, "WriterState", ast.NewIdent(ShortName(ctx.className))), Sel: ast.NewIdent("Lock")}
}
