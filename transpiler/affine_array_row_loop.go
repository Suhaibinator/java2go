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
}

type affineArrayCanonicalColumnLoop struct {
	counterName   string
	counterGoName string
	start         affineArrayRowOperand
	end           affineArrayRowOperand
}

type affineArrayRowLoopPlan struct {
	canonical  affineArrayCanonicalColumnLoop
	startName  string
	endName    string
	span64Name string
	spanName   string
	offsetName string
	rowSlices  []*affineArrayRowSlice
	callSites  map[affineArrayCallSiteKey]*affineArrayRowCallSite
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
	for index, rowSlice := range orderedGroups {
		discriminator := strconv.Itoa(index)
		rowSlice.baseName = affineUniqueLocalName(prefix+"Base"+discriminator, usedNames)
		rowSlice.rowName = affineUniqueLocalName(prefix+"Row"+discriminator, usedNames)
		rowSlice.productName = affineUniqueLocalName(prefix+"Product64"+discriminator, usedNames)
		rowSlice.firstName = affineUniqueLocalName(prefix+"First"+discriminator, usedNames)
		rowSlice.lastName = affineUniqueLocalName(prefix+"Last"+discriminator, usedNames)
		rowSlice.startIntName = affineUniqueLocalName(prefix+"Start"+discriminator, usedNames)
		rowSlice.sliceName = affineUniqueLocalName(prefix+"Slice"+discriminator, usedNames)
	}
	for _, callSite := range callSites {
		callSite.offsetName = plan.offsetName
	}

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
	set := func(name string, value ast.Expr) ast.Stmt {
		return &ast.AssignStmt{Lhs: []ast.Expr{&ast.Ident{Name: name}}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}}
	}
	declare := func(name, typeName string) ast.Stmt {
		return &ast.DeclStmt{Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{&ast.ValueSpec{
				Names: []*ast.Ident{{Name: name}},
				Type:  &ast.Ident{Name: typeName},
			}},
		}}
	}

	statements := []ast.Stmt{
		assign(plan.startName, int32Call(ParseExpr(plan.canonical.start.node, source, baseCtx))),
		assign(plan.endName, int32Call(ParseExpr(plan.canonical.end.node, source, baseCtx))),
		assign(plan.span64Name, &ast.BinaryExpr{
			X:  int64Call(&ast.Ident{Name: plan.endName}),
			Op: token.SUB,
			Y:  int64Call(&ast.Ident{Name: plan.startName}),
		}),
	}

	minimumSpanCondition := ast.Expr(&ast.BinaryExpr{
		X:  &ast.Ident{Name: plan.span64Name},
		Op: token.GEQ,
		Y:  &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(affineArrayRowMinimumSpan)},
	})
	condition := affineRowAnd(minimumSpanCondition, &ast.BinaryExpr{
		X:  &ast.Ident{Name: plan.span64Name},
		Op: token.LEQ,
		Y:  &ast.BasicLit{Kind: token.INT, Value: "2147483647"},
	})
	var rangeStatements []ast.Stmt

	for _, rowSlice := range plan.rowSlices {
		rowExpr := int32Call(ParseExpr(rowSlice.row.node, source, baseCtx))
		statements = append(statements,
			declare(rowSlice.rowName, "int32"),
			declare(rowSlice.productName, "int64"),
			declare(rowSlice.baseName, "int32"),
			declare(rowSlice.firstName, "int64"),
			declare(rowSlice.lastName, "int64"),
		)
		rangeStatements = append(rangeStatements,
			set(rowSlice.rowName, rowExpr),
			set(rowSlice.productName, &ast.BinaryExpr{
				X:  int64Call(&ast.Ident{Name: rowSlice.rowName}),
				Op: token.MUL,
				Y:  int64Call(&ast.Ident{Name: rowSlice.key.binding.strideName}),
			}),
			set(rowSlice.baseName, &ast.BinaryExpr{X: &ast.Ident{Name: rowSlice.rowName}, Op: token.MUL, Y: &ast.Ident{Name: rowSlice.key.binding.strideName}}),
			set(rowSlice.firstName, &ast.BinaryExpr{
				X:  int64Call(&ast.Ident{Name: rowSlice.baseName}),
				Op: token.ADD,
				Y:  int64Call(&ast.Ident{Name: plan.startName}),
			}),
			set(rowSlice.lastName, &ast.BinaryExpr{
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
		array := &ast.Ident{Name: rowSlice.key.binding.arrayName}
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
		for _, check := range checks {
			condition = affineRowAnd(condition, check)
		}
	}
	statements = append(statements, &ast.IfStmt{
		Cond: minimumSpanCondition,
		Body: &ast.BlockStmt{List: rangeStatements},
	})

	fastStatements := []ast.Stmt{assign(plan.spanName, intCall(&ast.Ident{Name: plan.span64Name}))}
	for _, rowSlice := range plan.rowSlices {
		fastStatements = append(fastStatements,
			assign(rowSlice.startIntName, intCall(&ast.Ident{Name: rowSlice.firstName})),
			assign(rowSlice.sliceName, &ast.SliceExpr{
				X:    &ast.Ident{Name: rowSlice.key.binding.arrayName},
				Low:  &ast.Ident{Name: rowSlice.startIntName},
				High: &ast.BinaryExpr{X: &ast.Ident{Name: rowSlice.startIntName}, Op: token.ADD, Y: &ast.Ident{Name: plan.spanName}},
			}),
		)
	}
	fastStatements = append(fastStatements, specializedLoop)
	statements = append(statements, &ast.IfStmt{
		Cond: condition,
		Body: &ast.BlockStmt{List: fastStatements},
		Else: &ast.BlockStmt{List: []ast.Stmt{fallbackLoop}},
	})
	return &ast.BlockStmt{List: statements}
}

func affineRowAnd(left, right ast.Expr) ast.Expr {
	return &ast.BinaryExpr{X: left, Op: token.LAND, Y: right}
}
