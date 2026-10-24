package transpiler

import (
	"bytes"
	"go/ast"
	"go/token"
	"maps"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

type raf209ConstructorKind uint8

const (
	raf209Refused raf209ConstructorKind = iota
	raf209String
	raf209NativeFile
)

// Namespace sidecars must agree with the actual registered source header. The
// immutable input snapshot detects byte replacement; neither it nor these
// tuples establish declaration membership or expression semantics.
func raf209FileCoherent(file *symbol.FileScope) bool {
	if file == nil || file.SourceInputSnapshot().Validate() != nil {
		return false
	}
	root := file.ParsedRoot()
	if root == nil {
		return false
	}
	imports := map[string]string{}
	pkg := ""
	for _, node := range nodeutil.NamedChildrenOf(root) {
		switch node.Type() {
		case "package_declaration":
			if node.NamedChildCount() == 0 {
				return false
			}
			pkg = node.NamedChild(0).Content(file.Source)
		case "import_declaration":
			// Match ParseSymbols' ordinary single-type table exactly. Static
			// and on-demand imports remain actual AST namespace facts, but
			// neither imports its owner as an ordinary individual type name.
			wildcard, static := false, false
			for index := 0; index < int(node.ChildCount()); index++ {
				wildcard = wildcard || node.Child(index).Content(file.Source) == "*"
				static = static || node.Child(index).Type() == "static"
			}
			if wildcard || static {
				continue
			}
			if node.NamedChildCount() == 0 {
				return false
			}
			selected := node.NamedChild(0)
			name, scope := selected.ChildByFieldName("name"), selected.ChildByFieldName("scope")
			if name == nil || scope == nil {
				return false
			}
			imports[name.Content(file.Source)] = scope.Content(file.Source)
		}
	}
	return pkg == file.Package && maps.Equal(imports, file.Imports)
}

// An unresolved single-static import can name a member type as well as a
// value. Do not borrow the ordinary-import/default-owner fallback for a matching
// written type root. Static wildcard resolution on the legacy String route is
// a separate existing boundary; only new native File admission is held below.
func raf209StaticTypeConflict(typ string, ctx Ctx) bool {
	rootName := strings.Split(typ, ".")[0]
	for _, node := range nodeutil.NamedChildrenOf(ctx.currentFile.ParsedRoot()) {
		if node.Type() != "import_declaration" || node.NamedChildCount() == 0 {
			continue
		}
		static, wildcard := false, false
		for i := 0; i < int(node.ChildCount()); i++ {
			static = static || node.Child(i).Type() == "static"
			wildcard = wildcard || node.Child(i).Content(ctx.currentFile.Source) == "*"
		}
		if static && !wildcard {
			written := node.NamedChild(0).Content(ctx.currentFile.Source)
			if stripJavaQualifier(written) == rootName {
				return true
			}
		}
	}
	return false
}

func raf209HasStaticWildcard(ctx Ctx) bool {
	for _, node := range nodeutil.NamedChildrenOf(ctx.currentFile.ParsedRoot()) {
		if node.Type() != "import_declaration" {
			continue
		}
		static, wildcard := false, false
		for i := 0; i < int(node.ChildCount()); i++ {
			static = static || node.Child(i).Type() == "static"
			wildcard = wildcard || node.Child(i).Content(ctx.currentFile.Source) == "*"
		}
		if static && wildcard {
			return true
		}
	}
	return false
}

// A registered declaration and its actual syntax own the proof. A matching
// Java/Go spelling or a detached same-byte-range tree is not a certificate.
func raf209Owns(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil || ctx.currentClass == nil || ctx.currentClass.Class == nil || ctx.currentFile == nil {
		return false
	}
	file := declaredRefusalRegisteredScopeFile(ctx.currentClass, ctx)
	if file == nil || file != ctx.currentFile || !raf209FileCoherent(file) || !bytes.Equal(file.Source, source) || !readerSourceNodeWithin(node, file.ParsedRoot()) {
		return false
	}
	var owner *sitter.Node
	for at := node; at != nil; at = at.Parent() {
		switch at.Type() {
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
			owner = at
		case "object_creation_expression":
			if objectCreationClassBody(at) != nil {
				owner = at
			}
		}
		if owner != nil {
			break
		}
	}
	if owner == nil || !owner.Equal(ctx.currentClass.Class.DeclarationNode) {
		return false
	}
	if ctx.localScope != nil {
		registered := false
		for _, method := range ctx.currentClass.Methods {
			if method == ctx.localScope {
				registered = true
				break
			}
		}
		if !registered || !helpfulRegisteredMethodHeader(ctx.currentClass, ctx.localScope, ctx) || !readerSourceNodeWithin(node, ctx.localScope.DeclarationNode) {
			return false
		}
		// A descendant in another executable cannot borrow this method's
		// lexical bindings. Ordinary lambda lowering must supply its own proof.
		for at := node; at != nil && !at.Equal(ctx.localScope.DeclarationNode); at = at.Parent() {
			switch at.Type() {
			case "lambda_expression", "method_declaration", "constructor_declaration", "compact_constructor_declaration":
				return false
			}
		}
	}
	return true
}

// Inspect the written lexical headers as well as symbol bindings: deleting a
// sidecar must not turn a source type-parameter spelling into a JDK owner.
func raf209SyntaxParameter(name string, ctx Ctx) bool {
	var start *sitter.Node
	if ctx.localScope != nil {
		start = ctx.localScope.DeclarationNode
	}
	if start == nil && ctx.currentClass != nil && ctx.currentClass.Class != nil {
		start = ctx.currentClass.Class.DeclarationNode
	}
	if ctx.currentFile == nil {
		return false
	}
	for at := start; at != nil; at = at.Parent() {
		for _, parameter := range symbol.ExtractTypeParameters(at.ChildByFieldName("type_parameters"), ctx.currentFile.Source) {
			if parameter.Name == name {
				return true
			}
		}
	}
	return false
}

func raf209WrittenString(typ string, ctx Ctx) bool {
	if !raf209FileCoherent(ctx.currentFile) {
		return false
	}
	base, rank := javaArrayTypeParts(strings.TrimSpace(typ))
	_, args := parseJavaTypeString(base)
	if rank != 0 || len(args) != 0 || raf209StaticTypeConflict(base, ctx) {
		return false
	}
	if visibleTypeParameterDeclarationForJavaType(base, ctx) == nil {
		if raf209SyntaxParameter(strings.Split(base, ".")[0], ctx) {
			return false
		}
		return isBuiltinJavaString(base, ctx)
	}
	// Bounds are mutable symbol sidecars. Attest their registered class/method
	// headers and captured declaration bindings before following any bound chain.
	if _, ok := helpfulClassHeaderParameters(ctx.currentClass, ctx, map[*symbol.ClassScope]bool{}); !ok {
		return false
	}
	if ctx.localScope != nil && !helpfulRegisteredMethodHeader(ctx.currentClass, ctx.localScope, ctx) {
		return false
	}
	actual := symbol.JavaType{Original: base, TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}
	return raf209SelectedStringBound(actual, ctx, map[typeParameterIdentityKey]bool{})
}

// Follow only the selected declaration's actual bound chain. An unrelated
// visible parameter must not donate either a String proof or a refusal.
func raf209SelectedStringBound(actual symbol.JavaType, ctx Ctx, visiting map[typeParameterIdentityKey]bool) bool {
	if !raf209FileCoherent(ctx.currentFile) {
		return false
	}
	if binding, found := resolveReferenceTypeParameter(actual, ctx); found {
		key := identityKeyForTypeParameter(binding.parameter)
		if key.declaration == nil || visiting[key] {
			return false
		}
		declaring := binding.context
		if _, ok := helpfulClassHeaderParameters(declaring.currentClass, declaring, map[*symbol.ClassScope]bool{}); !ok {
			return false
		}
		if declaring.localScope != nil && !helpfulRegisteredMethodHeader(declaring.currentClass, declaring.localScope, declaring) {
			return false
		}
		registered := false
		for _, parameter := range declaring.currentClass.TypeParameters {
			registered = registered || parameter.Declaration == key.declaration
		}
		if declaring.localScope != nil {
			for _, parameter := range declaring.localScope.TypeParameters {
				registered = registered || parameter.Declaration == key.declaration
			}
		}
		if !registered {
			return false
		}
		visiting[key] = true
		defer delete(visiting, key)
		for _, bound := range binding.parameter.Bounds {
			if raf209SelectedStringBound(bound, declaring, visiting) {
				return true
			}
		}
		return false
	}
	base, rank := javaArrayTypeParts(strings.TrimSpace(actual.Original))
	base, args := parseJavaTypeString(base)
	return rank == 0 && len(args) == 0 && !raf209StaticTypeConflict(base, ctx) && intrinsicStringBoundType(actual, ctx, map[typeParameterIdentityKey]bool{})
}

func raf209DeclaredString(node *sitter.Node, ctx Ctx, source []byte) bool {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil || !raf209Owns(node, ctx, source) {
		return false
	}
	switch node.Type() {
	case "string_literal":
		return true
	case "object_creation_expression":
		// This attests the operand's native declared type. Its constructor
		// overload and argument lowering remain the ordinary String route.
		return objectCreationClassBody(node) == nil && raf209CanonicalType(node.ChildByFieldName("type"), "java.lang.String", ctx, source)
	case "null_literal":
		return true
	case "cast_expression":
		typ := node.ChildByFieldName("type")
		return typ != nil && raf209WrittenString(typ.Content(source), ctx)
	case "identifier":
		if binding, found := fileChannelReadCaptureDeclaration(node, source); found {
			declaration := binding.declaration
			if declaration.ChildByFieldName("dimensions") != nil {
				return false
			}
			var typ *sitter.Node
			if declaration.Type() == "variable_declarator" {
				typ = declaration.Parent().ChildByFieldName("type")
			} else {
				typ = declaration.ChildByFieldName("type")
			}
			return typ != nil && readerSourceNodeWithin(declaration, ctx.currentFile.ParsedRoot()) && raf209WrittenString(typ.Content(source), ctx)
		}
		field := findFieldResolutionInHierarchy(ctx.currentClass, node.Content(source), ctx)
		if field != nil {
			return raf209FieldString(field, ctx)
		}
	case "method_invocation":
		resolution, target, sourceTarget := raf209SelectSourceInvocation(node, ctx, source)
		if resolution != nil && resolution.owner != nil && resolution.def != nil {
			file := declaredRefusalRegisteredScopeFile(resolution.owner, ctx)
			def := resolution.def
			if file == nil || !helpfulRegisteredMethodHeader(resolution.owner, def, ctx) {
				return false
			}
			typ := def.DeclarationNode.ChildByFieldName("type")
			if typ == nil || def.DeclarationNode.ChildByFieldName("dimensions") != nil || typ.Content(file.Source) != def.OriginalType {
				return false
			}
			declaring := classScopeCtx(resolution.owner, ctx.Clone())
			declaring.localScope = def
			// Every selected bare method binder, including a String-bounded
			// one, must pass this invocation's explicit/witness/origin checks.
			if symbol.DirectTypeParamForJavaType(typ.Content(file.Source), def.TypeParameters) != nil {
				return raf209MethodStringResult(node, resolution, target, ctx, source)
			}
			return raf209WrittenString(typ.Content(file.Source), declaring)
		}
		if sourceTarget {
			return false
		}
	case "field_access":
		object, name := node.ChildByFieldName("object"), node.ChildByFieldName("field")
		if object != nil && name != nil {
			if owner, _ := staticFieldQualifierScope(object, source, ctx); owner != nil {
				return raf209FieldString(findFieldResolutionInHierarchy(owner, name.Content(source), ctx), ctx)
			}
		}
	case "ternary_expression":
		left, right := node.ChildByFieldName("consequence"), node.ChildByFieldName("alternative")
		return raf209DeclaredString(left, ctx, source) && raf209DeclaredString(right, ctx, source)
	case "binary_expression":
		operator := node.ChildByFieldName("operator")
		return operator != nil && operator.Content(source) == "+" && (raf209DeclaredStringNonNull(node.ChildByFieldName("left"), ctx, source) || raf209DeclaredStringNonNull(node.ChildByFieldName("right"), ctx, source))
	}
	// Existing external String-returning intrinsics have no source Definition.
	// Keep their final canonical result, but never use this fallback for a local,
	// formal, source field, or selected source method with unproved syntax.
	if node.Type() != "method_invocation" && node.Type() != "field_access" {
		return false
	}
	var origin inferredJavaTypeOrigin
	actual, known := inferExprJavaType(node, ctx, source, &origin)
	return known && !origin.unresolved && origin.parameter == nil && origin.nominalScope == nil && origin.arrayRank == 0 && len(origin.arguments) == 0 && origin.externalName == "java.lang.String" && isBuiltinJavaString(actual, stringReaderReferenceDeclaringContext(origin, ctx))
}

func raf209DeclaredStringNonNull(node *sitter.Node, ctx Ctx, source []byte) bool {
	node = unwrapParenthesizedExpressionNode(node)
	return node != nil && node.Type() != "null_literal" && raf209DeclaredString(node, ctx, source)
}

func raf209FieldString(field *fieldResolution, ctx Ctx) bool {
	if field == nil || field.def == nil {
		return false
	}
	file := declaredRefusalRegisteredScopeFile(field.owner, ctx)
	declaration := field.def.DeclarationNode
	if file == nil || declaration == nil || !readerSourceNodeWithin(declaration, field.owner.Class.DeclarationNode) {
		return false
	}
	if declaration.Type() == "variable_declarator" {
		declaration = declaration.Parent()
	}
	typ := declaration.ChildByFieldName("type")
	named := false
	for _, declarator := range nodeutil.VariableDeclarators(declaration) {
		name := declarator.ChildByFieldName("name")
		if name != nil && name.Content(file.Source) == field.def.OriginalName && declarator.ChildByFieldName("dimensions") == nil {
			named = true
		}
	}
	return named && typ != nil && typ.Content(file.Source) == field.def.OriginalType && raf209WrittenString(typ.Content(file.Source), classScopeCtx(field.owner, ctx.Clone()))
}

func raf209CanonicalType(typ *sitter.Node, owner string, ctx Ctx, source []byte) bool {
	if typ == nil {
		return false
	}
	base, rank := javaArrayTypeParts(strings.TrimSpace(typ.Content(source)))
	base, args := parseJavaTypeString(base)
	if rank != 0 || len(args) != 0 || raf209StaticTypeConflict(base, ctx) || raf209SyntaxParameter(strings.Split(base, ".")[0], ctx) || visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return false
	}
	// String deliberately uses the existing semantic predicate rather than
	// the migrated intrinsic-owner registry used by File and RAF.
	if owner == "java.lang.String" {
		return isBuiltinJavaString(base, ctx)
	}
	actual, known := canonicalIntrinsicOwner(base, ctx)
	return known && actual == owner
}

func raf209NativeFileProducer(node *sitter.Node, ctx Ctx, source []byte) bool {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil || !raf209Owns(node, ctx, source) {
		return false
	}
	switch node.Type() {
	case "cast_expression":
		value := unwrapParenthesizedExpressionNode(node.ChildByFieldName("value"))
		return raf209CanonicalType(node.ChildByFieldName("type"), "java.io.File", ctx, source) && value != nil && value.Type() == "null_literal"
	case "object_creation_expression":
		return objectCreationClassBody(node) == nil && raf209CanonicalType(node.ChildByFieldName("type"), "java.io.File", ctx, source) && invocationArgumentCount(node) == 1 && raf209DeclaredString(invocationArgumentNode(node, 0), ctx, source)
	case "method_invocation":
		object, name := node.ChildByFieldName("object"), node.ChildByFieldName("name")
		if object == nil || raf209StaticTypeConflict(object.Content(source), ctx) || name == nil || name.Content(source) != "createTempFile" || invocationArgumentCount(node) != 2 || node.ChildByFieldName("type_arguments") != nil {
			return false
		}
		owner, known := intrinsicStaticClassName(object, ctx, source)
		canonical, registered := canonicalIntrinsicOwner(owner, ctx)
		if !known || !registered || canonical != "java.io.File" || resolveInvocationTarget(object, ctx, source) != nil {
			return false
		}
		return raf209DeclaredString(invocationArgumentNode(node, 0), ctx, source) && raf209DeclaredString(invocationArgumentNode(node, 1), ctx, source)
	}
	return false
}

func raf209NativeFileOrigin(node *sitter.Node, ctx Ctx, source []byte) bool {
	node = unwrapParenthesizedExpressionNode(node)
	if raf209NativeFileProducer(node, ctx, source) {
		return true
	}
	if node == nil || node.Type() != "identifier" || !raf209Owns(node, ctx, source) || ctx.localScope == nil || !helpfulRegisteredMethodHeader(ctx.currentClass, ctx.localScope, ctx) || !fileChannelReadExternalExpression(node, "java.io.File", ctx, source) {
		return false
	}
	binding, found := fileChannelReadCaptureDeclaration(node, source)
	if !found || binding.declaration.Type() != "variable_declarator" || binding.declaration.ChildByFieldName("dimensions") != nil {
		return false
	}
	declaration := binding.declaration.Parent()
	method := ctx.localScope.DeclarationNode
	if declaration == nil || declaration.Type() != "local_variable_declaration" || !readerSourceNodeWithin(binding.declaration, method) || !raf209CanonicalType(declaration.ChildByFieldName("type"), "java.io.File", ctx, source) || !raf209NativeFileProducer(binding.declaration.ChildByFieldName("value"), ctx, source) {
		return false
	}
	// Reject writes to this exact binding anywhere, including nested syntax.
	// Same-name fields and shadowed locals do not mutate this declaration.
	mutated := false
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil || mutated {
			return
		}
		var target *sitter.Node
		if n.Type() == "assignment_expression" {
			target = n.ChildByFieldName("left")
		} else if n.Type() == "update_expression" && n.NamedChildCount() > 0 {
			target = n.NamedChild(0)
		}
		target = unwrapParenthesizedExpressionNode(target)
		if target != nil && target.Type() == "identifier" {
			if actual, found := fileChannelReadCaptureDeclaration(target, source); found && actual.declaration.Equal(binding.declaration) {
				mutated = true
				return
			}
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(n) {
			walk(child)
		}
	}
	walk(method.ChildByFieldName("body"))
	return !mutated
}

func raf209Constructor(node *sitter.Node, typeArgs []ast.Expr, ctx Ctx, source []byte) raf209ConstructorKind {
	if node == nil || len(typeArgs) != 0 || invocationArgumentCount(node) != 2 || objectCreationClassBody(node) != nil || !raf209Owns(node, ctx, source) || !raf209CanonicalType(node.ChildByFieldName("type"), "java.io.RandomAccessFile", ctx, source) {
		return raf209Refused
	}
	proof := ctx.Clone()
	proof.expectedType, proof.expectedTypeRoot = "", nil
	path, mode := invocationArgumentNode(node, 0), invocationArgumentNode(node, 1)
	plain := unwrapParenthesizedExpressionNode(path)
	if plain == nil || plain.Type() == "null_literal" || !raf209DeclaredString(mode, proof, source) {
		return raf209Refused
	}
	if raf209NativeFileOrigin(path, proof, source) {
		// No complete external static member-type inventory is available. Hold
		// new File admission here; it must never fall through to String's
		// historical conversion fallback. Legacy String wildcard behavior is
		// retained independently below.
		if raf209HasStaticWildcard(proof) {
			return raf209Refused
		}
		return raf209NativeFile
	}
	actual, known := inferExprJavaType(path, proof, source)
	if !known || actual == "null" || actual == ternaryNullJavaType {
		return raf209Refused
	}
	if raf209DeclaredString(path, proof, source) && randomAccessFileStringConstructorApplicable(node, typeArgs, proof, source) {
		return raf209String
	}
	return raf209Refused
}

// Operands have already been parsed/snapshotted by the ordinary caller. Store
// each exactly once in Java order before any constructor-body validation.
func raf209LowerFile(args []ast.Expr, ctx Ctx, node *sitter.Node, source []byte) ast.Expr {
	used := affineLoopUsedNames(node, source, ctx)
	used[ctx.executionContextName] = struct{}{}
	used[ShortName(ctx.className)] = struct{}{}
	if ctx.currentClass != nil {
		for _, parameter := range ctx.currentClass.GoTypeParameterNames() {
			used[parameter] = struct{}{}
		}
	}
	if ctx.localScope != nil {
		for _, parameter := range ctx.localScope.GoTypeParameterNames() {
			used[parameter] = struct{}{}
		}
	}
	name := func(base string) *ast.Ident {
		n := synchronizedUniqueLocalName(base, used)
		used[n] = struct{}{}
		return ast.NewIdent(n)
	}
	file, mode := name("__java2goRAFFile"), name("__java2goRAFMode")
	path := coerceArgumentToExpectedType(args[0], invocationArgumentNode(node, 0), "java.io.File", ctx, source)
	modeValue := coerceArgumentToExpectedType(args[1], invocationArgumentNode(node, 1), "java.lang.String", ctx, source)
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("RandomAccessFile", ctx)}}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{
		&ast.AssignStmt{Lhs: []ast.Expr{file}, Tok: token.DEFINE, Rhs: []ast.Expr{path}},
		&ast.AssignStmt{Lhs: []ast.Expr{mode}, Tok: token.DEFINE, Rhs: []ast.Expr{modeValue}},
		&ast.ReturnStmt{Results: []ast.Expr{stdjavaCall(ctx, "NewRandomAccessFileFileExecution", intrinsicExecutionExpr(ctx), file, mode)}},
	}}}}
}

// A captured immutable TWR binding keeps the constructor's declaring executable.
// Its use occurrence still belongs to the lambda; only the initializer proof is
// replayed here, against the actual registered source declaration.
func raf209ResourceInitializerContext(occurrence, resource *sitter.Node, ctx Ctx, source []byte) (Ctx, bool) {
	if occurrence == nil || occurrence.Type() != "identifier" || resource == nil || resource.Type() != "resource" || ctx.currentClass == nil || ctx.currentFile == nil || ctx.localScope == nil {
		return ctx, false
	}
	file := declaredRefusalRegisteredScopeFile(ctx.currentClass, ctx)
	if file == nil || file != ctx.currentFile || !raf209FileCoherent(file) || !bytes.Equal(file.Source, source) || !readerSourceNodeWithin(occurrence, file.ParsedRoot()) || !readerSourceNodeWithin(resource, file.ParsedRoot()) {
		return ctx, false
	}
	binding, found := fileChannelReadCaptureDeclaration(occurrence, source)
	if !found || !binding.declaration.Equal(resource) {
		return ctx, false
	}
	if raf209Owns(resource, ctx, source) {
		return ctx, raf209Owns(occurrence, ctx, source)
	}
	// Only ordinary lambda lowering erases its copied declaration pointer. A
	// copied method, missing frame, or another lambda body cannot use this bridge.
	if ctx.localScope == nil || ctx.localScope.DeclarationNode != nil || ctx.localBindingBody == nil {
		return ctx, false
	}
	var lambda *sitter.Node
	for at := occurrence; at != nil; at = at.Parent() {
		if at.Type() == "lambda_expression" {
			lambda = at
			break
		}
		switch at.Type() {
		case "method_declaration", "constructor_declaration", "compact_constructor_declaration", "class_body":
			return ctx, false
		}
	}
	if lambda == nil || lambda.ChildByFieldName("body") == nil || !lambda.ChildByFieldName("body").Equal(ctx.localBindingBody) || !readerSourceNodeWithin(occurrence, ctx.localBindingBody) {
		return ctx, false
	}
	for at := resource.Parent(); at != nil; at = at.Parent() {
		switch at.Type() {
		case "lambda_expression":
			// A resource created in a lambda needs that executable's own
			// declaration protocol; an enclosing method is not its owner.
			return ctx, false
		case "method_declaration", "constructor_declaration", "compact_constructor_declaration":
			for _, method := range ctx.currentClass.Methods {
				if method == nil || method.DeclarationNode == nil || !method.DeclarationNode.Equal(at) {
					continue
				}
				declaring := ctx.Clone()
				declaring.localScope = method
				declaring.localBindingBody = at.ChildByFieldName("body")
				if !raf209Owns(resource, declaring, source) {
					return ctx, false
				}
				return declaring, true
			}
			return ctx, false
		case "class_body", "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
			return ctx, false
		}
	}
	return ctx, false
}

// A bare selected method binder can produce String through this invocation's
// actual String arguments. Declaration identity and origin remain separate from
// the inference engine's returned spelling and from the generated Go shape.
func raf209MethodStringResult(node *sitter.Node, selected *methodResolution, target *invocationTargetInfo, ctx Ctx, source []byte) bool {
	if node == nil || node.Type() != "method_invocation" || selected == nil || selected.owner == nil || selected.def == nil || !raf209Owns(node, ctx, source) {
		return false
	}
	def := selected.def
	file := declaredRefusalRegisteredScopeFile(selected.owner, ctx)
	if file == nil || !raf209FileCoherent(file) || !helpfulRegisteredMethodHeader(selected.owner, def, ctx) || len(def.TypeParameters) == 0 || invocationArgumentCount(node) != len(def.Parameters) {
		return false
	}
	typ := def.DeclarationNode.ChildByFieldName("type")
	if typ == nil || def.DeclarationNode.ChildByFieldName("dimensions") != nil || typ.Content(file.Source) != def.OriginalType {
		return false
	}
	name, rank := javaArrayTypeParts(strings.TrimSpace(typ.Content(file.Source)))
	name, arguments := parseJavaTypeString(name)
	if rank != 0 || len(arguments) != 0 {
		return false
	}
	classParameters, proved := helpfulClassHeaderParameters(selected.owner, ctx, map[*symbol.ClassScope]bool{})
	if !proved {
		return false
	}
	visible := symbol.MergeTypeParams(classParameters, def.TypeParameters)
	declaration := symbol.DirectTypeParamForJavaType(name, visible)
	if declaration == nil || def.DirectTypeParameter != declaration || !helpfulSameBindings(def.TypeParameterBindings, symbol.VisibleTypeParamBindings(visible)) {
		return false
	}
	parameterIndex := -1
	for index, parameter := range def.TypeParameters {
		if parameter.Declaration == declaration {
			parameterIndex = index
			break
		}
	}
	if parameterIndex < 0 {
		return false
	}
	if explicit := node.ChildByFieldName("type_arguments"); explicit != nil {
		actuals := nodeutil.SemanticNamedChildrenOf(explicit)
		if len(actuals) != len(def.TypeParameters) || !raf209CanonicalType(actuals[parameterIndex], "java.lang.String", ctx, source) {
			return false
		}
	}
	witnesses := 0
	for index, formal := range def.Parameters {
		if executionParameterIsVariadic(def, index) {
			return false
		}
		if formal.DirectTypeParameter != declaration {
			parameters := nodeutil.SemanticNamedChildrenOf(def.DeclarationNode.ChildByFieldName("parameters"))
			written, _ := nodeutil.JavaParameterNodes(parameters[index])
			if raf209TypeSyntaxMentionsBinder(written, file.Source, formal.TypeParameterBindings, declaration) {
				return false
			}
			continue
		}
		base, dimensions := javaArrayTypeParts(strings.TrimSpace(formal.OriginalType))
		base, typeArguments := parseJavaTypeString(base)
		if dimensions != 0 || len(typeArguments) != 0 || symbol.DirectTypeParamForJavaType(base, visible) != declaration || !raf209DeclaredStringNonNull(invocationArgumentNode(node, index), ctx, source) {
			return false
		}
		witnesses++
	}
	if witnesses == 0 {
		return false
	}
	result, known := inferredGenericInvocationResult(def, node, ctx, source)
	if !known {
		return false
	}
	var origin inferredJavaTypeOrigin
	recordMethodTypeOrigin([]*inferredJavaTypeOrigin{&origin}, selected, target, node, ctx, source, result)
	base, declaring, eligible := fileChannelReadOriginContext(result, origin, ctx)
	if !eligible || origin.externalName != "java.lang.String" {
		return false
	}
	return isBuiltinJavaString(base, declaring)
}

func raf209TypeSyntaxMentionsBinder(node *sitter.Node, source []byte, bindings map[string]*symbol.TypeParamDeclaration, declaration *symbol.TypeParamDeclaration) bool {
	if node == nil {
		return false
	}
	if node.Type() == "type_identifier" && bindings[node.Content(source)] == declaration {
		return true
	}
	for _, child := range nodeutil.SemanticNamedChildrenOf(node) {
		if raf209TypeSyntaxMentionsBinder(child, source, bindings, declaration) {
			return true
		}
	}
	return false
}

// Source invocations keep their selected declaration before any external String
// result fallback. These are the existing source lookup routes; no free return
// inference is used to decide that an invocation is owned by source.
func raf209SelectSourceInvocation(node *sitter.Node, ctx Ctx, source []byte) (*methodResolution, *invocationTargetInfo, bool) {
	if node == nil || node.Type() != "method_invocation" || node.ChildByFieldName("name") == nil {
		return nil, nil, false
	}
	name := node.ChildByFieldName("name").Content(source)
	arguments := node.ChildByFieldName("arguments")
	if object := node.ChildByFieldName("object"); object != nil {
		if owner := resolveClassScopeByIdentifier(ctx, source, object); owner != nil {
			return findBestMethodInHierarchy(owner, name, arguments, false, true, ctx, source), nil, true
		}
		target := resolveInvocationTarget(object, ctx, source)
		if target == nil {
			return nil, nil, false
		}
		selected, actualTarget := findBestMethodForInvocationTarget(target, name, arguments, true, true, ctx, source)
		if actualTarget != nil {
			target = actualTarget
		}
		return selected, target, true
	}
	selected := findBestMethodInHierarchy(ctx.currentClass, name, arguments, ctx.localScope == nil || !ctx.localScope.IsStatic, true, ctx, source)
	if selected == nil {
		selected = findEnclosingStaticMethod(name, arguments, ctx, source)
	}
	if selected != nil && selected.owner != nil && selected.def != nil {
		return selected, nil, true
	}
	imported := resolveStaticImportedMethod(node, ctx, source)
	if imported.source != nil {
		return imported.source, nil, true
	}
	return nil, nil, imported.problem != "" || lexicalMethodNamePresent(name, ctx)
}
