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
func characterIOSuperConstructor(args []ast.Expr, ctx Ctx, allocation ...ast.Expr) ast.Stmt {
	if canonicalReaderNativeEdge(ctx.currentClass, ctx) {
		var receiver ast.Expr = ast.NewIdent(ShortName(ctx.className))
		if len(allocation) > 0 && allocation[0] != nil {
			receiver = allocation[0]
		}
		if len(args) == 0 {
			self := receiver
			if len(allocation) > 1 && allocation[1] != nil {
				self = allocation[1]
			}
			args = []ast.Expr{self}
		}
		return &ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(superclassEmbeddedSelectorName(ctx.currentClass, ctx))}}, Tok: token.ASSIGN, Rhs: []ast.Expr{stdjavaCall(ctx, "NewReaderBase", args...)}}
	}

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
	var readerOperation string
	var expectedArgs []string
	if name == "read" {
		var owner *symbol.ClassScope
		if isSuper {
			owner = ctx.currentClass
		} else {
			base, _ := parseJavaTypeString(javaType)
			owner = resolveClassScopeByQualifiedName(ctx, base)
		}
		if readerNativeCarrierPath(owner, ast.NewIdent(sourceInstanceReceiverName(ctx)), ctx) != nil {
			readerOperation = readerNativeReadOperation(object, name, argsNode, source, ctx)
			if readerOperation == "" {
				return nil
			}
			switch readerOperation {
			case "ReaderReadArray":
				expectedArgs = []string{"char[]"}
			case "ReaderReadChars":
				expectedArgs = []string{"char[]", "int", "int"}
			}
		}
	}
	args := parseArgumentListWithExpectedTypes(argsNode, source, ctx, expectedArgs)
	var receiver ast.Expr
	if isSuper {
		receiver = ast.NewIdent(sourceInstanceReceiverName(ctx))
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
	if readerOperation != "" {
		runtime = readerOperation
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
	if has("Reader") {
		used := generatedInstanceUsedNames(ctx)
		recv := ast.NewIdent(affineUniqueLocalName(ShortName(ctx.className), used))
		if carrier := readerNativeCarrierPath(ctx.currentClass, recv, ctx); carrier != nil {
			declarations = append(declarations, &ast.FuncDecl{Name: ast.NewIdent("JavaReaderState"), Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{recv}, Type: &ast.StarExpr{X: instantiateGenericType(ctx.className, typeParamExprs(ctx.currentClass.GoTypeParameterNames()))}}}}, Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("ReaderBase", ctx)}}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{carrier}}}}})
		}
	}
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
	if name == "lock" {
		if access := unqualifiedReaderLockAccess(ctx); access != nil {
			return readerLockRead(access, nil, ctx)
		}
	}

	if name != "lock" || sourceCharacterIOBase(ctx.currentClass, ctx) != "Writer" {
		return nil
	}
	if ctx.localScope != nil && (ctx.localScope.ParameterByName(name) != nil || ctx.localScope.FindVariable(name) != nil) {
		return nil
	}
	if findFieldInHierarchy(ctx.currentClass, name, ctx) != nil {
		return nil
	}
	return &ast.SelectorExpr{X: stdjavaCall(ctx, "WriterState", ast.NewIdent(sourceInstanceReceiverName(ctx))), Sel: ast.NewIdent("Lock")}
}

// canonicalReaderNativeEdge selects only an unresolved java.io.Reader header
// edge. Resolve each source parent in its declaring context, never by short name.
func canonicalReaderNativeEdge(scope *symbol.ClassScope, ctx Ctx) bool {
	if scope == nil {
		return false
	}
	header := classHeaderTypeCtx(scope, ctx)
	base, _ := parseJavaTypeString(scope.Superclass)
	if resolveClassScopeByQualifiedName(header, base) != nil {
		return false
	}
	if base == "java.io.Reader" {
		return true
	}
	return base == "Reader" && header.currentFile != nil && header.currentFile.Imports["Reader"] == "java.io"
}

// readerNativeCarrierPath follows actual embedded source storage to the native
// carrier. Fields named lock do not change the carrier used by a super access.
func readerNativeCarrierPath(scope *symbol.ClassScope, receiver ast.Expr, ctx Ctx) ast.Expr {
	seen := map[*symbol.ClassScope]bool{}
	for scope != nil && !seen[scope] {
		seen[scope] = true
		if canonicalReaderNativeEdge(scope, ctx) {
			return &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(superclassEmbeddedSelectorName(scope, ctx))}
		}
		parent := resolveSuperclassScopeInDeclaringContext(ctx, scope)
		if parent == nil {
			return nil
		}
		receiver = &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(superclassEmbeddedSelectorName(scope, classHeaderTypeCtx(scope, ctx)))}
		scope = parent
	}
	return nil
}

func sourceSelectsReaderLock(scope *symbol.ClassScope, ctx Ctx) bool {
	seen := map[*symbol.ClassScope]bool{}
	for scope != nil && !seen[scope] {
		seen[scope] = true
		if scope.FindFieldByName("lock") != nil {
			return false
		}
		if canonicalReaderNativeEdge(scope, ctx) {
			return true
		}
		scope = resolveSuperclassScopeInDeclaringContext(ctx, scope)
	}
	return false
}

type readerLockAccess struct {
	object   *sitter.Node
	receiver ast.Expr
}

// Lookup is shared by reads, assignment and type inference. It performs no
// expression parsing, so checking a Java field fact cannot evaluate a receiver.
func selectedReaderLockAccess(node *sitter.Node, source []byte, ctx Ctx) *readerLockAccess {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return nil
	}
	if node.Type() == "identifier" && node.Content(source) == "lock" {
		return unqualifiedReaderLockAccess(ctx)
	}
	if node.Type() != "field_access" {
		return nil
	}
	field := node.ChildByFieldName("field")
	object := node.ChildByFieldName("object")
	if field == nil || object == nil || field.Content(source) != "lock" {
		return nil
	}
	var owner *symbol.ClassScope
	switch object.Type() {
	case "this":
		if ctx.localScope != nil && ctx.localScope.IsStatic {
			return nil
		}
		owner = ctx.currentClass
	case "super":
		if ctx.localScope != nil && ctx.localScope.IsStatic {
			return nil
		}
		owner = resolveSuperclassScopeInDeclaringContext(ctx, ctx.currentClass)
		if owner == nil && canonicalReaderNativeEdge(ctx.currentClass, ctx) {
			return &readerLockAccess{receiver: ast.NewIdent(sourceInstanceReceiverName(ctx))}
		}
	default:
		if resolveClassScopeByIdentifier(ctx, source, object) != nil {
			return nil
		}
		if javaType, ok := inferExprJavaType(object, ctx, source); ok {
			base, _ := parseJavaTypeString(javaType)
			owner = resolveClassScopeByQualifiedName(ctx, base)
		}
	}
	if !sourceSelectsReaderLock(owner, ctx) {
		return nil
	}
	return &readerLockAccess{object: object}
}

func unqualifiedReaderLockAccess(ctx Ctx) *readerLockAccess {
	if ctx.localScope != nil {
		if ctx.localScope.IsStatic || ctx.localScope.ParameterByName("lock") != nil || ctx.localScope.FindVariable("lock") != nil {
			return nil
		}
	}
	var receiver ast.Expr = ast.NewIdent(sourceInstanceReceiverName(ctx))
	seen := map[*symbol.ClassScope]bool{}
	for owner := ctx.currentClass; owner != nil && !seen[owner]; owner = owner.Enclosing {
		seen[owner] = true
		if sourceSelectsReaderLock(owner, ctx) {
			return &readerLockAccess{receiver: receiver}
		}
		if findFieldInHierarchy(owner, "lock", classHeaderTypeCtx(owner, ctx)) != nil || !owner.IsInner || owner.Enclosing == nil {
			return nil
		}
		receiver = &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(owner.EnclosingFieldName())}
	}
	return nil
}

func readerLockAccessReceiver(access *readerLockAccess, source []byte, ctx Ctx) ast.Expr {
	if access.receiver != nil {
		return access.receiver
	}
	return ParseExpr(access.object, source, ctx)
}

func readerLockRead(access *readerLockAccess, source []byte, ctx Ctx) ast.Expr {
	return &ast.SelectorExpr{X: stdjavaCall(ctx, "ReaderState", readerLockAccessReceiver(access, source, ctx)), Sel: ast.NewIdent("Lock")}
}

// Java evaluates a qualified receiver and RHS before checking the receiver for
// null. Delaying state lookup until after the RHS preserves exception order.
func lowerReaderLockAssignment(node *sitter.Node, source []byte, ctx Ctx) (ast.Expr, bool) {
	lhs, op, rhs, valid := assignmentExpressionNodes(node, source)
	if !valid || op.Content(source) != "=" {
		return nil, false
	}
	access := selectedReaderLockAccess(lhs, source, ctx)
	if access == nil {
		return nil, false
	}
	used := sourceInstanceReceiverUsedNames(ctx)
	receiverName := affineUniqueLocalName("__readerLockReceiver", used)
	valueName := affineUniqueLocalName("__readerLockValue", used)
	rhsCtx := ctx.Clone()
	rhsCtx.expectedType = "java.lang.Object"
	rhsCtx.expectedTypeRoot = rhs
	value := coerceArgumentToExpectedType(ParseExpr(rhs, source, rhsCtx), rhs, "java.lang.Object", ctx, source)
	valueType := javaTypeStringToGoTypeExpr("java.lang.Object", nil, ctx)
	receiver := ast.NewIdent(receiverName)
	stored := ast.NewIdent(valueName)
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: valueType}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{
		&ast.AssignStmt{Lhs: []ast.Expr{receiver}, Tok: token.DEFINE, Rhs: []ast.Expr{readerLockAccessReceiver(access, source, ctx)}},
		&ast.AssignStmt{Lhs: []ast.Expr{stored}, Tok: token.DEFINE, Rhs: []ast.Expr{&ast.CallExpr{Fun: valueType, Args: []ast.Expr{value}}}},
		&ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: stdjavaCall(ctx, "ReaderState", receiver), Sel: ast.NewIdent("Lock")}}, Tok: token.ASSIGN, Rhs: []ast.Expr{stored}},
		&ast.ReturnStmt{Results: []ast.Expr{stored}},
	}}}}, true
}

// One selector supplies both native read lowering and its Java result fact.
// Unsupported signatures keep ordinary source overload resolution.
func readerNativeReadOperation(object *sitter.Node, name string, arguments *sitter.Node, source []byte, ctx Ctx) string {
	if object == nil || name != "read" {
		return ""
	}
	var owner *symbol.ClassScope
	if object.Type() == "super" {
		owner = ctx.currentClass
	} else if javaType, ok := inferExprJavaType(object, ctx, source); ok {
		base, _ := parseJavaTypeString(javaType)
		owner = resolveClassScopeByQualifiedName(ctx, base)
	}
	if readerNativeCarrierPath(owner, ast.NewIdent(sourceInstanceReceiverName(ctx)), ctx) == nil {
		return ""
	}
	count := 0
	if arguments != nil {
		count = nodeutil.SemanticNamedChildCount(arguments)
	}
	if target := resolveInvocationTarget(object, ctx, source); target != nil {
		if selected, _ := findBestMethodForInvocationTarget(target, name, arguments, true, true, ctx, source); selected != nil && selected.def != nil {
			parameters := selected.def.Parameters
			if len(parameters) != count {
				return ""
			}
			for i, parameter := range parameters {
				expected := "int"
				if i == 0 {
					expected = "char[]"
				}
				if definitionJavaType(parameter) != expected {
					return ""
				}
			}
		}
	}
	if count == 0 {
		return "ReaderReadChar"
	}
	if count != 1 && count != 3 {
		return ""
	}
	first := nodeutil.SemanticNamedChild(arguments, 0)
	firstType, known := inferExprJavaType(first, ctx, source)
	if first.Type() != "null_literal" && (!known || firstType != "char[]") {
		return ""
	}
	if count == 1 {
		return "ReaderReadArray"
	}
	for i := 1; i < 3; i++ {
		argument := nodeutil.SemanticNamedChild(arguments, i)
		javaType, known := inferExprJavaType(argument, ctx, source)
		if !known {
			return ""
		}
		switch javaType {
		case "byte", "short", "char", "int":
		default:
			return ""
		}
	}
	return "ReaderReadChars"
}

func readerReadInvocationResultType(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	if node == nil || node.Type() != "method_invocation" {
		return "", false
	}
	name := node.ChildByFieldName("name")
	if name == nil || readerNativeReadOperation(node.ChildByFieldName("object"), name.Content(source), node.ChildByFieldName("arguments"), source, ctx) == "" {
		return "", false
	}
	return "int", true
}
