package transpiler

import (
	"bytes"
	"go/ast"
	"go/token"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Enum case identifiers bind in the actual selector declaration's enum, not
// the lexical value namespace used by ordinary identifier expressions.
func parseSwitchLabelExpression(label *sitter.Node, source []byte, ctx Ctx, evidence *enumSwitchLoweringEvidence) ast.Expr {
	fallback := func() ast.Expr { return ParseExpr(label, source, ctx) }
	if label == nil || label.Type() != "identifier" || ctx.currentFile == nil || !bytes.Equal(source, ctx.currentFile.Source) {
		return fallback()
	}
	anchor := ctx.currentFile.ParsedRoot()
	if anchor == nil || anchor.HasError() || anchor.HasChanges() || !enumSwitchNodeWithinSource(label, source) {
		return fallback()
	}
	root := label
	for root.Parent() != nil {
		root = root.Parent()
	}
	if !root.Equal(anchor) {
		return fallback()
	}
	declarationLabel := label.Parent()
	if declarationLabel == nil || declarationLabel.Type() != "switch_label" {
		return fallback()
	}
	arm := declarationLabel.Parent()
	if arm == nil || (arm.Type() != "switch_rule" && arm.Type() != "switch_block_statement_group") {
		return fallback()
	}
	block := arm.Parent()
	if block == nil || block.Type() != "switch_block" {
		return fallback()
	}
	selection := block.Parent()
	if selection == nil || (selection.Type() != "switch_statement" && selection.Type() != "switch_expression") {
		return fallback()
	}
	var selector *sitter.Node
	for _, child := range nodeutil.SemanticNamedChildrenOf(selection) {
		if child.Equal(block) {
			continue
		}
		if selector != nil {
			return fallback()
		}
		selector = child
	}
	if selector == nil || !enumSwitchNodeWithinSource(selector, source) {
		return fallback()
	}
	owner := sourceEnumSwitchOwner(selector, source, ctx)
	if owner == nil {
		return fallback()
	}
	file := findFileScopeForClassScope(owner, ctx)
	name := label.Content(source)
	var access *staticFieldAccess
	for _, constant := range owner.EnumConstants {
		if constant.Name != name {
			continue
		}
		field := constant.Field
		if access != nil || field == nil || !field.IsStatic || !field.IsFinal || field.Name == "" || field.OriginalName != name || owner.FindFieldByName(name) != field || field.DeclarationNode == nil || field.DeclarationNode.Type() != "enum_constant" || !enumSwitchNodeWithinSource(field.DeclarationNode, file.Source) {
			return fallback()
		}
		declaration := field.DeclarationNode
		body := declaration.Parent()
		if body == nil || body.Type() != "enum_body" || body.Parent() == nil || !body.Parent().Equal(owner.Class.DeclarationNode) {
			return fallback()
		}
		declaredName := declaration.ChildByFieldName("name")
		if declaredName == nil || !enumSwitchNodeWithinSource(declaredName, file.Source) || declaredName.Content(file.Source) != name {
			return fallback()
		}
		access = &staticFieldAccess{resolution: &fieldResolution{def: field, owner: owner}}
	}
	if access == nil {
		return fallback()
	}
	return certifyEnumSwitchStorage(staticFieldStorageExpr(access, ctx), label, access.resolution.def, owner, ctx, evidence)
}

func enumSwitchNodeWithinSource(node *sitter.Node, source []byte) bool {
	return node != nil && !node.HasError() && !node.HasChanges() && node.StartByte() <= node.EndByte() && uint64(node.EndByte()) <= uint64(len(source))
}

// A selector with a proven source enum type rejects null unless this switch
// declares its own null label. The generic runtime check preserves the tag's
// exact physical type and evaluates the already-lowered selector only once.
func canonicalSourceEnumSwitchSelector(tag ast.Expr, selector, block *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	if sourceEnumSwitchOwner(selector, source, ctx) == nil || block == nil || block.Type() != "switch_block" || !enumSwitchNodeWithinSource(block, source) {
		return tag
	}
	selection := block.Parent()
	if selection == nil || selector.Parent() == nil || !selector.Parent().Equal(selection) || (selection.Type() != "switch_statement" && selection.Type() != "switch_expression") {
		return tag
	}
	for _, arm := range nodeutil.SemanticNamedChildrenOf(block) {
		if (arm.Type() != "switch_rule" && arm.Type() != "switch_block_statement_group") || !enumSwitchNodeWithinSource(arm, source) {
			return tag
		}
		for _, child := range nodeutil.NamedChildrenOf(arm) {
			if child.Type() != "switch_label" {
				continue
			}
			for _, label := range nodeutil.NamedChildrenOf(child) {
				if label.Type() == "null_literal" {
					return tag
				}
			}
		}
	}
	return stdjavaCall(ctx, "ReferenceRequireNonNull", tag)
}

// Only actual source declarations can establish enum selector ownership.
// There is no name inventory or resolver-result cache: every query reads the
// current context and graph, with an uncertain proof leaving the old path.
func sourceEnumSwitchOwner(selector *sitter.Node, source []byte, ctx Ctx) *symbol.ClassScope {
	if selector == nil || ctx.currentFile == nil || !bytes.Equal(source, ctx.currentFile.Source) || !enumSwitchNodeWithinSource(selector, source) {
		return nil
	}
	anchor := ctx.currentFile.ParsedRoot()
	if anchor == nil || anchor.HasError() || anchor.HasChanges() {
		return nil
	}
	root := selector
	for root.Parent() != nil {
		root = root.Parent()
	}
	if !root.Equal(anchor) {
		return nil
	}
	var origin inferredJavaTypeOrigin
	javaType, known := inferExprJavaType(selector, ctx, source, &origin)
	if !known || origin.unresolved || origin.parameter != nil || origin.nominalScope == nil || origin.externalName != "" || origin.arrayRank != 0 || origin.wildcard || len(origin.arguments) != 0 {
		return nil
	}
	base, rank := javaArrayTypeParts(javaType)
	_, arguments := parseJavaTypeString(base)
	if rank != 0 || len(arguments) != 0 {
		return nil
	}
	owner := resolveClassScopeByQualifiedName(ctx, base)
	if owner == nil || owner != origin.nominalScope || !owner.IsEnum || owner.Class == nil || owner.Class.DeclarationNode == nil || owner.Class.DeclarationNode.Type() != "enum_declaration" {
		return nil
	}
	file := findFileScopeForClassScope(owner, ctx)
	if file == nil || file.ParsedRoot() == nil || file.ParsedRoot().HasError() || file.ParsedRoot().HasChanges() || !enumSwitchNodeWithinSource(owner.Class.DeclarationNode, file.Source) {
		return nil
	}
	ownerRoot := owner.Class.DeclarationNode
	for ownerRoot.Parent() != nil {
		ownerRoot = ownerRoot.Parent()
	}
	if !ownerRoot.Equal(file.ParsedRoot()) {
		return nil
	}
	return owner
}

// The current Java grammar represents the contextual default marker as an
// identifier. Only the exact direct Java 21 null/default pair has this role;
// other expressions and unproven source trees retain the ordinary split path.
func combinedNullDefaultSwitchLabel(label *sitter.Node, source []byte, ctx Ctx) bool {
	if label == nil || label.Type() != "switch_label" || ctx.currentFile == nil || !bytes.Equal(source, ctx.currentFile.Source) || !enumSwitchNodeWithinSource(label, source) {
		return false
	}
	arm := label.Parent()
	if arm == nil || (arm.Type() != "switch_rule" && arm.Type() != "switch_block_statement_group") {
		return false
	}
	block := arm.Parent()
	if block == nil || block.Type() != "switch_block" || block.Parent() == nil || (block.Parent().Type() != "switch_statement" && block.Parent().Type() != "switch_expression") {
		return false
	}
	anchor := ctx.currentFile.ParsedRoot()
	if anchor == nil || anchor.HasError() || anchor.HasChanges() {
		return false
	}
	root := label
	for root.Parent() != nil {
		root = root.Parent()
	}
	if !root.Equal(anchor) {
		return false
	}
	children := nodeutil.SemanticNamedChildrenOf(label)
	if len(children) != 2 || !enumSwitchNodeWithinSource(children[0], source) || !enumSwitchNodeWithinSource(children[1], source) {
		return false
	}
	return children[0].Type() == "null_literal" && children[0].Content(source) == "null" &&
		(children[1].Type() == "identifier" || children[1].Type() == "default") && children[1].Content(source) == "default"
}

type enumNullSwitchCasePlan struct {
	clause *ast.CaseClause
	nulls  []bool
}

// An explicit null arm must not match an enum's still-nil static storage.
// Keep a single switch and the original bodies, using Java reference semantics
// so an erased any containing a typed nil behaves like the source reference.
func canonicalSourceEnumNullSwitch(statement *ast.SwitchStmt, selector, block *sitter.Node, source []byte, ctx Ctx, evidence *enumSwitchLoweringEvidence) *ast.SwitchStmt {
	if statement == nil || statement.Init != nil || statement.Tag == nil || statement.Body == nil {
		return statement
	}
	if block == nil || block.Type() != "switch_block" || !enumSwitchNodeWithinSource(block, source) || !enumSwitchBlockHasDirectNull(block) {
		return statement
	}
	owner := sourceEnumSwitchOwner(selector, source, ctx)
	if owner == nil || !evidence.matches(selector, block, ctx) {
		return statement
	}
	selection := block.Parent()
	if selection == nil || selector.Parent() == nil || !selector.Parent().Equal(selection) || (selection.Type() != "switch_statement" && selection.Type() != "switch_expression") {
		return statement
	}
	arms := nodeutil.SemanticNamedChildrenOf(block)
	if len(arms) != len(statement.Body.List) {
		// Stacked groups may have merged. Do not guess their case mapping.
		return statement
	}
	plans := make([]enumNullSwitchCasePlan, 0, len(arms))
	seen := make(map[*ast.CaseClause]bool)
	hasNull, expressions := false, 0
	for index, arm := range arms {
		if (arm.Type() != "switch_rule" && arm.Type() != "switch_block_statement_group") || !enumSwitchNodeWithinSource(arm, source) {
			return statement
		}
		clause, ok := statement.Body.List[index].(*ast.CaseClause)
		if !ok || clause == nil || seen[clause] {
			return statement
		}
		seen[clause] = true
		isDefault := false
		var nulls []bool
		for _, child := range nodeutil.SemanticNamedChildrenOf(arm) {
			if child.Type() != "switch_label" {
				continue
			}
			if combinedNullDefaultSwitchLabel(child, source, ctx) {
				isDefault, hasNull = true, true
				continue
			}
			labels := nodeutil.SemanticNamedChildrenOf(child)
			if len(labels) == 0 {
				isDefault = true
				continue
			}
			for _, label := range labels {
				if !enumSwitchNodeWithinSource(label, source) {
					return statement
				}
				if label.Type() == "null_literal" && label.Content(source) == "null" {
					nulls = append(nulls, true)
					hasNull = true
				} else if enumSwitchSourceConstant(label, source, owner, ctx) {
					if len(nulls) >= len(clause.List) || !enumSwitchStorageCorresponds(clause.List[len(nulls)], label, owner.FindFieldByName(label.Content(source)), owner, ctx, evidence) {
						return statement
					}
					nulls = append(nulls, false)
				} else {
					return statement
				}
			}
		}
		if isDefault {
			if len(clause.List) != 0 {
				return statement
			}
			nulls = nil
		} else {
			if len(nulls) == 0 || len(nulls) != len(clause.List) {
				return statement
			}
			for index, null := range nulls {
				if enumSwitchNilLiteral(clause.List[index]) != null {
					return statement
				}
			}
		}
		expressions += len(nulls)
		plans = append(plans, enumNullSwitchCasePlan{clause: clause, nulls: nulls})
	}
	if !hasNull || expressions == 0 {
		// A sole null/default clause already works, and needs no unused temp.
		return statement
	}

	// Complete every source/mapping proof before marking imports or changing AST.
	used := affineLoopUsedNames(selection, source, ctx)
	reserve := func(node ast.Node) bool {
		if name, ok := node.(*ast.Ident); ok {
			used[name.Name] = struct{}{}
		}
		return true
	}
	ast.Inspect(statement.Tag, reserve)
	ast.Inspect(statement.Body, reserve)
	name := synchronizedUniqueLocalName("__java2goEnumSwitchTag", used)
	conditions := make([][]ast.Expr, len(plans))
	for index, plan := range plans {
		if len(plan.nulls) == 0 {
			continue
		}
		conditions[index] = make([]ast.Expr, len(plan.nulls))
		for labelIndex, null := range plan.nulls {
			nullTest := stdjavaCall(ctx, "JavaReferenceEqual", ast.NewIdent(name), ast.NewIdent("nil"))
			var condition ast.Expr = nullTest
			if !null {
				condition = &ast.BinaryExpr{
					X:  &ast.UnaryExpr{Op: token.NOT, X: nullTest},
					Op: token.LAND,
					Y:  stdjavaCall(ctx, "JavaReferenceEqual", ast.NewIdent(name), plan.clause.List[labelIndex]),
				}
			}
			conditions[index][labelIndex] = condition
		}
	}
	statement.Init = &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(name)}, Tok: token.DEFINE, Rhs: []ast.Expr{statement.Tag}}
	statement.Tag = nil
	for index, plan := range plans {
		if len(plan.nulls) != 0 {
			plan.clause.List = conditions[index]
		}
	}
	return statement
}

func enumSwitchNilLiteral(expression ast.Expr) bool {
	for {
		parenthesized, ok := expression.(*ast.ParenExpr)
		if !ok {
			break
		}
		expression = parenthesized.X
	}
	name, ok := expression.(*ast.Ident)
	return ok && name.Name == "nil"
}

// Reuse declaration ownership, not the emitted spelling, to classify a source
// case. All original emitted storage expressions are retained by the plan.
func enumSwitchSourceConstant(label *sitter.Node, source []byte, owner *symbol.ClassScope, ctx Ctx) bool {
	if label == nil || label.Type() != "identifier" || owner == nil || !owner.IsEnum || owner.Class == nil || owner.Class.DeclarationNode == nil {
		return false
	}
	file := findFileScopeForClassScope(owner, ctx)
	if file == nil || file.ParsedRoot() == nil || file.ParsedRoot().HasError() || file.ParsedRoot().HasChanges() {
		return false
	}
	name := label.Content(source)
	found := false
	for _, constant := range owner.EnumConstants {
		if constant.Name != name {
			continue
		}
		field := constant.Field
		if found || field == nil || !field.IsStatic || !field.IsFinal || field.Name == "" || field.OriginalName != name || owner.FindFieldByName(name) != field || field.DeclarationNode == nil || field.DeclarationNode.Type() != "enum_constant" || !enumSwitchNodeWithinSource(field.DeclarationNode, file.Source) {
			return false
		}
		declaration := field.DeclarationNode
		body := declaration.Parent()
		if body == nil || body.Type() != "enum_body" || body.Parent() == nil || !body.Parent().Equal(owner.Class.DeclarationNode) {
			return false
		}
		declaredName := declaration.ChildByFieldName("name")
		if declaredName == nil || !enumSwitchNodeWithinSource(declaredName, file.Source) || declaredName.Content(file.Source) != name {
			return false
		}
		found = true
	}
	return found
}

func enumSwitchBlockHasDirectNull(block *sitter.Node) bool {
	for _, arm := range nodeutil.SemanticNamedChildrenOf(block) {
		if arm.Type() != "switch_rule" && arm.Type() != "switch_block_statement_group" {
			return false
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(arm) {
			if child.Type() != "switch_label" {
				continue
			}
			for _, label := range nodeutil.SemanticNamedChildrenOf(child) {
				if label.Type() == "null_literal" {
					return true
				}
			}
		}
	}
	return false
}

// Evidence is owned by one switch-lowering invocation and passed explicitly
// from label emission to dispatch proof. Neither AST nodes nor Ctx retain it.
type enumSwitchLoweringEvidence struct {
	selector, block, sourceRoot *sitter.Node
	file                        *symbol.FileScope
	storage                     map[ast.Expr]*enumSwitchStorageCertificate
}

func newEnumSwitchLoweringEvidence(selector, block *sitter.Node, ctx Ctx) *enumSwitchLoweringEvidence {
	if selector == nil || block == nil || ctx.currentFile == nil || ctx.currentFile.ParsedRoot() == nil {
		return nil
	}
	return &enumSwitchLoweringEvidence{selector: selector, block: block, file: ctx.currentFile, sourceRoot: ctx.currentFile.ParsedRoot()}
}

func (evidence *enumSwitchLoweringEvidence) matches(selector, block *sitter.Node, ctx Ctx) bool {
	return evidence != nil && selector != nil && block != nil && evidence.selector != nil && evidence.block != nil &&
		evidence.selector.Equal(selector) && evidence.block.Equal(block) && block.Type() == "switch_block" &&
		selector.Parent() != nil && block.Parent() != nil && selector.Parent().Equal(block.Parent()) &&
		evidence.file == ctx.currentFile &&
		ctx.currentFile != nil && evidence.sourceRoot != nil && ctx.currentFile.ParsedRoot() != nil &&
		evidence.sourceRoot.Equal(ctx.currentFile.ParsedRoot())
}

// A certificate records the exact compiler-selected storage shape. Its only
// owner is the local evidence object, so it cannot affect printing or walking.
type enumSwitchStorageCertificate struct {
	expression                                 ast.Expr
	carrier                                    *ast.Ident
	qualifier                                  *ast.Ident
	call                                       *ast.CallExpr
	star                                       *ast.StarExpr
	label, sourceRoot, declaration, ownerRoot  *sitter.Node
	field                                      *symbol.Definition
	owner                                      *symbol.ClassScope
	file, ownerFile                            *symbol.FileScope
	fieldName, storageName, packageName, alias string
}

func certifyEnumSwitchStorage(expression ast.Expr, label *sitter.Node, field *symbol.Definition, owner *symbol.ClassScope, ctx Ctx, evidence *enumSwitchLoweringEvidence) ast.Expr {
	if expression == nil || label == nil || field == nil || owner == nil || evidence == nil || !evidence.matches(evidence.selector, evidence.block, ctx) {
		return expression
	}
	declarationLabel := label.Parent()
	if declarationLabel == nil || declarationLabel.Parent() == nil || declarationLabel.Parent().Parent() == nil || !declarationLabel.Parent().Parent().Equal(evidence.block) {
		return expression
	}
	certificate := &enumSwitchStorageCertificate{expression: expression, label: label, field: field, owner: owner, file: ctx.currentFile, sourceRoot: ctx.currentFile.ParsedRoot(), declaration: field.DeclarationNode, fieldName: field.Name, packageName: findJavaPackageForClassScope(owner)}
	certificate.ownerFile = findFileScopeForClassScope(owner, ctx)
	if certificate.ownerFile == nil {
		return expression
	}
	certificate.ownerRoot = certificate.ownerFile.ParsedRoot()
	switch emitted := expression.(type) {
	case *ast.Ident:
		certificate.carrier = emitted
	case *ast.SelectorExpr:
		qualifier, ok := emitted.X.(*ast.Ident)
		if !ok || qualifier == nil || emitted.Sel == nil {
			return expression
		}
		certificate.carrier, certificate.qualifier = emitted.Sel, qualifier
		certificate.alias = qualifier.Name
	case *ast.StarExpr:
		call, ok := emitted.X.(*ast.CallExpr)
		if !ok || call == nil || len(call.Args) != 0 || call.Ellipsis.IsValid() {
			return expression
		}
		name, ok := call.Fun.(*ast.Ident)
		if !ok || name == nil {
			return expression
		}
		certificate.carrier, certificate.call, certificate.star = name, call, emitted
	default:
		return expression
	}
	// Existing Go object/binding metadata is never overwritten.
	if certificate.carrier == nil || certificate.carrier.Obj != nil {
		return expression
	}
	certificate.storageName = certificate.carrier.Name
	if evidence.storage == nil {
		evidence.storage = make(map[ast.Expr]*enumSwitchStorageCertificate)
	}
	evidence.storage[expression] = certificate
	return expression
}

// Check the exact compiler-produced root, its current shape, and live source
// declaration/namespace without re-emitting storage, allocating a helper name,
// marking imports, or invoking callbacks.
func enumSwitchStorageCorresponds(expression ast.Expr, label *sitter.Node, field *symbol.Definition, owner *symbol.ClassScope, ctx Ctx, evidence *enumSwitchLoweringEvidence) bool {
	if evidence == nil || !evidence.matches(evidence.selector, evidence.block, ctx) || label == nil || label.Parent() == nil || label.Parent().Parent() == nil || label.Parent().Parent().Parent() == nil || !label.Parent().Parent().Parent().Equal(evidence.block) {
		return false
	}
	var carrier, qualifier *ast.Ident
	var call *ast.CallExpr
	var star *ast.StarExpr
	switch emitted := expression.(type) {
	case *ast.Ident:
		carrier = emitted
	case *ast.SelectorExpr:
		var ok bool
		qualifier, ok = emitted.X.(*ast.Ident)
		if !ok || qualifier == nil {
			return false
		}
		carrier = emitted.Sel
	case *ast.StarExpr:
		var ok bool
		call, ok = emitted.X.(*ast.CallExpr)
		if !ok || call == nil || len(call.Args) != 0 || call.Ellipsis.IsValid() {
			return false
		}
		carrier, ok = call.Fun.(*ast.Ident)
		if !ok {
			return false
		}
		star = emitted
	default:
		return false
	}
	if carrier == nil || carrier.Obj != nil {
		return false
	}
	certificate, ok := evidence.storage[expression]
	if !ok || certificate == nil || certificate.expression != expression || certificate.carrier != carrier || certificate.qualifier != qualifier || certificate.call != call || certificate.star != star || certificate.field != field || certificate.owner != owner || field == nil || certificate.fieldName != field.Name || certificate.storageName != carrier.Name || certificate.label == nil || label == nil || !certificate.label.Equal(label) || certificate.declaration == nil || field.DeclarationNode == nil || !certificate.declaration.Equal(field.DeclarationNode) || certificate.file != ctx.currentFile || ctx.currentFile == nil || certificate.sourceRoot == nil || ctx.currentFile.ParsedRoot() == nil || !certificate.sourceRoot.Equal(ctx.currentFile.ParsedRoot()) {
		return false
	}
	ownerFile := findFileScopeForClassScope(owner, ctx)
	if ownerFile != certificate.ownerFile || ownerFile == nil || certificate.ownerRoot == nil || ownerFile.ParsedRoot() == nil || !certificate.ownerRoot.Equal(ownerFile.ParsedRoot()) || certificate.packageName != findJavaPackageForClassScope(owner) {
		return false
	}
	qualified := certificate.packageName != "" && certificate.packageName != ctx.currentFile.Package && !isJavaStdlibPackage(certificate.packageName)
	if qualified {
		return qualifier != nil && star == nil && qualifier.Name == certificate.alias && ctx.importAliases[certificate.packageName] == certificate.alias && certificate.alias != ""
	}
	if qualifier != nil || certificate.alias != "" || (star != nil) != staticFieldStorageShadowed(field, ctx) {
		return false
	}
	return true
}
