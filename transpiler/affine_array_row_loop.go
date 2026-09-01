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

const affineArrayRowMinimumSpan = 16

// affineArrayRowCallSite identifies one accessor whose Java affine index can be
// represented by an equal-span row slice and the hidden offset of a canonical
// column loop. The typed invocation IIFE is retained; only its final index
// expression changes.
type affineArrayRowCallSite struct {
	binding    *affineArrayLoopBinding
	rowSlice   *affineArrayRowSlice
	offsetName string
}

type affineArrayRowOperand struct {
	node       *sitter.Node
	definition *symbol.Definition
	key        string
	name       string
}

type affineArrayRowSliceKey struct {
	binding       *affineArrayLoopBinding
	rowDefinition *symbol.Definition
	rowLiteral    string
}

type affineArrayRowSlice struct {
	key          affineArrayRowSliceKey
	row          affineArrayRowOperand
	rowName      string
	productName  string
	baseName     string
	firstName    string
	lastName     string
	startIntName string
	sliceName    string
	hoistTarget  *sitter.Node
}

type affineArrayCanonicalColumnLoop struct {
	counterName   string
	counterGoName string
	start         affineArrayRowOperand
	end           affineArrayRowOperand
}

type affineArrayRowLoopPlan struct {
	canonical   affineArrayCanonicalColumnLoop
	startName   string
	endName     string
	span64Name  string
	spanName    string
	offsetName  string
	hoistTarget *sitter.Node
	rowSlices   []*affineArrayRowSlice
	callSites   map[affineArrayCallSiteKey]*affineArrayRowCallSite
	wholeRange  *affineArrayWholeRangePlan
}

type affineArrayWholeRangeBinding struct {
	binding     *affineArrayLoopBinding
	productName string
	lastName    string
}

// affineArrayWholeRangePlan is available only for an exact overflow-safe
// blocked-loop nest. Its runtime proof covers every row and column that the
// nest can visit, allowing inner scopes to carve slices without repeating the
// full Java-int overflow and backing-length proof.
type affineArrayWholeRangePlan struct {
	owner       *sitter.Node
	extent      affineArrayRowOperand
	step        affineArrayRowOperand
	extentName  string
	stepName    string
	lastRowName string
	bindings    []*affineArrayWholeRangeBinding
}

// affineArrayRowHoist is a pure, non-panicking preamble that belongs directly
// before one exact Java statement in the bounds-specialized affine branch.
// bindings guards against installing it into a separately rendered guarded
// copy of the same source loop.
type affineArrayRowHoist struct {
	bindings   []*affineArrayLoopBinding
	preamble   []ast.Stmt
	condition  ast.Expr
	fastPrefix []ast.Stmt
	fallback   ast.Stmt
}

// prepareAffineArrayRowLoop recognizes the deliberately narrow loop form used
// by dense numerical kernels:
//
//	for (int column = start; column < end; column++)
//
// Only accessors with an exact `column` column argument and a stable, simple int
// row are selected. Every selected backing view must already be proven non-null
// by an enclosing affine loop version; the guarded null branch never enters
// this pass.
func prepareAffineArrayRowLoop(node *sitter.Node, source []byte, ctx Ctx) (Ctx, *affineArrayRowLoopPlan) {
	if node == nil || node.Type() != "for_statement" || ctx.localScope == nil || len(ctx.affineArrayCallSites) == 0 || len(ctx.affineArrayNonNullBindings) == 0 {
		return ctx, nil
	}
	if parent := node.Parent(); parent != nil && parent.Type() == "labeled_statement" {
		return ctx, nil
	}
	if affineRowRuntimeIdentifiersShadowed(ctx, source) {
		return ctx, nil
	}
	body := node.ChildByFieldName("body")
	if body == nil || affineLoopContainsDeferredScope(body) || affineLoopContainsLabel(body) || affineRowLoopContainsNestedLoop(body) || affineRowLoopContainsControlFlow(body) {
		return ctx, nil
	}

	canonical, ok := analyzeAffineCanonicalColumnLoop(node, body, source, ctx)
	if !ok {
		return ctx, nil
	}

	candidates := collectAffineArrayCallCandidates(body, source, ctx)
	if len(candidates) < 2 {
		return ctx, nil
	}

	groups := make(map[affineArrayRowSliceKey]*affineArrayRowSlice)
	var orderedGroups []*affineArrayRowSlice
	callSites := make(map[affineArrayCallSiteKey]*affineArrayRowCallSite)
	for _, candidate := range candidates {
		callSiteKey := affineArrayCallSiteKey{start: candidate.node.StartByte(), end: candidate.node.EndByte()}
		binding := ctx.affineArrayCallSites[callSiteKey]
		if binding == nil || candidate.accessor == nil || candidate.accessor.View != binding.key.view {
			continue
		}
		if _, provenNonNull := ctx.affineArrayNonNullBindings[binding]; !provenNonNull {
			continue
		}
		argsNode := candidate.node.ChildByFieldName("arguments")
		args := nodeutil.NamedChildrenOf(argsNode)
		if candidate.accessor.RowParameter < 0 || candidate.accessor.RowParameter >= len(args) || candidate.accessor.ColumnParameter < 0 || candidate.accessor.ColumnParameter >= len(args) {
			continue
		}
		columnNode := unwrapParenthesizedExpressionNode(args[candidate.accessor.ColumnParameter])
		if columnNode == nil || columnNode.Type() != "identifier" || columnNode.Content(source) != canonical.counterName {
			continue
		}
		row, ok := affineSimpleStableIntOperand(args[candidate.accessor.RowParameter], body, canonical.counterName, source, ctx)
		if !ok {
			continue
		}

		groupKey := affineArrayRowSliceKey{binding: binding, rowDefinition: row.definition}
		if row.definition == nil {
			groupKey.rowLiteral = row.key
		}
		group := groups[groupKey]
		if group == nil {
			group = &affineArrayRowSlice{key: groupKey, row: row}
			groups[groupKey] = group
			orderedGroups = append(orderedGroups, group)
		}
		callSites[callSiteKey] = &affineArrayRowCallSite{binding: binding, rowSlice: group}
	}
	// One row slice can be useful, but the extra proof branch and slice setup are
	// most reliably profitable when at least two hot affine calls are removed.
	if len(callSites) < 2 {
		return ctx, nil
	}
	for _, candidate := range candidates {
		key := affineArrayCallSiteKey{start: candidate.node.StartByte(), end: candidate.node.EndByte()}
		if callSites[key] != nil && affineRowCallConditionallyEvaluated(candidate.node, body, source) {
			return ctx, nil
		}
	}

	usedNames := affineLoopUsedNames(node, source, ctx)
	for _, candidate := range candidates {
		for name := range affineIIFETypeIdentifiers(candidate.receiverClass, candidate.method, ctx) {
			usedNames[name] = struct{}{}
		}
	}
	for _, typeParameter := range inScopeTypeParameters(ctx) {
		usedNames[typeParameter] = struct{}{}
		usedNames[sanitizeGoIdent(typeParameter)] = struct{}{}
	}
	prefix := "__java2goAffineRow" + strconv.FormatUint(uint64(node.StartByte()), 10)
	plan := &affineArrayRowLoopPlan{
		canonical:  canonical,
		startName:  affineUniqueLocalName(prefix+"ColumnStart", usedNames),
		endName:    affineUniqueLocalName(prefix+"ColumnEnd", usedNames),
		span64Name: affineUniqueLocalName(prefix+"Span64", usedNames),
		spanName:   affineUniqueLocalName(prefix+"Span", usedNames),
		offsetName: affineUniqueLocalName(prefix+"Offset", usedNames),
		rowSlices:  orderedGroups,
		callSites:  callSites,
	}
	plan.hoistTarget = affineArrayRowHoistTarget(node, []affineArrayRowOperand{canonical.start, canonical.end}, affineRowPlanBindings(orderedGroups), source)
	for index, rowSlice := range orderedGroups {
		discriminator := strconv.Itoa(index)
		rowSlice.baseName = affineUniqueLocalName(prefix+"Base"+discriminator, usedNames)
		rowSlice.rowName = affineUniqueLocalName(prefix+"Row"+discriminator, usedNames)
		rowSlice.productName = affineUniqueLocalName(prefix+"Product64"+discriminator, usedNames)
		rowSlice.firstName = affineUniqueLocalName(prefix+"First"+discriminator, usedNames)
		rowSlice.lastName = affineUniqueLocalName(prefix+"Last"+discriminator, usedNames)
		rowSlice.startIntName = affineUniqueLocalName(prefix+"Start"+discriminator, usedNames)
		rowSlice.sliceName = affineUniqueLocalName(prefix+"Slice"+discriminator, usedNames)
		rowSlice.hoistTarget = affineArrayRowHoistTarget(node, []affineArrayRowOperand{rowSlice.row}, []*affineArrayLoopBinding{rowSlice.key.binding}, source)
		// A row preamble consumes the common start/span locals, so it cannot be
		// placed outside the common proof even when this individual binding and row
		// are invariant across more loops.
		rowSlice.hoistTarget = affineRowHoistTargetNoEarlierThan(rowSlice.hoistTarget, plan.hoistTarget)
	}
	for _, callSite := range callSites {
		callSite.offsetName = plan.offsetName
	}
	plan.wholeRange = analyzeAffineArrayWholeRange(node, plan, source, ctx, usedNames, prefix)

	rowCtx := ctx.Clone()
	rowCtx.affineArrayRowCallSites = cloneAffineArrayRowCallSites(ctx.affineArrayRowCallSites)
	for key, callSite := range callSites {
		rowCtx.affineArrayRowCallSites[key] = callSite
	}
	return rowCtx, plan
}

func cloneAffineArrayRowCallSites(source map[affineArrayCallSiteKey]*affineArrayRowCallSite) map[affineArrayCallSiteKey]*affineArrayRowCallSite {
	result := make(map[affineArrayCallSiteKey]*affineArrayRowCallSite, len(source))
	for key, callSite := range source {
		result[key] = callSite
	}
	return result
}

type affineBlockedDimension struct {
	blockLoop *sitter.Node
	extent    affineArrayRowOperand
	step      affineArrayRowOperand
}

func analyzeAffineArrayWholeRange(inner *sitter.Node, plan *affineArrayRowLoopPlan, source []byte, ctx Ctx, usedNames map[string]struct{}, prefix string) *affineArrayWholeRangePlan {
	if inner == nil || plan == nil || len(plan.rowSlices) == 0 {
		return nil
	}
	// An in-scope user class named Math can legally provide a side-effecting or
	// nonstandard min method. The blocked-range proof applies only to the
	// side-effect-free java.lang.Math intrinsic.
	if resolveClassScopeByQualifiedName(ctx, "Math") != nil {
		return nil
	}
	if ctx.localScope != nil && affineTreeDeclaresAnyName(ctx.localScope.DeclarationNode, map[string]struct{}{"Math": {}}, source) {
		return nil
	}
	if ctx.currentClass != nil && findFieldInHierarchy(ctx.currentClass, "Math", ctx) != nil {
		return nil
	}
	if ctx.currentFile != nil {
		for _, imported := range ctx.currentFile.Imports {
			if strings.HasSuffix(imported, ".Math") && imported != "java.lang.Math" {
				return nil
			}
		}
	}
	dimensions := make([]affineBlockedDimension, 0, len(plan.rowSlices)+1)
	columnDimension, ok := analyzeAffineBlockedDimension(inner, plan.canonical, source, ctx)
	if !ok {
		return nil
	}
	dimensions = append(dimensions, columnDimension)
	seenRows := make(map[*symbol.Definition]struct{})
	for _, rowSlice := range plan.rowSlices {
		if rowSlice == nil || rowSlice.row.name == "" || rowSlice.row.definition == nil {
			return nil
		}
		if _, exists := seenRows[rowSlice.row.definition]; exists {
			continue
		}
		seenRows[rowSlice.row.definition] = struct{}{}
		rowLoop, rowCanonical, found := affineCanonicalAncestorForCounter(inner, rowSlice.row.name, source, ctx)
		if !found {
			return nil
		}
		dimension, found := analyzeAffineBlockedDimension(rowLoop, rowCanonical, source, ctx)
		if !found {
			return nil
		}
		dimensions = append(dimensions, dimension)
	}

	extent := dimensions[0].extent
	step := dimensions[0].step
	if extent.definition == nil || step.definition == nil {
		return nil
	}
	for _, dimension := range dimensions[1:] {
		if dimension.extent.definition != extent.definition || dimension.step.definition != step.definition {
			return nil
		}
	}

	bindings := affineRowPlanBindings(plan.rowSlices)
	if len(bindings) == 0 {
		return nil
	}
	ownerStart := bindings[0].ownerLoopStart
	for _, binding := range bindings[1:] {
		if binding == nil || binding.ownerLoopStart != ownerStart {
			return nil
		}
	}
	var owner *sitter.Node
	for current := inner; current != nil; current = current.Parent() {
		if current.Type() == "for_statement" && current.StartByte() == ownerStart {
			owner = current
			break
		}
	}
	if owner == nil || !affineRowIdentifierStableInBody(owner, extent.name, source) || !affineRowIdentifierStableInBody(owner, step.name, source) {
		return nil
	}
	ownerMatched := false
	for _, dimension := range dimensions {
		if dimension.blockLoop != nil && dimension.blockLoop.StartByte() == owner.StartByte() {
			ownerMatched = true
			break
		}
	}
	if !ownerMatched {
		return nil
	}

	whole := &affineArrayWholeRangePlan{
		owner:       owner,
		extent:      extent,
		step:        step,
		extentName:  affineUniqueLocalName(prefix+"Extent", usedNames),
		stepName:    affineUniqueLocalName(prefix+"Step", usedNames),
		lastRowName: affineUniqueLocalName(prefix+"LastRow64", usedNames),
	}
	for index, binding := range bindings {
		discriminator := strconv.Itoa(index)
		whole.bindings = append(whole.bindings, &affineArrayWholeRangeBinding{
			binding:     binding,
			productName: affineUniqueLocalName(prefix+"WholeProduct64"+discriminator, usedNames),
			lastName:    affineUniqueLocalName(prefix+"WholeLast64"+discriminator, usedNames),
		})
	}
	return whole
}

func affineCanonicalAncestorForCounter(inner *sitter.Node, counterName string, source []byte, ctx Ctx) (*sitter.Node, affineArrayCanonicalColumnLoop, bool) {
	for current := inner.Parent(); current != nil; current = current.Parent() {
		if current.Type() != "for_statement" {
			continue
		}
		body := current.ChildByFieldName("body")
		canonical, ok := analyzeAffineCanonicalColumnLoop(current, body, source, ctx)
		if ok && canonical.counterName == counterName {
			return current, canonical, true
		}
	}
	return nil, affineArrayCanonicalColumnLoop{}, false
}

func analyzeAffineBlockedDimension(indexLoop *sitter.Node, index affineArrayCanonicalColumnLoop, source []byte, ctx Ctx) (affineBlockedDimension, bool) {
	if indexLoop == nil || index.start.name == "" || index.end.name == "" {
		return affineBlockedDimension{}, false
	}
	limitValue := affineLocalInitializerNode(ctx.localScope.DeclarationNode, index.end.name, source)
	stepNode, extentNode, ok := matchAffineBlockedLimit(limitValue, index.start.name, source)
	if !ok {
		return affineBlockedDimension{}, false
	}
	step, ok := affineIntIdentifierOperand(stepNode, ctx, source)
	if !ok {
		return affineBlockedDimension{}, false
	}
	extent, ok := affineIntIdentifierOperand(extentNode, ctx, source)
	if !ok {
		return affineBlockedDimension{}, false
	}
	var blockLoop *sitter.Node
	for current := indexLoop.Parent(); current != nil; current = current.Parent() {
		if current.Type() == "for_statement" && matchAffineBlockedOuterLoop(current, index.start.name, step.name, extent.name, source) {
			blockLoop = current
			break
		}
	}
	if blockLoop == nil || !affineRowIdentifierNotMutated(blockLoop, index.end.name, source) {
		return affineBlockedDimension{}, false
	}
	return affineBlockedDimension{blockLoop: blockLoop, extent: extent, step: step}, true
}

func affineLocalInitializerNode(method *sitter.Node, name string, source []byte) *sitter.Node {
	var value *sitter.Node
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil || value != nil {
			return
		}
		if node.Type() == "variable_declarator" {
			nameNode := node.ChildByFieldName("name")
			if nameNode != nil && nameNode.Content(source) == name {
				value = node.ChildByFieldName("value")
				if value == nil && node.NamedChildCount() > 1 {
					value = node.NamedChild(1)
				}
				return
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(method)
	return value
}

func matchAffineBlockedLimit(node *sitter.Node, blockName string, source []byte) (*sitter.Node, *sitter.Node, bool) {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil || node.Type() != "method_invocation" {
		return nil, nil, false
	}
	object := node.ChildByFieldName("object")
	name := node.ChildByFieldName("name")
	if object == nil || object.Content(source) != "Math" || name == nil || name.Content(source) != "min" {
		return nil, nil, false
	}
	args := nodeutil.NamedChildrenOf(node.ChildByFieldName("arguments"))
	if len(args) != 2 {
		return nil, nil, false
	}
	sum := unwrapParenthesizedExpressionNode(args[0])
	extent := unwrapParenthesizedExpressionNode(args[1])
	if sum == nil || sum.Type() != "binary_expression" || sum.ChildCount() < 3 || sum.Child(1).Content(source) != "+" || extent == nil || extent.Type() != "identifier" {
		return nil, nil, false
	}
	block := unwrapParenthesizedExpressionNode(sum.Child(0))
	step := unwrapParenthesizedExpressionNode(sum.Child(2))
	if block == nil || block.Type() != "identifier" || block.Content(source) != blockName || step == nil || step.Type() != "identifier" {
		return nil, nil, false
	}
	return step, extent, true
}

func affineIntIdentifierOperand(node *sitter.Node, ctx Ctx, source []byte) (affineArrayRowOperand, bool) {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil || node.Type() != "identifier" {
		return affineArrayRowOperand{}, false
	}
	name := node.Content(source)
	definition := affineLocalDefinition(name, ctx)
	if definition == nil || strings.TrimSpace(definition.OriginalType) != "int" {
		return affineArrayRowOperand{}, false
	}
	return affineArrayRowOperand{node: node, definition: definition, key: "identifier:" + name, name: name}, true
}

func matchAffineBlockedOuterLoop(loop *sitter.Node, blockName, stepName, extentName string, source []byte) bool {
	initNode := loop.ChildByFieldName("init")
	condition := unwrapParenthesizedExpressionNode(loop.ChildByFieldName("condition"))
	if initNode == nil || initNode.Type() != "local_variable_declaration" || condition == nil || condition.Type() != "binary_expression" || condition.ChildCount() < 3 {
		return false
	}
	declarator := initNode.ChildByFieldName("declarator")
	if declarator == nil {
		for _, child := range nodeutil.NamedChildrenOf(initNode) {
			if child.Type() == "variable_declarator" {
				if declarator != nil {
					return false
				}
				declarator = child
			}
		}
	}
	nameNode := (*sitter.Node)(nil)
	if declarator != nil {
		nameNode = declarator.ChildByFieldName("name")
	}
	if declarator == nil || nameNode == nil || nameNode.Content(source) != blockName {
		return false
	}
	initial := unwrapParenthesizedExpressionNode(declarator.ChildByFieldName("value"))
	if initial == nil || initial.Type() != "decimal_integer_literal" || initial.Content(source) != "0" {
		return false
	}
	left := unwrapParenthesizedExpressionNode(condition.Child(0))
	right := unwrapParenthesizedExpressionNode(condition.Child(2))
	if left == nil || left.Type() != "identifier" || left.Content(source) != blockName || condition.Child(1).Content(source) != "<" || right == nil || right.Type() != "identifier" || right.Content(source) != extentName {
		return false
	}
	var update *sitter.Node
	count := 0
	for index := 0; index < int(loop.ChildCount()); index++ {
		if loop.FieldNameForChild(index) == "update" {
			count++
			update = loop.Child(index)
		}
	}
	if count != 1 || update == nil || update.Type() != "assignment_expression" || update.ChildCount() < 3 {
		return false
	}
	updateLeft := unwrapParenthesizedExpressionNode(update.Child(0))
	updateRight := unwrapParenthesizedExpressionNode(update.Child(2))
	body := loop.ChildByFieldName("body")
	return body != nil && affineRowIdentifierNotMutated(body, blockName, source) &&
		updateLeft != nil && updateLeft.Type() == "identifier" && updateLeft.Content(source) == blockName &&
		update.Child(1).Content(source) == "+=" && updateRight != nil && updateRight.Type() == "identifier" && updateRight.Content(source) == stepName
}

func affineRowIdentifierNotMutated(root *sitter.Node, name string, source []byte) bool {
	stable := true
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil || !stable {
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
	walk(root)
	return stable
}

func analyzeAffineCanonicalColumnLoop(node, body *sitter.Node, source []byte, ctx Ctx) (affineArrayCanonicalColumnLoop, bool) {
	initNode := node.ChildByFieldName("init")
	conditionNode := unwrapParenthesizedExpressionNode(node.ChildByFieldName("condition"))
	var updateNode *sitter.Node
	updateCount := 0
	for index := 0; index < int(node.ChildCount()); index++ {
		if node.FieldNameForChild(index) != "update" {
			continue
		}
		updateCount++
		updateNode = node.Child(index)
	}
	if initNode == nil || initNode.Type() != "local_variable_declaration" || conditionNode == nil || conditionNode.Type() != "binary_expression" || updateNode == nil || updateNode.Type() != "update_expression" {
		return affineArrayCanonicalColumnLoop{}, false
	}
	if updateCount != 1 {
		return affineArrayCanonicalColumnLoop{}, false
	}
	typeNode := initNode.ChildByFieldName("type")
	if typeNode == nil || strings.TrimSpace(typeNode.Content(source)) != "int" {
		return affineArrayCanonicalColumnLoop{}, false
	}
	var declarator *sitter.Node
	for _, child := range nodeutil.NamedChildrenOf(initNode) {
		if child.Type() != "variable_declarator" {
			continue
		}
		if declarator != nil {
			return affineArrayCanonicalColumnLoop{}, false
		}
		declarator = child
	}
	if declarator == nil {
		declarator = initNode.ChildByFieldName("declarator")
	}
	if declarator == nil {
		return affineArrayCanonicalColumnLoop{}, false
	}
	nameNode := declarator.ChildByFieldName("name")
	startNode := declarator.ChildByFieldName("value")
	if startNode == nil && declarator.NamedChildCount() > 1 {
		startNode = declarator.NamedChild(1)
	}
	if nameNode == nil || startNode == nil {
		return affineArrayCanonicalColumnLoop{}, false
	}
	counterName := nameNode.Content(source)
	counterDef := affineLocalDefinition(counterName, ctx)
	if counterDef == nil || strings.TrimSpace(counterDef.OriginalType) != "int" || !affineRowIdentifierStableInBody(body, counterName, source) {
		return affineArrayCanonicalColumnLoop{}, false
	}

	if conditionNode.ChildCount() < 3 || conditionNode.Child(1).Content(source) != "<" {
		return affineArrayCanonicalColumnLoop{}, false
	}
	left := unwrapParenthesizedExpressionNode(conditionNode.Child(0))
	if left == nil || left.Type() != "identifier" || left.Content(source) != counterName {
		return affineArrayCanonicalColumnLoop{}, false
	}
	endNode := conditionNode.Child(2)

	if updateNode.ChildCount() < 2 || !updateNode.Child(0).IsNamed() || updateNode.Child(0).Type() != "identifier" || updateNode.Child(0).Content(source) != counterName || updateNode.Child(1).Content(source) != "++" {
		return affineArrayCanonicalColumnLoop{}, false
	}

	start, ok := affineSimpleStableIntOperand(startNode, body, counterName, source, ctx)
	if !ok {
		return affineArrayCanonicalColumnLoop{}, false
	}
	end, ok := affineSimpleStableIntOperand(endNode, body, counterName, source, ctx)
	if !ok {
		return affineArrayCanonicalColumnLoop{}, false
	}
	return affineArrayCanonicalColumnLoop{
		counterName:   counterName,
		counterGoName: sanitizeGoIdent(counterDef.Name),
		start:         start,
		end:           end,
	}, true
}

func affineSimpleStableIntOperand(node, body *sitter.Node, counterName string, source []byte, ctx Ctx) (affineArrayRowOperand, bool) {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return affineArrayRowOperand{}, false
	}
	switch node.Type() {
	case "identifier":
		name := node.Content(source)
		if name == counterName || !affineRowIdentifierStableInBody(body, name, source) {
			return affineArrayRowOperand{}, false
		}
		definition := affineLocalDefinition(name, ctx)
		if definition == nil || strings.TrimSpace(definition.OriginalType) != "int" {
			return affineArrayRowOperand{}, false
		}
		return affineArrayRowOperand{node: node, definition: definition, key: "identifier:" + name, name: name}, true
	case "decimal_integer_literal", "hex_integer_literal", "octal_integer_literal", "binary_integer_literal":
		javaType, ok := inferExprJavaType(node, ctx, source)
		if !ok || strings.TrimSpace(javaType) != "int" {
			return affineArrayRowOperand{}, false
		}
		return affineArrayRowOperand{node: node, key: "literal:" + node.Content(source)}, true
	default:
		return affineArrayRowOperand{}, false
	}
}

func affineRowIdentifierStableInBody(body *sitter.Node, name string, source []byte) bool {
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
	walk(body)
	return stable
}

func affineRowLoopContainsNestedLoop(body *sitter.Node) bool {
	if body == nil {
		return false
	}
	switch body.Type() {
	case "for_statement", "enhanced_for_statement", "while_statement", "do_statement":
		return true
	}
	for _, child := range nodeutil.NamedChildrenOf(body) {
		if affineRowLoopContainsNestedLoop(child) {
			return true
		}
	}
	return false
}

func affineRowLoopContainsControlFlow(body *sitter.Node) bool {
	if body == nil {
		return false
	}
	switch body.Type() {
	case "if_statement", "switch_statement", "switch_expression", "try_statement", "try_with_resources_statement", "synchronized_statement",
		"break_statement", "continue_statement", "return_statement", "throw_statement":
		return true
	}
	for _, child := range nodeutil.NamedChildrenOf(body) {
		if affineRowLoopContainsControlFlow(child) {
			return true
		}
	}
	return false
}

// Ternaries and short-circuit expressions remain legal in a numerical loop as
// long as they do not conditionally suppress a selected affine access. This is
// important for stencil kernels that compute unrelated neighbor columns with a
// ternary before performing their unconditional row accesses.
func affineRowCallConditionallyEvaluated(call, body *sitter.Node, source []byte) bool {
	for parent := call.Parent(); parent != nil && parent != body; parent = parent.Parent() {
		switch parent.Type() {
		case "ternary_expression":
			return true
		case "binary_expression":
			if parent.ChildCount() >= 3 {
				operator := parent.Child(1).Content(source)
				if operator == "&&" || operator == "||" {
					return true
				}
			}
		}
	}
	return false
}

func affineRowRuntimeIdentifiersShadowed(ctx Ctx, source []byte) bool {
	reserved := map[string]struct{}{"len": {}, "int64": {}}
	if ctx.currentFile != nil {
		if packageScope := symbol.GlobalScope.FindPackage(ctx.currentFile.Package); packageScope != nil {
			for _, file := range packageScope.Files {
				for name := range reserved {
					if affineRowFileDeclaresPackageIdent(file, name) {
						return true
					}
				}
			}
		} else {
			for name := range reserved {
				if affineRowFileDeclaresPackageIdent(ctx.currentFile, name) {
					return true
				}
			}
		}
	}
	if ctx.localScope != nil && affineTreeDeclaresAnyName(ctx.localScope.DeclarationNode, reserved, source) {
		return true
	}
	if affineTypeParametersDeclareAnyName(ctx, reserved) {
		return true
	}
	for _, alias := range ctx.importAliases {
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

// len and int64 are referenced bare by the row proof. Only declarations that
// reach the generated Go package block can shadow them across files. Parameters
// and locals in another method are function-scoped and must not disable an
// otherwise profitable loop in this method.
func affineRowFileDeclaresPackageIdent(file *symbol.FileScope, name string) bool {
	if file == nil || name == "" {
		return false
	}
	classes := file.TopLevelClasses
	if len(classes) == 0 && file.BaseClass != nil {
		classes = []*symbol.ClassScope{file.BaseClass}
	}
	for _, class := range classes {
		if affineRowClassDeclaresPackageIdent(class, name) {
			return true
		}
	}
	return false
}

func affineRowClassDeclaresPackageIdent(class *symbol.ClassScope, name string) bool {
	if class == nil {
		return false
	}
	declares := func(definition *symbol.Definition) bool {
		return definition != nil && sanitizeGoIdent(definition.Name) == name
	}
	if declares(class.Class) {
		return true
	}
	for _, field := range class.Fields {
		if field != nil && field.IsStatic && declares(field) {
			return true
		}
	}
	for _, method := range class.Methods {
		if method != nil && method.IsStatic && !method.Constructor && declares(method) {
			return true
		}
	}
	for _, constant := range class.EnumConstants {
		if sanitizeGoIdent(constant.Name) == name {
			return true
		}
	}
	for _, nested := range class.Subclasses {
		if affineRowClassDeclaresPackageIdent(nested, name) {
			return true
		}
	}
	return false
}

func affineRowPlanBindings(rowSlices []*affineArrayRowSlice) []*affineArrayLoopBinding {
	seen := make(map[*affineArrayLoopBinding]struct{})
	bindings := make([]*affineArrayLoopBinding, 0, len(rowSlices))
	for _, rowSlice := range rowSlices {
		if rowSlice == nil || rowSlice.key.binding == nil {
			continue
		}
		if _, exists := seen[rowSlice.key.binding]; exists {
			continue
		}
		seen[rowSlice.key.binding] = struct{}{}
		bindings = append(bindings, rowSlice.key.binding)
	}
	return bindings
}

// affineArrayRowHoistTarget walks only a direct block/for-loop ancestry. A
// preamble may cross an enclosing loop when each of its simple operands is
// unchanged throughout that loop. It never crosses the loop that owns an
// affine view binding: the cached backing slice and stride are declared inside
// that loop's versioned fast branch.
func affineArrayRowHoistTarget(inner *sitter.Node, operands []affineArrayRowOperand, bindings []*affineArrayLoopBinding, source []byte) *sitter.Node {
	target := inner
	for target != nil {
		block := target.Parent()
		if block == nil || block.Type() != "block" {
			break
		}
		loop := block.Parent()
		if loop == nil || loop.Type() != "for_statement" || loop.ChildByFieldName("body") != block {
			break
		}
		ownedHere := false
		for _, binding := range bindings {
			if binding != nil && binding.ownerLoopStart == loop.StartByte() {
				ownedHere = true
				break
			}
		}
		if ownedHere {
			break
		}
		invariant := true
		for _, operand := range operands {
			if operand.name != "" && !affineRowIdentifierStableInBody(loop, operand.name, source) {
				invariant = false
				break
			}
		}
		if !invariant {
			break
		}
		target = loop
	}
	return target
}

// affineRowHoistTargetNoEarlierThan preserves dominance between two preambles
// discovered independently from the same inner loop. If target encloses the
// prerequisite, using the prerequisite's target is the earliest legal scope.
func affineRowHoistTargetNoEarlierThan(target, prerequisite *sitter.Node) *sitter.Node {
	if target == nil {
		return prerequisite
	}
	if prerequisite == nil {
		return target
	}
	if target.StartByte() <= prerequisite.StartByte() && target.EndByte() >= prerequisite.EndByte() {
		return prerequisite
	}
	return target
}

func registerAffineArrayRowHoist(ctx Ctx, target *sitter.Node, hoist *affineArrayRowHoist) bool {
	if target == nil || hoist == nil || ctx.affineArrayRowHoists == nil || (hoist.condition == nil && len(hoist.preamble) == 0 && len(hoist.fastPrefix) == 0) || (hoist.condition != nil && hoist.fallback == nil) {
		return false
	}
	key := affineArrayCallSiteKey{start: target.StartByte(), end: target.EndByte()}
	ctx.affineArrayRowHoists[key] = append(ctx.affineArrayRowHoists[key], hoist)
	return true
}

// applyAffineArrayRowHoists consumes only hoists whose binding proofs are
// active in this exact parse context. Each tier branches outside the target
// loop, so the hot child is dominated by the proof and its row slices can use
// direct declarations. This preserves Go's bounds-check elimination; assigning
// a maybe-empty slice before the branch makes the compiler retain a hot check.
func applyAffineArrayRowHoists(node *sitter.Node, stmt ast.Stmt, ctx Ctx) []ast.Stmt {
	if node == nil || stmt == nil || ctx.affineArrayRowHoists == nil {
		return []ast.Stmt{stmt}
	}
	key := affineArrayCallSiteKey{start: node.StartByte(), end: node.EndByte()}
	hoists := ctx.affineArrayRowHoists[key]
	if len(hoists) == 0 {
		return []ast.Stmt{stmt}
	}
	result := stmt
	var prefix []ast.Stmt
	remaining := hoists[:0]
	for _, hoist := range hoists {
		eligible := hoist != nil && (hoist.condition == nil || hoist.fallback != nil)
		if eligible {
			for _, binding := range hoist.bindings {
				if _, proven := ctx.affineArrayNonNullBindings[binding]; !proven {
					eligible = false
					break
				}
			}
		}
		if eligible {
			fast := append([]ast.Stmt{}, hoist.fastPrefix...)
			fast = append(fast, result)
			if hoist.condition == nil {
				result = &ast.BlockStmt{List: fast}
			} else {
				result = &ast.IfStmt{
					Cond: hoist.condition,
					Body: &ast.BlockStmt{List: fast},
					Else: &ast.BlockStmt{List: []ast.Stmt{hoist.fallback}},
				}
			}
			prefix = append(prefix, hoist.preamble...)
			continue
		}
		remaining = append(remaining, hoist)
	}
	if len(remaining) == 0 {
		delete(ctx.affineArrayRowHoists, key)
	} else {
		ctx.affineArrayRowHoists[key] = remaining
	}
	return append(prefix, result)
}

// lowerAffineArrayRowLoop emits a pure bounds proof followed by two equivalent
// loop copies. The specialized branch is entered only when every selected Java
// int32 index is a contiguous, non-wrapping, in-bounds interval. Otherwise the
// already-correct flat affine loop handles the original exception timing.
func lowerAffineArrayRowLoop(
	node *sitter.Node,
	source []byte,
	baseCtx Ctx,
	rowCtx Ctx,
	plan *affineArrayRowLoopPlan,
	ordinaryInit ast.Stmt,
	ordinaryCond ast.Expr,
	ordinaryPost ast.Stmt,
) ast.Stmt {
	bodyNode := node.ChildByFieldName("body")
	fallbackCtx := baseCtx.Clone()
	fallbackCtx.localScope = cloneLocalScopeDefinition(baseCtx.localScope)
	fallbackCtx.suppressUnsupportedDiagnostics = true

	specializedBody := ParseStmt(bodyNode, source, rowCtx).(*ast.BlockStmt)
	fallbackBody := ParseStmt(bodyNode, source, fallbackCtx).(*ast.BlockStmt)

	if initNode := node.ChildByFieldName("init"); initNode != nil && initNode.Type() == "local_variable_declaration" {
		fallbackBody.List = append(unusedLocalDiscardStatements(ordinaryInit, node, source), fallbackBody.List...)
	}
	specializedBody.List = append([]ast.Stmt{&ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: plan.canonical.counterGoName}},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{&ast.BinaryExpr{
			X:  &ast.Ident{Name: plan.startName},
			Op: token.ADD,
			Y:  &ast.CallExpr{Fun: &ast.Ident{Name: "int32"}, Args: []ast.Expr{&ast.Ident{Name: plan.offsetName}}},
		}},
	}}, specializedBody.List...)
	specializedLoop := &ast.RangeStmt{
		Key:  &ast.Ident{Name: plan.offsetName},
		Tok:  token.DEFINE,
		X:    &ast.Ident{Name: plan.rowSlices[0].sliceName},
		Body: specializedBody,
	}
	fallbackLoop := &ast.ForStmt{Init: ordinaryInit, Cond: ordinaryCond, Post: ordinaryPost, Body: fallbackBody}

	int32Call := func(value ast.Expr) ast.Expr {
		return &ast.CallExpr{Fun: &ast.Ident{Name: "int32"}, Args: []ast.Expr{value}}
	}
	int64Call := func(value ast.Expr) ast.Expr {
		return &ast.CallExpr{Fun: &ast.Ident{Name: "int64"}, Args: []ast.Expr{value}}
	}
	intCall := func(value ast.Expr) ast.Expr {
		return &ast.CallExpr{Fun: &ast.Ident{Name: "int"}, Args: []ast.Expr{value}}
	}
	assign := func(name string, value ast.Expr) ast.Stmt {
		return &ast.AssignStmt{Lhs: []ast.Expr{&ast.Ident{Name: name}}, Tok: token.DEFINE, Rhs: []ast.Expr{value}}
	}

	type rowHoistTier struct {
		target     *sitter.Node
		bindings   []*affineArrayLoopBinding
		bindingSet map[*affineArrayLoopBinding]struct{}
		preamble   []ast.Stmt
		condition  ast.Expr
		fastPrefix []ast.Stmt
	}
	tiers := make(map[affineArrayCallSiteKey]*rowHoistTier)
	var orderedTiers []*rowHoistTier
	tierFor := func(target *sitter.Node) *rowHoistTier {
		key := affineArrayCallSiteKey{start: target.StartByte(), end: target.EndByte()}
		tier := tiers[key]
		if tier == nil {
			tier = &rowHoistTier{target: target, bindingSet: make(map[*affineArrayLoopBinding]struct{})}
			tiers[key] = tier
			orderedTiers = append(orderedTiers, tier)
		}
		return tier
	}
	addBinding := func(tier *rowHoistTier, binding *affineArrayLoopBinding) {
		if binding == nil {
			return
		}
		if _, exists := tier.bindingSet[binding]; exists {
			return
		}
		tier.bindingSet[binding] = struct{}{}
		tier.bindings = append(tier.bindings, binding)
	}
	appendCondition := func(tier *rowHoistTier, condition ast.Expr) {
		if tier.condition == nil {
			tier.condition = condition
		} else {
			tier.condition = affineRowAnd(tier.condition, condition)
		}
	}
	wholeRange := plan.wholeRange != nil
	if wholeRange {
		whole := plan.wholeRange
		tier := tierFor(whole.owner)
		for _, binding := range whole.bindings {
			addBinding(tier, binding.binding)
		}
		tier.preamble = append(tier.preamble,
			assign(whole.extentName, int32Call(ParseExpr(whole.extent.node, source, baseCtx))),
			assign(whole.stepName, int32Call(ParseExpr(whole.step.node, source, baseCtx))),
			assign(whole.lastRowName, &ast.BinaryExpr{
				X:  int64Call(&ast.Ident{Name: whole.extentName}),
				Op: token.SUB,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "1"},
			}),
		)
		appendCondition(tier, &ast.BinaryExpr{X: &ast.Ident{Name: whole.extentName}, Op: token.GTR, Y: &ast.BasicLit{Kind: token.INT, Value: "0"}})
		appendCondition(tier, &ast.BinaryExpr{X: &ast.Ident{Name: whole.stepName}, Op: token.GTR, Y: &ast.BasicLit{Kind: token.INT, Value: "0"}})
		appendCondition(tier, &ast.BinaryExpr{
			X: &ast.BinaryExpr{
				X: &ast.BinaryExpr{
					X:  int64Call(&ast.Ident{Name: whole.extentName}),
					Op: token.ADD,
					Y:  int64Call(&ast.Ident{Name: whole.stepName}),
				},
				Op: token.SUB,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "1"},
			},
			Op: token.LEQ,
			Y:  &ast.BasicLit{Kind: token.INT, Value: "2147483647"},
		})
		for _, wholeBinding := range whole.bindings {
			array := &ast.Ident{Name: wholeBinding.binding.arrayName}
			tier.preamble = append(tier.preamble,
				assign(wholeBinding.productName, &ast.BinaryExpr{
					X:  &ast.Ident{Name: whole.lastRowName},
					Op: token.MUL,
					Y:  int64Call(&ast.Ident{Name: wholeBinding.binding.strideName}),
				}),
				assign(wholeBinding.lastName, &ast.BinaryExpr{
					X:  &ast.Ident{Name: wholeBinding.productName},
					Op: token.ADD,
					Y:  &ast.Ident{Name: whole.lastRowName},
				}),
			)
			appendCondition(tier, &ast.BinaryExpr{X: &ast.Ident{Name: wholeBinding.productName}, Op: token.GEQ, Y: &ast.BasicLit{Kind: token.INT, Value: "0"}})
			// Reserve one int32 unit for the exclusive slice high on 32-bit Go targets.
			appendCondition(tier, &ast.BinaryExpr{X: &ast.Ident{Name: wholeBinding.lastName}, Op: token.LEQ, Y: &ast.BasicLit{Kind: token.INT, Value: "2147483646"}})
			appendCondition(tier, &ast.BinaryExpr{
				X:  &ast.Ident{Name: wholeBinding.lastName},
				Op: token.LSS,
				Y:  int64Call(&ast.CallExpr{Fun: &ast.Ident{Name: "len"}, Args: []ast.Expr{array}}),
			})
		}
	}

	commonTier := tierFor(plan.hoistTarget)
	for _, binding := range affineRowPlanBindings(plan.rowSlices) {
		addBinding(commonTier, binding)
	}
	commonTier.preamble = append(commonTier.preamble,
		assign(plan.startName, int32Call(ParseExpr(plan.canonical.start.node, source, baseCtx))),
		assign(plan.endName, int32Call(ParseExpr(plan.canonical.end.node, source, baseCtx))),
		assign(plan.span64Name, &ast.BinaryExpr{
			X:  int64Call(&ast.Ident{Name: plan.endName}),
			Op: token.SUB,
			Y:  int64Call(&ast.Ident{Name: plan.startName}),
		}),
		assign(plan.spanName, intCall(&ast.Ident{Name: plan.span64Name})),
	)
	minimumSpanCondition := ast.Expr(&ast.BinaryExpr{
		X:  &ast.Ident{Name: plan.span64Name},
		Op: token.GEQ,
		Y:  &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(affineArrayRowMinimumSpan)},
	})
	appendCondition(commonTier, affineRowAnd(minimumSpanCondition, &ast.BinaryExpr{
		X:  &ast.Ident{Name: plan.span64Name},
		Op: token.LEQ,
		Y:  &ast.BasicLit{Kind: token.INT, Value: "2147483647"},
	}))

	for _, rowSlice := range plan.rowSlices {
		tier := tierFor(rowSlice.hoistTarget)
		addBinding(tier, rowSlice.key.binding)
		rowExpr := int32Call(ParseExpr(rowSlice.row.node, source, baseCtx))
		array := &ast.Ident{Name: rowSlice.key.binding.arrayName}
		if wholeRange {
			tier.fastPrefix = append(tier.fastPrefix,
				assign(rowSlice.baseName, &ast.BinaryExpr{
					X:  intCall(rowExpr),
					Op: token.MUL,
					Y:  intCall(&ast.Ident{Name: rowSlice.key.binding.strideName}),
				}),
				assign(rowSlice.startIntName, &ast.BinaryExpr{
					X:  &ast.Ident{Name: rowSlice.baseName},
					Op: token.ADD,
					Y:  intCall(&ast.Ident{Name: plan.startName}),
				}),
				assign(rowSlice.sliceName, &ast.SliceExpr{
					X:    array,
					Low:  &ast.Ident{Name: rowSlice.startIntName},
					High: &ast.BinaryExpr{X: &ast.Ident{Name: rowSlice.startIntName}, Op: token.ADD, Y: &ast.Ident{Name: plan.spanName}},
				}),
			)
			continue
		}
		tier.preamble = append(tier.preamble,
			assign(rowSlice.rowName, rowExpr),
			assign(rowSlice.productName, &ast.BinaryExpr{
				X:  int64Call(&ast.Ident{Name: rowSlice.rowName}),
				Op: token.MUL,
				Y:  int64Call(&ast.Ident{Name: rowSlice.key.binding.strideName}),
			}),
			assign(rowSlice.baseName, &ast.BinaryExpr{X: &ast.Ident{Name: rowSlice.rowName}, Op: token.MUL, Y: &ast.Ident{Name: rowSlice.key.binding.strideName}}),
			assign(rowSlice.firstName, &ast.BinaryExpr{
				X:  int64Call(&ast.Ident{Name: rowSlice.baseName}),
				Op: token.ADD,
				Y:  int64Call(&ast.Ident{Name: plan.startName}),
			}),
			assign(rowSlice.lastName, &ast.BinaryExpr{
				X: &ast.BinaryExpr{
					X:  int64Call(&ast.Ident{Name: rowSlice.baseName}),
					Op: token.ADD,
					Y:  int64Call(&ast.Ident{Name: plan.endName}),
				},
				Op: token.SUB,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "1"},
			}),
		)

		first := &ast.Ident{Name: rowSlice.firstName}
		last := &ast.Ident{Name: rowSlice.lastName}
		product := &ast.Ident{Name: rowSlice.productName}
		base := &ast.Ident{Name: rowSlice.baseName}
		checks := []ast.Expr{
			&ast.BinaryExpr{X: product, Op: token.GEQ, Y: &ast.BasicLit{Kind: token.INT, Value: "-2147483648"}},
			&ast.BinaryExpr{X: product, Op: token.LEQ, Y: &ast.BasicLit{Kind: token.INT, Value: "2147483647"}},
			&ast.BinaryExpr{X: int64Call(base), Op: token.EQL, Y: product},
			&ast.BinaryExpr{X: first, Op: token.GEQ, Y: &ast.BasicLit{Kind: token.INT, Value: "-2147483648"}},
			&ast.BinaryExpr{X: first, Op: token.LEQ, Y: &ast.BasicLit{Kind: token.INT, Value: "2147483647"}},
			&ast.BinaryExpr{X: last, Op: token.GEQ, Y: &ast.BasicLit{Kind: token.INT, Value: "-2147483648"}},
			&ast.BinaryExpr{X: last, Op: token.LEQ, Y: &ast.BasicLit{Kind: token.INT, Value: "2147483647"}},
			&ast.BinaryExpr{X: first, Op: token.GEQ, Y: &ast.BasicLit{Kind: token.INT, Value: "0"}},
			&ast.BinaryExpr{
				X:  last,
				Op: token.LSS,
				Y:  int64Call(&ast.CallExpr{Fun: &ast.Ident{Name: "len"}, Args: []ast.Expr{array}}),
			},
		}
		valid := ast.Expr(checks[0])
		for _, check := range checks[1:] {
			valid = affineRowAnd(valid, check)
		}
		appendCondition(tier, valid)
		tier.fastPrefix = append(tier.fastPrefix,
			assign(rowSlice.startIntName, intCall(&ast.Ident{Name: rowSlice.firstName})),
			assign(rowSlice.sliceName, &ast.SliceExpr{
				X:    array,
				Low:  &ast.Ident{Name: rowSlice.startIntName},
				High: &ast.BinaryExpr{X: &ast.Ident{Name: rowSlice.startIntName}, Op: token.ADD, Y: &ast.Ident{Name: plan.spanName}},
			}),
		)
	}

	registered := true
	for _, tier := range orderedTiers {
		var fallback ast.Stmt
		if tier.condition != nil && tier.target.StartByte() == node.StartByte() && tier.target.EndByte() == node.EndByte() {
			fallback = fallbackLoop
		} else if tier.condition != nil {
			fallbackCtx := baseCtx.Clone()
			fallbackCtx.localScope = cloneLocalScopeDefinition(baseCtx.localScope)
			fallbackCtx.suppressUnsupportedDiagnostics = true
			fallbackCtx.disableAffineArrayRowSpecialization = true
			fallbackCtx.affineArrayRowHoists = make(map[affineArrayCallSiteKey][]*affineArrayRowHoist)
			fallback = ParseStmt(tier.target, source, fallbackCtx)
		}
		registered = registerAffineArrayRowHoist(baseCtx, tier.target, &affineArrayRowHoist{
			bindings:   tier.bindings,
			preamble:   tier.preamble,
			condition:  tier.condition,
			fastPrefix: tier.fastPrefix,
			fallback:   fallback,
		}) && registered
	}
	if !registered {
		return fallbackLoop
	}
	return specializedLoop
}

func affineRowAnd(left, right ast.Expr) ast.Expr {
	return &ast.BinaryExpr{X: left, Op: token.LAND, Y: right}
}
