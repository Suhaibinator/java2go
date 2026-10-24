package transpiler

import (
	"bytes"
	"go/ast"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// helpfulInvocationPlan describes the no-local-variable-debug JDK diagnostic
// for an unwritten source formal. Other origins retain their existing lowering.
// It does not model source-stack capture, which is unavailable in this cohort.
type helpfulInvocationPlan struct {
	message string
	formal  *symbol.Definition
	owner   *symbol.ClassScope
}

func helpfulSourceTreeOwns(root, node *sitter.Node) bool {
	if root == nil || node == nil {
		return false
	}
	for n := node; n != nil; n = n.Parent() {
		if n.Equal(root) {
			return true
		}
	}
	return false
}

// Local/anonymous classes have executable-specific binary names and captures.
// This slice accepts named top-level and member declarations only.
func helpfulNamedDeclarationBoundary(declaration *sitter.Node) bool {
	if declaration == nil {
		return false
	}
	switch declaration.Type() {
	case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
	default:
		return false
	}
	for node := declaration.Parent(); node != nil; node = node.Parent() {
		switch node.Type() {
		case "method_declaration", "constructor_declaration", "lambda_expression", "object_creation_expression":
			return false
		}
	}
	return true
}
func helpfulSourceOwnerRegistered(owner *symbol.ClassScope, ctx Ctx) bool {
	file := findFileScopeForClassScope(owner, ctx)
	return file != nil && owner != nil && owner.Class != nil && helpfulNamedDeclarationBoundary(owner.Class.DeclarationNode) && helpfulSourceTreeOwns(file.ParsedRoot(), owner.Class.DeclarationNode)
}

// Header checks certify current registered declaration roles. Symbol parameters
// have no immutable AST anchor, so consistently replaced, identical aliases
// cannot be distinguished here and are not claimed as original identities.
func helpfulSameBindings(left, right map[string]*symbol.TypeParamDeclaration) bool {
	if len(left) != len(right) {
		return false
	}
	for name, declaration := range left {
		if right[name] != declaration {
			return false
		}
	}
	return true
}
func helpfulHeaderGroupMatches(parameters, source []symbol.TypeParam, visible []symbol.TypeParam) bool {
	if len(parameters) != len(source) {
		return false
	}
	bindings := symbol.VisibleTypeParamBindings(visible)
	seen := map[*symbol.TypeParamDeclaration]bool{}
	for i, p := range parameters {
		actual := source[i]
		if p.Declaration == nil || seen[p.Declaration] || p.Name != actual.Name || p.Declaration.SourceName != actual.Name || len(p.Bounds) != len(actual.Bounds) {
			return false
		}
		seen[p.Declaration] = true
		for j, bound := range p.Bounds {
			if bound.Original != actual.Bounds[j].Original || !helpfulSameBindings(bound.TypeParameterBindings, bindings) {
				return false
			}
		}
	}
	return true
}
func helpfulClassHeaderParameters(owner *symbol.ClassScope, ctx Ctx, seen map[*symbol.ClassScope]bool) ([]symbol.TypeParam, bool) {
	if owner == nil || seen[owner] || !helpfulSourceOwnerRegistered(owner, ctx) {
		return nil, false
	}
	seen[owner] = true
	defer delete(seen, owner)
	file := findFileScopeForClassScope(owner, ctx)
	name := owner.Class.DeclarationNode.ChildByFieldName("name")
	if name == nil || name.Content(file.Source) != owner.Class.OriginalName {
		return nil, false
	}
	var outer []symbol.TypeParam
	if owner.Enclosing != nil {
		if owner.Enclosing.Class == nil || !helpfulSourceTreeOwns(owner.Enclosing.Class.DeclarationNode, owner.Class.DeclarationNode) {
			return nil, false
		}
		inherited, ok := helpfulClassHeaderParameters(owner.Enclosing, ctx, seen)
		if !ok {
			return nil, false
		}
		if owner.IsInner {
			outer = inherited
		}
	}
	source := symbol.ExtractTypeParameters(owner.Class.DeclarationNode.ChildByFieldName("type_parameters"), file.Source)
	own := owner.DeclaredTypeParameters
	if !helpfulHeaderGroupMatches(own, source, symbol.MergeTypeParams(outer, own)) {
		return nil, false
	}
	expected := symbol.AppendTypeParamsByDeclaration(outer, own)
	if len(owner.TypeParameters) != len(expected) {
		return nil, false
	}
	for i, p := range owner.TypeParameters {
		actual := expected[i]
		if p.Declaration != actual.Declaration || p.Name != actual.Name || len(p.Bounds) != len(actual.Bounds) {
			return nil, false
		}
		for j, bound := range p.Bounds {
			if bound.Original != actual.Bounds[j].Original || !helpfulSameBindings(bound.TypeParameterBindings, actual.Bounds[j].TypeParameterBindings) {
				return nil, false
			}
		}
	}
	return owner.TypeParameters, true
}
func helpfulRegisteredMethodHeader(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) bool {
	if owner == nil || method == nil || !sourceReflectionDeclaredMethod(owner, method) {
		return false
	}
	member := false
	for _, actual := range owner.Methods {
		if actual == method {
			member = true
			break
		}
	}
	if !member {
		return false
	}
	classParameters, ok := helpfulClassHeaderParameters(owner, ctx, map[*symbol.ClassScope]bool{})
	if !ok {
		return false
	}
	file := findFileScopeForClassScope(owner, ctx)
	declaration := method.DeclarationNode
	name := declaration.ChildByFieldName("name")
	if name == nil || name.Content(file.Source) != method.OriginalName {
		return false
	}
	sourceParameters := symbol.ExtractTypeParameters(declaration.ChildByFieldName("type_parameters"), file.Source)
	visible := symbol.MergeTypeParams(classParameters, method.TypeParameters)
	if !helpfulHeaderGroupMatches(method.TypeParameters, sourceParameters, visible) {
		return false
	}
	parameters := nodeutil.SemanticNamedChildrenOf(declaration.ChildByFieldName("parameters"))
	if len(parameters) != len(method.Parameters) {
		return false
	}
	bindings := symbol.VisibleTypeParamBindings(visible)
	for i, formal := range method.Parameters {
		node := parameters[i]
		if formal == nil || (node.Type() != "formal_parameter" && node.Type() != "spread_parameter") || node.ChildByFieldName("dimensions") != nil {
			return false
		}
		typ, name := nodeutil.JavaParameterNodes(node)
		if typ == nil || name == nil || typ.Content(file.Source) != formal.OriginalType || name.Content(file.Source) != formal.OriginalName || formal.DirectTypeParameter != symbol.DirectTypeParamForJavaType(typ.Content(file.Source), visible) || !helpfulSameBindings(formal.TypeParameterBindings, bindings) {
			return false
		}
	}
	return true
}
func helpfulFormalUnwritten(body *sitter.Node, name string, source []byte) bool {
	if body == nil {
		return false
	}
	var safe func(*sitter.Node) bool
	safe = func(n *sitter.Node) bool {
		switch n.Type() {
		case "lambda_expression", "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "object_creation_expression":
			// Nested executable scopes and synthetic captures need their own profile.
			if n.Type() != "object_creation_expression" || objectCreationClassBody(n) != nil {
				return false
			}
		case "assignment_expression", "update_expression":
			left := n.ChildByFieldName("left")
			if n.Type() == "update_expression" {
				left = nodeutil.SemanticNamedChild(n, 0)
			}
			for left != nil && left.Type() == "parenthesized_expression" {
				left = nodeutil.SemanticNamedChild(left, 0)
			}
			if left == nil {
				return false
			}
			if left.Type() == "identifier" && left.Content(source) == name {
				return false
			}
		case "variable_declarator":
			if id := n.ChildByFieldName("name"); id != nil && id.Content(source) == name {
				return false
			}
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(n) {
			if !safe(child) {
				return false
			}
		}
		return true
	}
	return safe(body)
}
func helpfulSourceInvocationPlan(invocation, receiver *sitter.Node, resolution *methodResolution, ctx Ctx, source []byte) *helpfulInvocationPlan {
	if invocation == nil || receiver == nil || resolution == nil || resolution.def == nil || resolution.owner == nil || ctx.localScope == nil || ctx.currentClass == nil {
		return nil
	}
	object := invocation.ChildByFieldName("object")
	called := invocation.ChildByFieldName("name")
	if invocation.Type() != "method_invocation" || object == nil || !object.Equal(receiver) || called == nil {
		return nil
	}
	def := resolution.def
	declaredName := def.DeclarationNode
	if declaredName != nil {
		declaredName = declaredName.ChildByFieldName("name")
	}
	ownerFile := findFileScopeForClassScope(resolution.owner, ctx)
	if called.Content(source) != def.OriginalName || declaredName == nil || ownerFile == nil || declaredName.Content(ownerFile.Source) != def.OriginalName || !methodInvocationArityApplicable(def, int(nodeutil.SemanticNamedChildCount(invocation.ChildByFieldName("arguments")))) {
		return nil
	}
	registeredMethod := false
	for _, method := range resolution.owner.Methods {
		if method == def {
			registeredMethod = true
			break
		}
	}
	if !registeredMethod || !helpfulRegisteredMethodHeader(resolution.owner, def, ctx) || !helpfulRegisteredMethodHeader(ctx.currentClass, ctx.localScope, ctx) {
		return nil
	}
	if def.IsStatic || def.Constructor || def.RequiresHelper || def.RuntimeDefault || !sourceReflectionDeclaredMethod(resolution.owner, def) || !sourceReflectionDeclaredMethod(ctx.currentClass, ctx.localScope) || !helpfulSourceOwnerRegistered(resolution.owner, ctx) || !helpfulSourceOwnerRegistered(ctx.currentClass, ctx) {
		return nil
	}
	callerFile := findFileScopeForClassScope(ctx.currentClass, ctx)
	if callerFile == nil || !bytes.Equal(callerFile.Source, source) {
		return nil
	}
	declaration := ctx.localScope.DeclarationNode
	reached := false
	for n := invocation; n != nil; n = n.Parent() {
		if n.Equal(declaration) {
			reached = true
			break
		}
		switch n.Type() {
		case "lambda_expression", "method_declaration", "constructor_declaration", "class_declaration", "interface_declaration", "object_creation_expression":
			return nil
		}
	}
	if !reached {
		return nil
	}
	for receiver.Type() == "parenthesized_expression" && nodeutil.SemanticNamedChildCount(receiver) == 1 {
		receiver = nodeutil.SemanticNamedChild(receiver, 0)
	}
	if receiver.Type() != "identifier" {
		return nil
	}
	formal := ctx.localScope.ParameterByName(receiver.Content(source))
	if formal == nil {
		return nil
	}
	parameters := nodeutil.SemanticNamedChildrenOf(declaration.ChildByFieldName("parameters"))
	if len(parameters) != len(ctx.localScope.Parameters) {
		return nil
	}
	ordinal := 0
	slot := 0
	if !ctx.localScope.IsStatic {
		slot = 1
	}
	for i, p := range ctx.localScope.Parameters {
		if p == nil {
			return nil
		}
		if parameters[i].Type() != "formal_parameter" && parameters[i].Type() != "spread_parameter" {
			return nil
		}
		_, name := nodeutil.JavaParameterNodes(parameters[i])
		if name == nil || name.Content(source) != p.OriginalName {
			return nil
		}
		if p == formal {
			if slot >= 64 {
				return nil
			}
			ordinal = i + 1
			break
		}
		slot++
		if p.OriginalType == "long" || p.OriginalType == "double" {
			slot++
		}
	}
	if ordinal == 0 || !helpfulFormalUnwritten(declaration.ChildByFieldName("body"), formal.OriginalName, source) {
		return nil
	}
	var origin inferredJavaTypeOrigin
	if _, known := inferExprJavaType(receiver, ctx, source, &origin); !known || origin.unresolved || origin.parameter != nil || origin.arrayRank != 0 || origin.nominalScope != resolution.owner {
		return nil
	}
	declaring := classScopeCtx(resolution.owner, ctx)
	declaring.localScope = def
	// Bounds belong to the source declaration, never to caller synthetic binders.
	declaring.syntheticTypeParameters = nil
	names := make([]string, len(def.Parameters))
	for i, p := range def.Parameters {
		if p == nil {
			return nil
		}
		typ := definitionDeclaredJavaType(p)
		typ.Original = definitionParameterJavaSignatureType(def, i)
		name, ok := helpfulErasedJavaTypeName(declaredJavaTypeOrigin(typ, declaring), declaring, map[*symbol.TypeParamDeclaration]bool{})
		if !ok {
			return nil
		}
		names[i] = name
	}
	owner := javaClassBinaryName(resolution.owner)
	if owner == "" {
		return nil
	}
	signature := owner + "." + def.OriginalName + "(" + strings.Join(names, ", ") + ")"
	return &helpfulInvocationPlan{message: "Cannot invoke \"" + signature + "\" because \"<parameter" + strconv.Itoa(ordinal) + ">\" is null", formal: formal, owner: resolution.owner}
}
func helpfulErasedJavaTypeName(origin inferredJavaTypeOrigin, ctx Ctx, seen map[*symbol.TypeParamDeclaration]bool) (string, bool) {
	if origin.unresolved || origin.wildcard {
		return "", false
	}
	name := ""
	switch {
	case origin.parameter != nil:
		if seen[origin.parameter] {
			return "", false
		}
		seen[origin.parameter] = true
		defer delete(seen, origin.parameter)
		for _, parameter := range visibleTypeParameterDeclarations(ctx) {
			if parameter.Declaration == origin.parameter {
				name = "java.lang.Object"
				if len(parameter.Bounds) > 0 {
					var ok bool
					name, ok = helpfulErasedJavaTypeName(declaredJavaTypeOrigin(parameter.Bounds[0], ctx), ctx, seen)
					if !ok {
						return "", false
					}
				}
				break
			}
		}
	case origin.nominalScope != nil:
		if !helpfulSourceOwnerRegistered(origin.nominalScope, ctx) {
			return "", false
		}
		name = javaClassBinaryName(origin.nominalScope)
	case origin.externalName != "":
		name = origin.externalName
	default:
		base, _ := javaArrayTypeParts(origin.javaType)
		if _, ok := javaPrimitiveType(base); !ok {
			return "", false
		}
		name = base
	}
	if name == "" {
		return "", false
	}
	// HotSpot's invocation diagnostic abbreviates these two signature types.
	// Other java.lang declarations retain their canonical package names.
	if name == "java.lang.Object" || name == "java.lang.String" {
		name = strings.TrimPrefix(name, "java.lang.")
	}
	return name + strings.Repeat("[]", origin.arrayRank), true
}
func appendHelpfulInvocationReceiverCheck(body []ast.Stmt, invocation, receiverNode *sitter.Node, receiver ast.Expr, resolution *methodResolution, ctx Ctx, source []byte) []ast.Stmt {
	plan := helpfulSourceInvocationPlan(invocation, receiverNode, resolution, ctx, source)
	if plan == nil {
		return body
	}
	return append(body, &ast.ExprStmt{X: stdjavaCall(ctx, "RequireInvocationReceiver", receiver, metadataString(plan.message))})
}

// This admission applies only to the new ordinary-call staging path. Existing
// virtual/companion staging continues to query the metadata plan directly.
func helpfulSourceInvocationStagingPlan(invocation, receiver *sitter.Node, resolution *methodResolution, ctx Ctx, source []byte) *helpfulInvocationPlan {
	plan := helpfulSourceInvocationPlan(invocation, receiver, resolution, ctx, source)
	if plan == nil || helpfulInvocationPrimitiveTypesShadowed(invocation, resolution, ctx, source) {
		return nil
	}
	return plan
}

// Capture/result projections can introduce predeclared primitive identifiers
// into an IIFE. Test their actual emitted binders at this lexical position;
// declarations in other methods, later statements or disjoint blocks do not
// shadow this use. This query promises no import/alias/diagnostic mutations,
// rather than claiming purity of all existing generic-family query caches.
func helpfulInvocationPrimitiveTypesShadowed(invocation *sitter.Node, resolution *methodResolution, ctx Ctx, source []byte) bool {
	names := map[string]struct{}{}
	reserve := func(javaType string) {
		primitive, ok := javaPrimitiveType(javaType)
		if !ok {
			primitive, ok = javaPrimitiveArrayComponent(javaType)
		}
		if !ok {
			return
		}
		name := affinePrimitiveGoTypeName(primitive)
		if primitive == "boolean" {
			name = "bool"
		}
		if name != "" {
			names[name] = struct{}{}
		}
	}
	reserve(resolution.def.OriginalType)
	for index, argument := range nodeutil.SemanticNamedChildrenOf(invocation.ChildByFieldName("arguments")) {
		physical := invocationPhysicalParameterJavaType(resolution, index, ctx)
		// Non-contextual array values use :=; a null capture names the array type.
		if invocationArgumentHasPrimitiveFormal(physical) || invocationArgumentNeedsContextualType(nil, argument) {
			reserve(physical)
		}
	}
	if len(names) == 0 {
		return false
	}
	for _, parameter := range visibleTypeParameterDeclarations(ctx) {
		if _, shadowed := names[parameter.EmittedName()]; shadowed {
			return true
		}
	}
	for _, parameter := range ctx.localScope.Parameters {
		if parameter != nil {
			if _, shadowed := names[sanitizeGoIdent(parameter.Name)]; shadowed {
				return true
			}
		}
	}
	emits := func(identifier *sitter.Node) bool {
		if identifier == nil || identifier.Type() != "identifier" {
			return false
		}
		_, shadowed := names[localBindingName(identifier.Content(source), ctx)]
		return shadowed
	}
	declaration := func(node *sitter.Node) bool {
		if node == nil || node.Type() != "local_variable_declaration" {
			return false
		}
		for _, variable := range nodeutil.VariableDeclarators(node) {
			identifier := variable.ChildByFieldName("name")
			if identifier != nil && variable.EndByte() <= invocation.StartByte() && emits(identifier) {
				return true
			}
		}
		return false
	}
	for child, parent := invocation, invocation.Parent(); parent != nil; child, parent = parent, parent.Parent() {
		if parent.Equal(ctx.localScope.DeclarationNode) {
			break
		}
		switch parent.Type() {
		case "block", "constructor_body":
			for _, sibling := range nodeutil.NamedChildrenOf(parent) {
				if sibling.StartByte() >= child.StartByte() {
					break
				}
				if declaration(sibling) {
					return true
				}
			}
		case "local_variable_declaration":
			if declaration(parent) {
				return true
			}
		case "for_statement":
			if declaration(parent.ChildByFieldName("init")) {
				return true
			}
		case "enhanced_for_statement":
			if receiverNodeWithin(invocation, parent.ChildByFieldName("body")) && emits(parent.ChildByFieldName("name")) {
				return true
			}
		case "catch_clause":
			if receiverNodeWithin(invocation, parent.ChildByFieldName("body")) {
				for _, parameter := range nodeutil.NamedChildrenOf(parent) {
					if parameter.Type() == "catch_formal_parameter" && emits(parameter.ChildByFieldName("name")) {
						return true
					}
				}
			}
		case "try_with_resources_statement":
			resources := parent.ChildByFieldName("resources")
			if receiverNodeWithin(invocation, resources) || receiverNodeWithin(invocation, parent.ChildByFieldName("body")) {
				for _, resource := range nodeutil.NamedChildrenOf(resources) {
					identifier := resource.ChildByFieldName("name")
					if identifier != nil && identifier.EndByte() <= invocation.StartByte() && emits(identifier) {
						return true
					}
				}
			}
		}
	}
	return false
}
