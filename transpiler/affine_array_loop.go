package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

type affineArrayBindingKey struct {
	receiverName string
	view         *symbol.AffineArrayView
}

type affineArrayCallSiteKey struct {
	start uint32
	end   uint32
}

type affineArrayLoopBinding struct {
	key              affineArrayBindingKey
	receiverGoName   string
	receiverJavaType string
	receiverClass    *symbol.ClassScope
	arrayName        string
	strideName       string
	ownerLoopStart   uint32
	used             bool
}

type affineArrayCallCandidate struct {
	node          *sitter.Node
	receiverName  string
	receiverDef   *symbol.Definition
	receiverClass *symbol.ClassScope
	method        *symbol.Definition
	accessor      *symbol.TrivialArrayAccessor
}

// prepareAffineArrayLoop approves exact source call sites and creates cache
// bindings for an ordinary for-loop body. It never descends across a closure or
// class boundary, and it leaves loop headers on the ordinary invocation path.
func prepareAffineArrayLoop(node *sitter.Node, source []byte, ctx Ctx) (Ctx, []*affineArrayLoopBinding) {
	if node == nil || node.Type() != "for_statement" || ctx.localScope == nil || ctx.localScope.Constructor || ctx.localScope.DeclarationNode == nil {
		return ctx, nil
	}
	if affineRuntimeIdentifiersShadowed(ctx, source) {
		return ctx, nil
	}
	if parent := node.Parent(); parent != nil && parent.Type() == "labeled_statement" {
		return ctx, nil
	}
	body := node.ChildByFieldName("body")
	if body == nil || affineLoopContainsDeferredScope(body) || affineLoopContainsLabel(body) {
		return ctx, nil
	}

	candidates := collectAffineArrayCallCandidates(body, source, ctx)
	if len(candidates) == 0 {
		return ctx, nil
	}

	loopCtx := ctx.Clone()
	loopCtx.affineArrayBindings = cloneAffineArrayBindings(ctx.affineArrayBindings)
	loopCtx.affineArrayCallSites = cloneAffineArrayCallSites(ctx.affineArrayCallSites)
	usedNames := affineLoopUsedNames(node, source, ctx)
	for _, candidate := range candidates {
		for name := range affineIIFETypeIdentifiers(candidate.receiverClass, candidate.method, ctx) {
			usedNames[name] = struct{}{}
		}
	}
	stable := make(map[string]bool)
	stabilityKnown := make(map[string]bool)
	created := make([]*affineArrayLoopBinding, 0)

	for _, candidate := range candidates {
		if !stabilityKnown[candidate.receiverName] {
			stable[candidate.receiverName] = affineLoopReceiverStable(node, candidate.receiverName, source)
			stabilityKnown[candidate.receiverName] = true
		}
		if !stable[candidate.receiverName] {
			continue
		}

		key := affineArrayBindingKey{receiverName: candidate.receiverName, view: candidate.accessor.View}
		binding := loopCtx.affineArrayBindings[key]
		if binding == nil {
			prefix := "__java2goAffine" + strconv.FormatUint(uint64(node.StartByte()), 10)
			binding = &affineArrayLoopBinding{
				key:              key,
				receiverGoName:   sanitizeGoIdent(candidate.receiverDef.Name),
				receiverJavaType: candidate.receiverDef.OriginalType,
				receiverClass:    candidate.receiverClass,
				arrayName:        affineUniqueLocalName(prefix+"Values", usedNames),
				strideName:       affineUniqueLocalName(prefix+"Stride", usedNames),
				ownerLoopStart:   node.StartByte(),
			}
			loopCtx.affineArrayBindings[key] = binding
			created = append(created, binding)
		}

		loopCtx.affineArrayCallSites[affineArrayCallSiteKey{start: candidate.node.StartByte(), end: candidate.node.EndByte()}] = binding
	}

	if len(created) == 0 && len(loopCtx.affineArrayCallSites) == len(ctx.affineArrayCallSites) {
		return ctx, nil
	}
	return loopCtx, created
}

// Go labels are function-scoped. Versioning duplicates the loop body, so even
// an unrelated nested Java label would otherwise become two declarations of
// the same Go label and fail compilation.
func affineLoopContainsLabel(root *sitter.Node) bool {
	if root == nil {
		return false
	}
	if root.Type() == "labeled_statement" {
		return true
	}
	for _, child := range nodeutil.NamedChildrenOf(root) {
		if affineLoopContainsLabel(child) {
			return true
		}
	}
	return false
}

// The inline guard deliberately uses Go's predeclared panic/nil identifiers and
// the fixed stdjava runtime alias. Java permits all three as identifiers. Fall
// back when any source declaration or import could shadow them rather than emit
// invalid or mis-bound Go.
func affineRuntimeIdentifiersShadowed(ctx Ctx, source []byte) bool {
	reserved := map[string]struct{}{
		"nil":     {},
		"panic":   {},
		"stdjava": {},
	}
	for name := range reserved {
		if fileScopeDeclaresGoIdent(ctx.currentFile, name) {
			return true
		}
	}
	if ctx.localScope != nil && affineTreeDeclaresAnyName(ctx.localScope.DeclarationNode, reserved, source) {
		return true
	}
	if affineTypeParametersDeclareAnyName(ctx, reserved) {
		return true
	}
	for javaPkg, alias := range ctx.importAliases {
		if javaPkg == stdjavaImportPath {
			continue
		}
		if _, collision := reserved[alias]; collision {
			return true
		}
	}
	if ctx.currentFile != nil {
		for _, javaPkg := range ctx.currentFile.Imports {
			if _, collision := reserved[packageAliasFromJavaPackage(javaPkg)]; collision {
				return true
			}
		}
	}
	return false
}

func cloneAffineArrayBindings(source map[affineArrayBindingKey]*affineArrayLoopBinding) map[affineArrayBindingKey]*affineArrayLoopBinding {
	result := make(map[affineArrayBindingKey]*affineArrayLoopBinding, len(source))
	for key, binding := range source {
		result[key] = binding
	}
	return result
}

func cloneAffineArrayCallSites(source map[affineArrayCallSiteKey]*affineArrayLoopBinding) map[affineArrayCallSiteKey]*affineArrayLoopBinding {
	result := make(map[affineArrayCallSiteKey]*affineArrayLoopBinding, len(source))
	for key, binding := range source {
		result[key] = binding
	}
	return result
}

func cloneAffineArrayNonNullBindings(source map[*affineArrayLoopBinding]struct{}) map[*affineArrayLoopBinding]struct{} {
	result := make(map[*affineArrayLoopBinding]struct{}, len(source))
	for binding := range source {
		result[binding] = struct{}{}
	}
	return result
}

func affineLoopContainsDeferredScope(root *sitter.Node) bool {
	var found bool
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil || found {
			return
		}
		switch node.Type() {
		case "lambda_expression", "method_reference", "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
			found = true
			return
		case "object_creation_expression":
			if objectCreationClassBody(node) != nil {
				found = true
				return
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(root)
	return found
}

func collectAffineArrayCallCandidates(root *sitter.Node, source []byte, ctx Ctx) []affineArrayCallCandidate {
	var candidates []affineArrayCallCandidate
	var walk func(*sitter.Node, bool)
	walk = func(node *sitter.Node, rootNode bool) {
		if node == nil {
			return
		}
		if !rootNode {
			switch node.Type() {
			case "for_statement", "enhanced_for_statement", "while_statement", "do_statement":
				if parent := node.Parent(); parent != nil && parent.Type() == "labeled_statement" {
					return
				}
				if body := affineNestedLoopBody(node); body != nil {
					walk(body, true)
				}
				return
			}
		}
		if node.Type() == "method_invocation" {
			if candidate, ok := analyzeAffineArrayCallCandidate(node, source, ctx); ok {
				candidates = append(candidates, candidate)
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child, false)
		}
	}
	walk(root, true)
	return candidates
}

func affineNestedLoopBody(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if body := node.ChildByFieldName("body"); body != nil {
		return body
	}
	switch node.Type() {
	case "while_statement":
		if node.NamedChildCount() > 1 {
			return node.NamedChild(1)
		}
	case "do_statement":
		if node.NamedChildCount() > 0 {
			return node.NamedChild(0)
		}
	case "for_statement", "enhanced_for_statement":
		if count := int(node.NamedChildCount()); count > 0 {
			return node.NamedChild(count - 1)
		}
	}
	return nil
}

func analyzeAffineArrayCallCandidate(node *sitter.Node, source []byte, ctx Ctx) (affineArrayCallCandidate, bool) {
	objectNode := node.ChildByFieldName("object")
	nameNode := node.ChildByFieldName("name")
	if objectNode == nil || objectNode.Type() != "identifier" || nameNode == nil {
		return affineArrayCallCandidate{}, false
	}
	receiverName := objectNode.Content(source)
	receiverDef := affineLocalDefinition(receiverName, ctx)
	if receiverDef == nil || strings.TrimSpace(receiverDef.OriginalType) == "" {
		return affineArrayCallCandidate{}, false
	}
	target := resolveInvocationTarget(objectNode, ctx, source)
	if target == nil || target.classScope == nil || target.classScope.Class == nil || !target.classScope.Class.IsFinal || len(target.classScope.TypeParameters) != 0 {
		return affineArrayCallCandidate{}, false
	}
	receiverBase, receiverTypeArgs := parseJavaTypeString(receiverDef.OriginalType)
	if len(receiverTypeArgs) != 0 || resolveClassScopeByQualifiedName(ctx, receiverBase) != target.classScope {
		// A type parameter may resolve calls through its upper bound, but it is not
		// the exact concrete receiver type required by this fast path.
		return affineArrayCallCandidate{}, false
	}
	argsNode := node.ChildByFieldName("arguments")
	resolution := findBestMethodInHierarchy(target.classScope, nameNode.Content(source), argsNode, true, false, ctx, source)
	if resolution == nil || resolution.def == nil || resolution.owner != target.classScope || resolution.def.TrivialArrayAccessor == nil {
		return affineArrayCallCandidate{}, false
	}
	if affineIIFETypeIdentifiersShadowed(target.classScope, resolution.def, ctx, source) {
		return affineArrayCallCandidate{}, false
	}
	accessor := resolution.def.TrivialArrayAccessor
	if accessor.View == nil || accessor.View.HelperName == "" || !affineArrayArgumentsSafe(argsNode, source, ctx) {
		return affineArrayCallCandidate{}, false
	}
	return affineArrayCallCandidate{
		node:          node,
		receiverName:  receiverName,
		receiverDef:   receiverDef,
		receiverClass: target.classScope,
		method:        resolution.def,
		accessor:      accessor,
	}, true
}

// A value declared in an outer Go function body shadows type identifiers used
// by a nested function literal. Java has separate type and value namespaces, so
// names such as `Grid Grid` and `int int32` are legal. Keep those call sites on
// ordinary dispatch instead of emitting a nested signature whose types no
// longer resolve.
func affineIIFETypeIdentifiersShadowed(receiverClass *symbol.ClassScope, method *symbol.Definition, ctx Ctx, source []byte) bool {
	typeNames := affineIIFETypeIdentifiers(receiverClass, method, ctx)
	if ctx.localScope == nil {
		return false
	}
	if affineTypeParametersDeclareAnyName(ctx, typeNames) {
		return true
	}
	if affineTreeDeclaresAnyName(ctx.localScope.DeclarationNode, typeNames, source) {
		return true
	}
	for name := range typeNames {
		for _, parameter := range ctx.localScope.Parameters {
			if definitionDeclaresGoIdent(parameter, name) {
				return true
			}
		}
		for _, local := range ctx.localScope.Children {
			if definitionDeclaresGoIdent(local, name) {
				return true
			}
		}
	}
	return false
}

func affineIIFETypeIdentifiers(receiverClass *symbol.ClassScope, method *symbol.Definition, ctx Ctx) map[string]struct{} {
	typeNames := map[string]struct{}{
		"int":   {},
		"int32": {},
	}
	if receiverClass != nil && receiverClass.Class != nil {
		declaringFile := findFileScopeForClassScope(receiverClass)
		if declaringFile == nil || ctx.currentFile == nil || declaringFile.Package == ctx.currentFile.Package {
			typeNames[receiverClass.Class.Name] = struct{}{}
		} else if alias := ctx.importAliases[declaringFile.Package]; alias != "" {
			typeNames[alias] = struct{}{}
		} else if alias := packageAliasFromJavaPackage(declaringFile.Package); alias != "" {
			typeNames[alias] = struct{}{}
		}
	}
	if method != nil {
		if goType := affinePrimitiveGoTypeName(method.OriginalType); goType != "" {
			typeNames[goType] = struct{}{}
		}
		for _, parameter := range method.Parameters {
			if parameter == nil {
				continue
			}
			if goType := affinePrimitiveGoTypeName(parameter.OriginalType); goType != "" {
				typeNames[goType] = struct{}{}
			}
		}
	}
	return typeNames
}

func affineTypeParametersDeclareAnyName(ctx Ctx, names map[string]struct{}) bool {
	for _, typeParameter := range inScopeTypeParameters(ctx) {
		if _, collision := names[typeParameter]; collision {
			return true
		}
		if _, collision := names[sanitizeGoIdent(typeParameter)]; collision {
			return true
		}
	}
	return false
}

func affineTreeDeclaresAnyName(root *sitter.Node, names map[string]struct{}, source []byte) bool {
	if root == nil || len(names) == 0 {
		return false
	}
	found := false
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil || found {
			return
		}
		for name := range names {
			if affineNodeDeclaresName(node, name, source) {
				found = true
				return
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(root)
	return found
}

func affinePrimitiveGoTypeName(javaType string) string {
	switch strings.TrimSpace(javaType) {
	case "byte":
		return "int8"
	case "short":
		return "int16"
	case "int":
		return "int32"
	case "long":
		return "int64"
	case "char":
		return "rune"
	case "float":
		return "float32"
	case "double":
		return "float64"
	default:
		return ""
	}
}

func affineLocalDefinition(name string, ctx Ctx) *symbol.Definition {
	if ctx.localScope == nil {
		return nil
	}
	var match *symbol.Definition
	if parameter := ctx.localScope.ParameterByName(name); parameter != nil {
		match = parameter
	}
	for _, local := range ctx.localScope.Children {
		if local != nil && local.OriginalName == name {
			if match != nil {
				// Symbol definitions are method-wide today. Reject ambiguous names from
				// disjoint lexical blocks instead of caching the wrong Go identifier.
				return nil
			}
			match = local
		}
	}
	return match
}

func affineArrayArgumentsSafe(argsNode *sitter.Node, source []byte, ctx Ctx) bool {
	if argsNode == nil {
		return true
	}
	for _, argument := range nodeutil.NamedChildrenOf(argsNode) {
		if !affineArrayArgumentSafe(argument, source, ctx) {
			return false
		}
	}
	return true
}

func affineArrayArgumentSafe(node *sitter.Node, source []byte, ctx Ctx) bool {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return false
	}
	switch node.Type() {
	case "decimal_integer_literal", "hex_integer_literal", "octal_integer_literal", "binary_integer_literal",
		"decimal_floating_point_literal", "hex_floating_point_literal", "character_literal":
		return true
	case "identifier":
		return affineLocalNumericIdentifier(node.Content(source), ctx)
	case "unary_expression":
		if node.ChildCount() < 2 {
			return false
		}
		switch node.Child(0).Content(source) {
		case "+", "-", "~":
			return affineArrayArgumentSafe(node.Child(1), source, ctx)
		default:
			return false
		}
	case "binary_expression":
		if node.ChildCount() < 3 {
			return false
		}
		switch node.Child(1).Content(source) {
		case "+", "-", "*", "&", "|", "^":
			return affineArrayArgumentSafe(node.Child(0), source, ctx) && affineArrayArgumentSafe(node.Child(2), source, ctx)
		default:
			return false
		}
	case "cast_expression":
		if node.NamedChildCount() != 2 || !affineNumericPrimitive(node.NamedChild(0).Content(source)) {
			return false
		}
		return affineArrayArgumentSafe(node.NamedChild(1), source, ctx)
	default:
		return false
	}
}

func affineLocalNumericIdentifier(name string, ctx Ctx) bool {
	if ctx.localScope == nil {
		return false
	}
	found := false
	if parameter := ctx.localScope.ParameterByName(name); parameter != nil {
		if !affineNumericPrimitive(parameter.OriginalType) {
			return false
		}
		found = true
	}
	for _, local := range ctx.localScope.Children {
		if local == nil || local.OriginalName != name {
			continue
		}
		if !affineNumericPrimitive(local.OriginalType) {
			return false
		}
		found = true
	}
	return found
}

func affineNumericPrimitive(javaType string) bool {
	switch strings.TrimSpace(javaType) {
	case "byte", "short", "int", "long", "char", "float", "double":
		return true
	default:
		return false
	}
}

func affineLoopReceiverStable(loop *sitter.Node, name string, source []byte) bool {
	stable := true
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil || !stable {
			return
		}
		if affineNodeDeclaresName(node, name, source) {
			stable = false
			return
		}
		switch node.Type() {
		case "assignment_expression":
			if node.ChildCount() > 0 && affineSimpleIdentifierName(node.Child(0), source) == name {
				stable = false
				return
			}
		case "update_expression":
			for _, child := range nodeutil.NamedChildrenOf(node) {
				if affineSimpleIdentifierName(child, source) == name {
					stable = false
					return
				}
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(loop)
	return stable
}

func affineNodeDeclaresName(node *sitter.Node, name string, source []byte) bool {
	if node == nil {
		return false
	}
	switch node.Type() {
	case "variable_declarator", "formal_parameter", "spread_parameter", "catch_formal_parameter", "type_pattern", "type_parameter":
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil && node.NamedChildCount() > 0 {
			nameNode = node.NamedChild(int(node.NamedChildCount()) - 1)
		}
		return nameNode != nil && nameNode.Type() == "identifier" && nameNode.Content(source) == name
	case "enhanced_for_statement":
		nameNode := node.ChildByFieldName("name")
		return nameNode != nil && nameNode.Content(source) == name
	default:
		return false
	}
}

func affineSimpleIdentifierName(node *sitter.Node, source []byte) string {
	node = unwrapParenthesizedExpressionNode(node)
	if node != nil && node.Type() == "identifier" {
		return node.Content(source)
	}
	return ""
}

func affineLoopUsedNames(loop *sitter.Node, source []byte, ctx Ctx) map[string]struct{} {
	used := make(map[string]struct{})
	if ctx.localScope != nil {
		for _, definition := range append(append([]*symbol.Definition{}, ctx.localScope.Parameters...), ctx.localScope.Children...) {
			if definition != nil {
				used[definition.Name] = struct{}{}
			}
		}
	}
	for _, alias := range ctx.importAliases {
		used[alias] = struct{}{}
	}
	for _, binding := range ctx.affineArrayBindings {
		if binding != nil {
			used[binding.arrayName] = struct{}{}
			used[binding.strideName] = struct{}{}
		}
	}
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "identifier" || node.Type() == "type_identifier" {
			used[sanitizeGoIdent(node.Content(source))] = struct{}{}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(loop)
	return used
}

func affineUniqueLocalName(base string, used map[string]struct{}) string {
	for suffix := 0; ; suffix++ {
		candidate := base
		if suffix > 0 {
			candidate += strconv.Itoa(suffix - 1)
		}
		if _, exists := used[candidate]; exists {
			continue
		}
		used[candidate] = struct{}{}
		return candidate
	}
}

func affineArrayLoopCacheStatements(bindings []*affineArrayLoopBinding) []ast.Stmt {
	statements := make([]ast.Stmt, 0, len(bindings))
	for _, binding := range bindings {
		if binding == nil || !binding.used {
			continue
		}
		statements = append(statements, &ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.Ident{Name: binding.arrayName},
				&ast.Ident{Name: binding.strideName},
			},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{&ast.CallExpr{Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: binding.receiverGoName},
				Sel: &ast.Ident{Name: binding.key.view.HelperName},
			}}},
		})
	}
	return statements
}

// affineArrayLoopValidityCondition selects the guard-free copy of a versioned
// loop only when every receiver and immutable backing-array view used by that
// copy is non-null. The helpers themselves are nil-safe, so evaluating this
// condition before a possibly zero-trip loop cannot introduce an exception.
func affineArrayLoopValidityCondition(bindings []*affineArrayLoopBinding) ast.Expr {
	var condition ast.Expr
	appendNonNil := func(value ast.Expr) {
		nonNil := &ast.BinaryExpr{X: value, Op: token.NEQ, Y: &ast.Ident{Name: "nil"}}
		if condition == nil {
			condition = nonNil
			return
		}
		condition = &ast.BinaryExpr{X: condition, Op: token.LAND, Y: nonNil}
	}
	for _, binding := range bindings {
		if binding == nil || !binding.used {
			continue
		}
		appendNonNil(&ast.Ident{Name: binding.receiverGoName})
		appendNonNil(&ast.Ident{Name: binding.arrayName})
	}
	return condition
}

// rewriteAffineArrayAccessorInvocation lowers one pre-approved resolved call to
// an inline typed closure. Supplying receiver and arguments as closure arguments
// preserves Java's receiver-then-arguments evaluation order and invocation
// conversions; nil checks execute only after all have been staged.
func rewriteAffineArrayAccessorInvocation(
	node *sitter.Node,
	objectNode *sitter.Node,
	objectExpr ast.Expr,
	target *invocationTargetInfo,
	resolution *methodResolution,
	args []ast.Expr,
	ctx Ctx,
	source []byte,
) ast.Expr {
	if node == nil || objectNode == nil || objectNode.Type() != "identifier" || target == nil || resolution == nil || resolution.def == nil {
		return nil
	}
	callSiteKey := affineArrayCallSiteKey{start: node.StartByte(), end: node.EndByte()}
	binding := ctx.affineArrayCallSites[callSiteKey]
	accessor := resolution.def.TrivialArrayAccessor
	if binding == nil || accessor == nil || accessor.View != binding.key.view || target.classScope != binding.receiverClass || resolution.owner != binding.receiverClass {
		return nil
	}
	if objectNode.Content(source) != binding.key.receiverName || len(args) != len(resolution.def.Parameters) || !affineArrayArgumentsSafe(node.ChildByFieldName("arguments"), source, ctx) {
		return nil
	}

	receiverParam := "__java2goAffineReceiver"
	parameters := []*ast.Field{{
		Names: []*ast.Ident{{Name: receiverParam}},
		Type:  javaTypeStringToGoTypeExpr(binding.receiverJavaType, inScopeTypeParameters(ctx), ctx),
	}}
	callArgs := []ast.Expr{objectExpr}
	argumentNames := make([]string, len(args))
	for index, argument := range args {
		argumentName := "__java2goAffineArg" + strconv.Itoa(index)
		argumentNames[index] = argumentName
		parameters = append(parameters, &ast.Field{
			Names: []*ast.Ident{{Name: argumentName}},
			Type:  javaTypeStringToGoTypeExpr(resolution.def.Parameters[index].OriginalType, inScopeTypeParameters(ctx), ctx),
		})
		callArgs = append(callArgs, argument)
	}

	var body []ast.Stmt
	if _, provenNonNull := ctx.affineArrayNonNullBindings[binding]; !provenNonNull {
		body = append(body,
			affineNilPanicGuard(&ast.Ident{Name: receiverParam}, "affine accessor called on null", ctx),
			affineNilPanicGuard(&ast.Ident{Name: binding.arrayName}, "array access on null", ctx),
		)
	}
	indexExpr := func() ast.Expr {
		row := &ast.Ident{Name: argumentNames[accessor.RowParameter]}
		column := &ast.Ident{Name: argumentNames[accessor.ColumnParameter]}
		javaIndex := &ast.BinaryExpr{
			X: &ast.BinaryExpr{
				X:  row,
				Op: token.MUL,
				Y:  &ast.Ident{Name: binding.strideName},
			},
			Op: token.ADD,
			Y:  column,
		}
		return &ast.CallExpr{Fun: &ast.Ident{Name: "int"}, Args: []ast.Expr{javaIndex}}
	}
	arrayIndex := func() *ast.IndexExpr {
		if rowCallSite := ctx.affineArrayRowCallSites[callSiteKey]; rowCallSite != nil && rowCallSite.binding == binding && rowCallSite.rowSlice != nil {
			return &ast.IndexExpr{
				X:     &ast.Ident{Name: rowCallSite.rowSlice.sliceName},
				Index: &ast.Ident{Name: rowCallSite.offsetName},
			}
		}
		return &ast.IndexExpr{X: &ast.Ident{Name: binding.arrayName}, Index: indexExpr()}
	}

	functionType := &ast.FuncType{Params: &ast.FieldList{List: parameters}}
	switch accessor.Kind {
	case symbol.TrivialArrayAccessorGet:
		functionType.Results = &ast.FieldList{List: []*ast.Field{{
			Type: javaTypeStringToGoTypeExpr(resolution.def.OriginalType, inScopeTypeParameters(ctx), ctx),
		}}}
		body = append(body, &ast.ReturnStmt{Results: []ast.Expr{arrayIndex()}})
	case symbol.TrivialArrayAccessorSet:
		body = append(body, &ast.AssignStmt{
			Lhs: []ast.Expr{arrayIndex()},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.Ident{Name: argumentNames[accessor.ValueParameter]}},
		})
	case symbol.TrivialArrayAccessorAdd:
		if accessor.ValueFirst {
			body = append(body, &ast.AssignStmt{
				Lhs: []ast.Expr{arrayIndex()},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{&ast.BinaryExpr{
					X:  &ast.Ident{Name: argumentNames[accessor.ValueParameter]},
					Op: token.ADD,
					Y:  arrayIndex(),
				}},
			})
		} else {
			body = append(body, &ast.AssignStmt{
				Lhs: []ast.Expr{arrayIndex()},
				Tok: token.ADD_ASSIGN,
				Rhs: []ast.Expr{&ast.Ident{Name: argumentNames[accessor.ValueParameter]}},
			})
		}
	default:
		return nil
	}

	binding.used = true
	return &ast.CallExpr{
		Fun:  &ast.FuncLit{Type: functionType, Body: &ast.BlockStmt{List: body}},
		Args: callArgs,
	}
}

func affineNilPanicGuard(value ast.Expr, message string, ctx Ctx) ast.Stmt {
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{X: value, Op: token.EQL, Y: &ast.Ident{Name: "nil"}},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{
			Fun: &ast.Ident{Name: "panic"},
			Args: []ast.Expr{stdjavaCall(ctx, "NewNullPointerException", &ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(message),
			})},
		}}}},
	}
}
