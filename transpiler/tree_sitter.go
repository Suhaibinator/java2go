package transpiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
	sitter "github.com/smacker/go-tree-sitter"
)

// Inspect is a function for debugging that prints out every named child of a
// given node and the source code for that child
func Inspect(node *sitter.Node, source []byte) {
	for _, c := range nodeutil.NamedChildrenOf(node) {
		fmt.Println(c, c.Content(source))
	}
}

// CapitalizeIdent capitalizes the first letter of a `*ast.Ident` to mark the
// result as a public method or field
func CapitalizeIdent(in *ast.Ident) *ast.Ident {
	return &ast.Ident{Name: symbol.Uppercase(in.Name)}
}

// LowercaseIdent lowercases the first letter of a `*ast.Ident` to mark the
// result as a private method or field
func LowercaseIdent(in *ast.Ident) *ast.Ident {
	return &ast.Ident{Name: symbol.Lowercase(in.Name)}
}

func identFromNode(node *sitter.Node, source []byte) *ast.Ident {
	if node == nil {
		return &ast.Ident{}
	}
	return &ast.Ident{Name: sanitizeGoIdent(node.Content(source))}
}

// A Ctx is all the context that is needed to parse a single source file
type Ctx struct {
	// Used to generate the names of all the methods, as well as the names
	// of the constructors
	className string

	// Symbols for the current file being parsed
	currentFile  *symbol.FileScope
	currentClass *symbol.ClassScope

	// The symbols of the current
	localScope *symbol.Definition

	// Used when generating arrays, because in Java, these are defined as
	// arrType[] varName = {item, item, item}, and no class name data is defined
	// Can either be of type `*ast.Ident` or `*ast.StarExpr`
	lastType ast.Expr

	// Expected type from variable declaration, used for diamond operator inference
	expectedType string
	// expectedTypeRoot records the expression that is actually in the target-typed
	// assignment, invocation, or return context. Java only treats a poly
	// conditional as target-typed when the conditional itself is that expression
	// (parentheses aside); carrying expectedType alone through enclosing binary
	// expressions can otherwise change the conditional's standalone type.
	expectedTypeRoot *sitter.Node

	// Additional type parameters synthesized for a generated top-level function.
	// Java raw generic parameters can accept every instantiation, while Go has no
	// equivalent raw generic type. Static Java methods that receive a raw generic
	// are therefore emitted as generic Go functions, with these parameters kept
	// separate from the immutable Java symbol definition.
	syntheticTypeParameters []symbol.TypeParam
	// Rewritten Java types for raw generic formal parameters, keyed by both the
	// Java and generated Go parameter names. Keeping these in the conversion
	// context lets body type inference observe the same synthesized arguments as
	// the emitted function signature.
	rawGenericParameterTypes map[string]string

	// Java package -> Go alias map for generated imports in the current output file
	importAliases map[string]string
	// Tracks which imported Java packages are actually used by generated nodes
	usedImports map[string]bool

	// When set, return statements are rewritten into assignments + bare return
	// for lowered try/catch/finally closures.
	tryReturnTarget *tryReturnTarget
	// tryControlBoundary is the Java source block represented by the current
	// lowered try/catch/finally func literal. A break or continue whose target is
	// outside this block cannot be emitted directly in Go: func literals are a
	// control-flow boundary even when they are lexically nested in a loop. Such a
	// transfer is recorded on tryReturnTarget and replayed after the closure has
	// run (and, importantly, after finally has had a chance to override it).
	tryControlBoundary *sitter.Node
	// doWhileContinueTargets maps active Java do-while statements to their
	// generated condition-phase labels. Java continue targets this phase, while
	// native Go continue would incorrectly skip a trailing condition guard.
	doWhileContinueTargets map[javaControlTargetKey]*doWhileContinueTarget
	// javaLabelTargets tracks whether a Java label still needs a corresponding
	// Go loop label after target-specific rewrites. A labeled continue to a
	// do-while uses the internal condition label instead, and Java permits labels
	// that are never referenced whereas Go rejects unused labels.
	javaLabelTargets map[javaControlTargetKey]*javaLabelTarget
	// suppressUnsupportedDiagnostics is used only while rendering the guarded
	// copy of a versioned affine loop. The first copy already reported (or, in
	// strict mode, failed on) the same source node; the second must still emit its
	// placeholder AST without recording the diagnostic twice.
	suppressUnsupportedDiagnostics bool
	// disableAffineArrayRowSpecialization is set only while rendering the cold
	// fallback subtree for a hoisted row-proof tier. Ordinary affine indexing
	// remains enabled, but no descendant may register another row specialization.
	disableAffineArrayRowSpecialization bool

	// Collects top-level declarations synthesized while parsing nested contexts,
	// such as anonymous (non-SAM) and local classes that must be hoisted to file
	// scope. It is a shared pointer so cloned contexts append to the same slice.
	hoistedDecls *[]ast.Decl

	// Monotonic counter used to give synthesized anonymous/local classes unique
	// names within a file. Shared via pointer across cloned contexts.
	anonClassCounter *int
	// Monotonic counter used for function-scoped control labels whose source node
	// may be rendered more than once (for example affine fast/guarded loop
	// versions). Shared across cloned contexts so every emitted declaration in a
	// generated file receives a distinct name.
	controlLabelCounter *int

	// Maps a local class's Java name to the synthesized file-scope struct it was
	// hoisted into, so `new LocalClass(...)` in the same method body resolves to
	// the hoisted struct and threads captured locals. Shared via pointer.
	localClasses map[string]*localClassInfo

	// Affine array bindings are installed only while parsing approved ordinary
	// for-loop bodies. Call sites are keyed by their exact source span so a
	// same-spelled selector in a header, closure, or shadowing scope cannot be
	// rewritten accidentally.
	affineArrayBindings  map[affineArrayBindingKey]*affineArrayLoopBinding
	affineArrayCallSites map[affineArrayCallSiteKey]*affineArrayLoopBinding
	// affineArrayNonNullBindings contains only bindings proven non-null by the
	// currently enclosing versioned-loop branches. This is binding-specific: an
	// inner fast branch may prove its own receiver while an inherited binding is
	// still on an outer guarded path.
	affineArrayNonNullBindings map[*affineArrayLoopBinding]struct{}
	// affineArrayRowCallSites is installed only while rendering the proven
	// bounds-specialized copy of a canonical column loop. Calls absent from this
	// exact source-span map retain the ordinary flat affine index.
	affineArrayRowCallSites map[affineArrayCallSiteKey]*affineArrayRowCallSite
	// affineArrayRowHoists carries pure row-proof preambles from a recognized
	// inner column loop back to the enclosing lexical block that can evaluate
	// them least often. The map is shared by cloned contexts while a method body
	// is rendered; exact source spans keep duplicated guarded loop copies from
	// consuming a fast-branch preamble.
	affineArrayRowHoists map[affineArrayCallSiteKey][]*affineArrayRowHoist
}

// localClassInfo records how a local class was hoisted to file scope.
type localClassInfo struct {
	structName                 string
	captured                   []capturedLocal
	scope                      *symbol.ClassScope
	fieldInitializerMethodName string
}

// addHoistedDecl appends a synthesized top-level declaration to be emitted at
// file scope. No-op when the collector is not initialized.
func (ctx Ctx) addHoistedDecl(decl ast.Decl) {
	if ctx.hoistedDecls != nil && decl != nil {
		*ctx.hoistedDecls = append(*ctx.hoistedDecls, decl)
	}
}

// nextAnonClassIndex returns the next unique index for a synthesized
// anonymous/local class in the current file.
func (ctx Ctx) nextAnonClassIndex() int {
	if ctx.anonClassCounter == nil {
		return 0
	}
	*ctx.anonClassCounter++
	return *ctx.anonClassCounter
}

func (ctx Ctx) nextControlLabelIndex() int {
	if ctx.controlLabelCounter == nil {
		return 0
	}
	*ctx.controlLabelCounter++
	return *ctx.controlLabelCounter
}

// tryReturnTarget carries Java abrupt completion across generated function
// literals. Try/finally and synchronized lowering share it so nested closure
// boundaries can propagate returns and loop transfers one level at a time.
type tryReturnTarget struct {
	FlagName    string
	ValueName   string
	HasValue    bool
	ControlName string
	controls    map[tryControlTransferKey]*tryControlTransfer
	controlList []*tryControlTransfer
}

type javaControlTargetKey struct {
	TargetType  string
	TargetStart uint32
	TargetEnd   uint32
}

type doWhileContinueTarget struct {
	Label string
	Used  bool
}

type javaLabelTarget struct {
	NeedsGoLabel bool
	// BreakLabel is set for a Java label whose statement is not directly
	// breakable in Go. Resolved breaks to that exact Java target become gotos to
	// this synthetic end label, avoiding ambiguous comparisons between sanitized
	// source-label spellings.
	BreakLabel string
}

func javaControlKey(node *sitter.Node) (javaControlTargetKey, bool) {
	if node == nil {
		return javaControlTargetKey{}, false
	}
	return javaControlTargetKey{
		TargetType:  node.Type(),
		TargetStart: node.StartByte(),
		TargetEnd:   node.EndByte(),
	}, true
}

// javaSourceLabelName assigns every Java labeled statement a deterministic Go
// label based on the resolved source target rather than its spelling. Distinct
// Java identifiers can sanitize to the same Go identifier (`map` and `map_`),
// but their source spans cannot be identical.
func javaSourceLabelName(target *sitter.Node) string {
	if target == nil {
		return ""
	}
	return fmt.Sprintf("__java2goLabel_%d_%d", target.StartByte(), target.EndByte())
}

type tryControlTransferKey struct {
	Tok         token.Token
	Label       string
	TargetType  string
	TargetStart uint32
	TargetEnd   uint32
}

type tryControlTransfer struct {
	tryControlTransferKey
	Code       int
	JavaTarget *sitter.Node
}

func (target *tryReturnTarget) registerControlTransfer(tok token.Token, label string, javaTarget *sitter.Node) *tryControlTransfer {
	if target == nil || javaTarget == nil {
		return nil
	}
	if target.controls == nil {
		target.controls = make(map[tryControlTransferKey]*tryControlTransfer)
	}
	key := tryControlTransferKey{
		Tok:         tok,
		Label:       label,
		TargetType:  javaTarget.Type(),
		TargetStart: javaTarget.StartByte(),
		TargetEnd:   javaTarget.EndByte(),
	}
	if existing := target.controls[key]; existing != nil {
		return existing
	}
	transfer := &tryControlTransfer{
		tryControlTransferKey: key,
		Code:                  len(target.controlList) + 1,
		JavaTarget:            javaTarget,
	}
	target.controls[key] = transfer
	target.controlList = append(target.controlList, transfer)
	return transfer
}

func javaBranchTarget(node *sitter.Node, source []byte, tok token.Token) (*sitter.Node, string) {
	if node == nil {
		return nil, ""
	}

	rawLabel := ""
	fallbackLabel := ""
	if node.NamedChildCount() > 0 {
		rawLabel = node.NamedChild(0).Content(source)
		fallbackLabel = sanitizeGoIdent(rawLabel)
	}

	for ancestor := node.Parent(); ancestor != nil; ancestor = ancestor.Parent() {
		if rawLabel != "" {
			if ancestor.Type() != "labeled_statement" || ancestor.NamedChildCount() == 0 {
				continue
			}
			if ancestor.NamedChild(0).Content(source) == rawLabel {
				return ancestor, javaSourceLabelName(ancestor)
			}
			continue
		}

		switch ancestor.Type() {
		case "for_statement", "enhanced_for_statement", "while_statement", "do_statement":
			return ancestor, ""
		case "switch_statement", "switch_expression":
			if tok == token.BREAK {
				return ancestor, ""
			}
		}
	}
	return nil, fallbackLabel
}

func javaTargetInsideBoundary(target *sitter.Node, boundary *sitter.Node) bool {
	if target == nil || boundary == nil {
		return false
	}
	return target.StartByte() >= boundary.StartByte() && target.EndByte() <= boundary.EndByte()
}

func transferTargetInsideBoundary(transfer *tryControlTransfer, boundary *sitter.Node) bool {
	if transfer == nil || boundary == nil {
		return false
	}
	return transfer.TargetStart >= boundary.StartByte() && transfer.TargetEnd <= boundary.EndByte()
}

// Clone performs a shallow copy on a `Ctx`, returning a new Ctx with its pointers
// pointing at the same things as the previous Ctx
func (c Ctx) Clone() Ctx {
	return Ctx{
		className:                           c.className,
		currentFile:                         c.currentFile,
		currentClass:                        c.currentClass,
		localScope:                          c.localScope,
		lastType:                            c.lastType,
		expectedType:                        c.expectedType,
		expectedTypeRoot:                    c.expectedTypeRoot,
		syntheticTypeParameters:             c.syntheticTypeParameters,
		rawGenericParameterTypes:            c.rawGenericParameterTypes,
		importAliases:                       c.importAliases,
		usedImports:                         c.usedImports,
		tryReturnTarget:                     c.tryReturnTarget,
		tryControlBoundary:                  c.tryControlBoundary,
		doWhileContinueTargets:              c.doWhileContinueTargets,
		javaLabelTargets:                    c.javaLabelTargets,
		suppressUnsupportedDiagnostics:      c.suppressUnsupportedDiagnostics,
		disableAffineArrayRowSpecialization: c.disableAffineArrayRowSpecialization,
		hoistedDecls:                        c.hoistedDecls,
		anonClassCounter:                    c.anonClassCounter,
		controlLabelCounter:                 c.controlLabelCounter,
		localClasses:                        c.localClasses,
		affineArrayBindings:                 c.affineArrayBindings,
		affineArrayCallSites:                c.affineArrayCallSites,
		affineArrayNonNullBindings:          c.affineArrayNonNullBindings,
		affineArrayRowCallSites:             c.affineArrayRowCallSites,
		affineArrayRowHoists:                c.affineArrayRowHoists,
	}
}

// ParseNode parses a given tree-sitter node and returns the ast representation
//
// This function is called when the node being parsed might not be a direct
// expression or statement, as those are parsed with `ParseExpr` and `ParseStmt`
// respectively
func ParseNode(node *sitter.Node, source []byte, ctx Ctx) interface{} {
	switch node.Type() {
	case "ERROR":
		log.WithFields(log.Fields{
			"parsed":    node.Content(source),
			"className": ctx.className,
		}).Warn("Error parsing generic node")
		return &ast.BadStmt{}
	case "program":
		// A program contains all the source code, in this case, one `class_declaration`
		if ctx.importAliases == nil {
			ctx.importAliases = make(map[string]string)
		}
		if ctx.usedImports == nil {
			ctx.usedImports = make(map[string]bool)
		}
		if ctx.hoistedDecls == nil {
			ctx.hoistedDecls = &[]ast.Decl{}
		}
		if ctx.anonClassCounter == nil {
			ctx.anonClassCounter = new(int)
		}
		if ctx.controlLabelCounter == nil {
			ctx.controlLabelCounter = new(int)
		}
		if ctx.localClasses == nil {
			ctx.localClasses = make(map[string]*localClassInfo)
		}

		program := &ast.File{
			Name: &ast.Ident{Name: "main"},
		}

		for _, c := range nodeutil.NamedChildrenOf(node) {
			switch c.Type() {
			case "package_declaration":
				pkg := c.NamedChild(0)
				if pkg != nil && pkg.NamedChildCount() > 0 {
					program.Name = &ast.Ident{Name: pkg.NamedChild(int(pkg.NamedChildCount() - 1)).Content(source)}
				}
			case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
				declCtx := ctx.Clone()
				if nameNode := c.ChildByFieldName("name"); nameNode != nil && ctx.currentFile != nil {
					if scope := ctx.currentFile.FindClassScope(nameNode.Content(source)); scope != nil {
						declCtx.currentClass = scope
					}
				}
				program.Decls = append(program.Decls, ParseDecls(c, source, declCtx)...)
			}
		}

		// Emit any classes synthesized during parsing (anonymous non-SAM and local
		// classes) at file scope.
		program.Decls = append(program.Decls, *ctx.hoistedDecls...)

		program.Imports = buildUsedImportSpecs(ctx)
		if len(program.Imports) > 0 {
			importSpecs := make([]ast.Spec, len(program.Imports))
			for ind, spec := range program.Imports {
				importSpecs[ind] = spec
			}
			program.Decls = append([]ast.Decl{
				&ast.GenDecl{
					Tok:   token.IMPORT,
					Specs: importSpecs,
				},
			}, program.Decls...)
		}
		return program
	case "field_declaration":
		var public bool

		if node.NamedChild(0).Type() == "modifiers" {
			for _, modifier := range nodeutil.UnnamedChildrenOf(node.NamedChild(0)) {
				if modifier.Type() == "public" {
					public = true
				}
			}
		}

		fieldType := javaTypeStringToGoTypeExpr(node.ChildByFieldName("type").Content(source), inScopeTypeParameters(ctx), ctx)
		fieldName := identFromNode(node.ChildByFieldName("declarator").ChildByFieldName("name"), source)
		fieldName.Name = symbol.HandleExportStatus(public, fieldName.Name)

		// If the field is assigned to a value (ex: int field = 1)
		fieldAssignmentNode := node.ChildByFieldName("declarator").ChildByFieldName("value")
		if fieldAssignmentNode != nil {
			return &ast.ValueSpec{
				Names: []*ast.Ident{fieldName},
				Type:  fieldType,
				Values: []ast.Expr{
					ParseExpr(fieldAssignmentNode, source, ctx),
				},
			}
		}

		return &ast.Field{
			Names: []*ast.Ident{fieldName},
			Type:  fieldType,
		}
	case "import_declaration":
		return parseImportDeclaration(node, source)
	case "method_declaration":
		comments := []*ast.Comment{}

		if node.NamedChild(0).Type() == "modifiers" {
			for _, modifier := range nodeutil.UnnamedChildrenOf(node.NamedChild(0)) {
				switch modifier.Type() {
				case "marker_annotation", "annotation":
					comments = append(comments, &ast.Comment{Text: "//" + modifier.Content(source)})
					if _, in := excludedAnnotations[modifier.Content(source)]; in {
						// If this entire method is ignored, we return an empty field, which
						// is handled by the logic that parses a class file
						return &ast.Field{}
					}
				}
			}
		}

		methodName := node.ChildByFieldName("name").Content(source)
		methodParameters := node.ChildByFieldName("parameters")

		comparison := func(d *symbol.Definition) bool {
			// The names must match
			if methodName != d.OriginalName {
				return false
			}

			// Size of parameters must match
			if int(methodParameters.NamedChildCount()) != len(d.Parameters) {
				return false
			}

			// Go through the types and check to see if they differ
			for index, param := range nodeutil.NamedChildrenOf(methodParameters) {
				var paramType string
				if param.Type() == "spread_parameter" {
					paramType = param.NamedChild(0).Content(source)
				} else {
					paramType = param.ChildByFieldName("type").Content(source)
				}
				if paramType != d.Parameters[index].OriginalType {
					return false
				}
			}

			return true
		}

		def := ctx.currentClass.FindMethod().By(comparison)[0]
		ctx.localScope = def

		parameters := &ast.FieldList{}
		for _, param := range nodeutil.NamedChildrenOf(methodParameters) {
			parameters.List = append(parameters.List, ParseNode(param, source, ctx).(*ast.Field))
		}

		var results *ast.FieldList
		if strings.TrimSpace(def.OriginalType) != "" && strings.TrimSpace(def.OriginalType) != "void" {
			results = &ast.FieldList{List: []*ast.Field{
				{
					Type: javaTypeStringToGoTypeExpr(def.OriginalType, inScopeTypeParameters(ctx), ctx),
				},
			}}
		}

		return &ast.Field{
			Doc:   &ast.CommentGroup{List: comments},
			Names: []*ast.Ident{&ast.Ident{Name: def.Name}},
			Type: &ast.FuncType{
				Params:  parameters,
				Results: results,
			},
		}
	case "try_with_resources_statement":
		return lowerTryStatement(node, source, ctx, true)
	case "try_statement":
		return lowerTryStatement(node, source, ctx, false)
	case "synchronized_statement":
		return lowerSynchronizedStatement(node, source, ctx)
	case "switch_label":
		if node.NamedChildCount() > 0 {
			return &ast.CaseClause{
				List: []ast.Expr{ParseExpr(node.NamedChild(0), source, ctx)},
			}
		}
		return &ast.CaseClause{}
	case "argument_list":
		args := []ast.Expr{}
		for _, c := range nodeutil.NamedChildrenOf(node) {
			args = append(args, ParseExpr(c, source, ctx))
		}
		return args

	case "formal_parameters":
		params := &ast.FieldList{}
		for _, param := range nodeutil.NamedChildrenOf(node) {
			params.List = append(params.List, ParseNode(param, source, ctx).(*ast.Field))
		}
		return params
	case "formal_parameter":
		if ctx.localScope != nil {
			paramDef := ctx.localScope.ParameterByName(node.ChildByFieldName("name").Content(source))
			if paramDef == nil {
				paramDef = &symbol.Definition{
					Name:         node.ChildByFieldName("name").Content(source),
					OriginalType: node.ChildByFieldName("type").Content(source),
				}
			}

			paramType := paramDef.OriginalType
			if strings.TrimSpace(paramType) == "" {
				paramType = node.ChildByFieldName("type").Content(source)
			}
			if rewritten, ok := ctx.rawGenericParameterTypes[paramDef.OriginalName]; ok {
				paramType = rewritten
			} else if rewritten, ok := ctx.rawGenericParameterTypes[paramDef.Name]; ok {
				paramType = rewritten
			}

			return &ast.Field{
				Names: []*ast.Ident{&ast.Ident{Name: paramDef.Name}},
				Type:  abstractClassToInterface(javaTypeStringToGoTypeExpr(paramType, inScopeTypeParameters(ctx), ctx), paramType, ctx),
			}
		}
		fallbackType := node.ChildByFieldName("type").Content(source)
		return &ast.Field{
			Names: []*ast.Ident{identFromNode(node.ChildByFieldName("name"), source)},
			Type:  abstractClassToInterface(javaTypeStringToGoTypeExpr(fallbackType, inScopeTypeParameters(ctx), ctx), fallbackType, ctx),
		}
	case "spread_parameter":
		// The spread paramater takes a list and separates it into multiple elements
		// Ex: addElements([]int elements...)

		spreadType := node.NamedChild(0)
		spreadDeclarator := node.NamedChild(1)

		return &ast.Field{
			Names: []*ast.Ident{identFromNode(spreadDeclarator.ChildByFieldName("name"), source)},
			Type: &ast.Ellipsis{
				Elt: javaTypeStringToGoTypeExpr(spreadType.Content(source), inScopeTypeParameters(ctx), ctx),
			},
		}
	case "inferred_parameters":
		params := &ast.FieldList{}
		for _, param := range nodeutil.NamedChildrenOf(node) {
			params.List = append(params.List, &ast.Field{
				Names: []*ast.Ident{identFromNode(param, source)},
				// When we're not sure what parameters to infer, set them as interface
				// values to avoid a panic
				Type: &ast.Ident{Name: "interface{}"},
			})
		}
		return params
	case "comment", "line_comment", "block_comment": // Ignore comments
		return nil
	}

	reportUnsupported("node", node, source, ctx)
	// Mirror the ERROR case: return a bad statement so a single unknown node does
	// not abort the whole conversion.
	return &ast.BadStmt{}
}

func lowerTryStatement(node *sitter.Node, source []byte, ctx Ctx, withResources bool) []ast.Stmt {
	if node == nil {
		return nil
	}

	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		for _, child := range nodeutil.NamedChildrenOf(node) {
			if child.Type() == "block" {
				bodyNode = child
				break
			}
		}
	}
	if bodyNode == nil {
		return nil
	}

	var resourceDecls []resourceDecl
	if withResources {
		resourceDecls = parseResourceDecls(node.ChildByFieldName("resources"), source, ctx)
	}

	catchClauses := []*sitter.Node{}
	var finallyClause *sitter.Node
	for _, child := range nodeutil.NamedChildrenOf(node) {
		switch child.Type() {
		case "catch_clause":
			catchClauses = append(catchClauses, child)
		case "finally_clause":
			finallyClause = child
		}
	}

	suffix := fmt.Sprintf("_%d", node.StartByte())
	recoveredName := "__java2goRecovered" + suffix
	didPanicName := "__java2goDidPanic" + suffix
	handledName := "__java2goCatchHandled" + suffix
	shouldReturnName := "__java2goShouldReturn" + suffix
	returnValueName := "__java2goReturnValue" + suffix
	controlName := "__java2goControl" + suffix
	hasReturnValue := false
	var returnValueType ast.Expr
	if ctx.localScope != nil {
		returnType := strings.TrimSpace(ctx.localScope.OriginalType)
		if returnType != "" && returnType != "void" {
			hasReturnValue = true
			returnValueType = javaTypeStringToGoTypeExpr(returnType, inScopeTypeParameters(ctx), ctx)
		}
	}
	returnTarget := &tryReturnTarget{
		FlagName:    shouldReturnName,
		ValueName:   returnValueName,
		HasValue:    hasReturnValue,
		ControlName: controlName,
	}

	varSpecs := []ast.Spec{
		&ast.ValueSpec{
			Names: []*ast.Ident{{Name: recoveredName}},
			Type:  &ast.InterfaceType{Methods: &ast.FieldList{}},
		},
		&ast.ValueSpec{
			Names: []*ast.Ident{{Name: didPanicName}},
			Type:  &ast.Ident{Name: "bool"},
		},
		&ast.ValueSpec{
			Names: []*ast.Ident{{Name: handledName}},
			Type:  &ast.Ident{Name: "bool"},
		},
		&ast.ValueSpec{
			Names: []*ast.Ident{{Name: shouldReturnName}},
			Type:  &ast.Ident{Name: "bool"},
		},
	}
	if hasReturnValue {
		varSpecs = append(varSpecs, &ast.ValueSpec{
			Names: []*ast.Ident{{Name: returnValueName}},
			Type:  returnValueType,
		})
	}

	varDecl := &ast.GenDecl{Tok: token.VAR, Specs: varSpecs}
	stmts := []ast.Stmt{
		&ast.DeclStmt{
			Decl: varDecl,
		},
	}

	tryBodyStmts := []ast.Stmt{}
	for _, decl := range resourceDecls {
		resourceCtx := ctx.Clone()
		resourceCtx.tryReturnTarget = returnTarget
		resourceCtx.tryControlBoundary = bodyNode
		tryBodyStmts = append(tryBodyStmts, ParseStmt(decl.node, source, resourceCtx))
		tryBodyStmts = append(tryBodyStmts, buildResourceCloseDeferStmt(decl.name, ctx))
	}
	tryCtx := ctx.Clone()
	tryCtx.tryReturnTarget = returnTarget
	tryCtx.tryControlBoundary = bodyNode
	tryBodyStmts = append(tryBodyStmts, ParseStmt(bodyNode, source, tryCtx).(*ast.BlockStmt).List...)

	recoverStmt := buildTryRecoverStmt(recoveredName, didPanicName, "", "r", returnTarget, ctx)

	tryBody := &ast.BlockStmt{
		List: append(
			[]ast.Stmt{
				&ast.DeferStmt{
					Call: &ast.CallExpr{
						Fun: &ast.FuncLit{
							Type: &ast.FuncType{},
							Body: &ast.BlockStmt{List: []ast.Stmt{recoverStmt}},
						},
					},
				},
			},
			tryBodyStmts...,
		),
	}

	stmts = append(stmts, &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.FuncLit{
				Type: &ast.FuncType{},
				Body: tryBody,
			},
		},
	})

	catchDispatch := buildCatchDispatchStmt(catchClauses, recoveredName, didPanicName, handledName, returnTarget, source, ctx)

	var parsedFinally *ast.BlockStmt
	if finallyClause != nil {
		finallyBody := finallyClause.ChildByFieldName("body")
		if finallyBody == nil && finallyClause.NamedChildCount() > 0 {
			finallyBody = finallyClause.NamedChild(0)
		}
		if finallyBody != nil {
			finallyCtx := ctx.Clone()
			finallyCtx.tryReturnTarget = returnTarget
			finallyCtx.tryControlBoundary = finallyBody
			parsedFinally = ParseStmt(finallyBody, source, finallyCtx).(*ast.BlockStmt)
		}
	}
	if len(returnTarget.controlList) > 0 {
		varDecl.Specs = append(varDecl.Specs, &ast.ValueSpec{
			Names: []*ast.Ident{{Name: controlName}},
			Type:  &ast.Ident{Name: "int"},
		})
	}

	if parsedFinally != nil {
		// Catch bodies are another possible abrupt-completion source. Capture a
		// panic from a catch as pending state before running finally, instead of
		// running finally while Go is actively unwinding. This is what lets a
		// return/break/continue from finally supersede a catch throw, while a
		// normally completing finally still rethrows it below.
		if catchDispatch != nil {
			catchRecoverStmt := buildTryRecoverStmt(recoveredName, didPanicName, handledName, "catchPanic", returnTarget, ctx)
			stmts = append(stmts, &ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.FuncLit{
				Type: &ast.FuncType{},
				Body: &ast.BlockStmt{List: []ast.Stmt{
					&ast.DeferStmt{Call: &ast.CallExpr{Fun: &ast.FuncLit{
						Type: &ast.FuncType{},
						Body: &ast.BlockStmt{List: []ast.Stmt{catchRecoverStmt}},
					}}},
					catchDispatch,
				}},
			}}})
		}
		stmts = append(stmts, &ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.FuncLit{
			Type: &ast.FuncType{},
			Body: parsedFinally,
		}}})
	} else if catchDispatch != nil {
		stmts = append(stmts, catchDispatch)
	}

	// When this try statement is itself nested inside another try's body, the
	// lowered code runs inside a `func() {}` body closure that returns nothing.
	// A method-level `return` therefore cannot happen here; instead, propagate
	// the pending return to the enclosing try's return target and return from the
	// body closure with a bare return. The enclosing machinery performs the real
	// method return once its own body closure unwinds.
	if enclosing := ctx.tryReturnTarget; enclosing != nil {
		propagation := []ast.Stmt{}
		if hasReturnValue && enclosing.HasValue {
			propagation = append(propagation, &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: enclosing.ValueName}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{&ast.Ident{Name: returnValueName}},
			})
		}
		if len(enclosing.controlList) > 0 {
			propagation = append(propagation, &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: enclosing.ControlName}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "0"}},
			})
		}
		propagation = append(propagation, &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: enclosing.FlagName}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.Ident{Name: "true"}},
		})
		propagation = append(propagation, &ast.ReturnStmt{})
		stmts = append(stmts, &ast.IfStmt{
			Cond: &ast.Ident{Name: shouldReturnName},
			Body: &ast.BlockStmt{List: propagation},
		})
	} else {
		stmts = append(stmts, &ast.IfStmt{
			Cond: &ast.Ident{Name: shouldReturnName},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					replayedMethodReturnStmt(ctx, hasReturnValue, returnValueName),
				},
			},
		})
	}

	// Replay loop control only after catch/finally completion. An enclosing try
	// introduces another func-literal boundary, so a transfer that also crosses
	// that boundary is propagated one level at a time. If its Java target lives
	// inside the enclosing closure (for example an inner try inside a loop in an
	// outer try), it is safe to emit the real branch at this level.
	for _, transfer := range returnTarget.controlList {
		if transfer == nil {
			continue
		}
		branch := lowerJavaControlTransferStmt(transfer.Tok, transfer.Label, transfer.JavaTarget, ctx)
		body := []ast.Stmt{branch}
		if enclosing := ctx.tryReturnTarget; enclosing != nil && !transferTargetInsideBoundary(transfer, ctx.tryControlBoundary) {
			enclosingTransfer := enclosing.registerControlTransfer(transfer.Tok, transfer.Label, transfer.JavaTarget)
			if enclosingTransfer != nil {
				body = []ast.Stmt{
					&ast.AssignStmt{
						Lhs: []ast.Expr{&ast.Ident{Name: enclosing.FlagName}},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{&ast.Ident{Name: "false"}},
					},
					&ast.AssignStmt{
						Lhs: []ast.Expr{&ast.Ident{Name: enclosing.ControlName}},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", enclosingTransfer.Code)}},
					},
					&ast.ReturnStmt{},
				}
			}
		}
		stmts = append(stmts, &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  &ast.Ident{Name: controlName},
				Op: token.EQL,
				Y:  &ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", transfer.Code)},
			},
			Body: &ast.BlockStmt{List: body},
		})
	}

	stmts = append(stmts, &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  &ast.Ident{Name: didPanicName},
			Op: token.LAND,
			Y:  &ast.UnaryExpr{Op: token.NOT, X: &ast.Ident{Name: handledName}},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ExprStmt{
					X: &ast.CallExpr{
						Fun:  &ast.Ident{Name: "panic"},
						Args: []ast.Expr{&ast.Ident{Name: recoveredName}},
					},
				},
			},
		},
	})

	return stmts
}

func buildTryRecoverStmt(recoveredName, didPanicName, handledName, recoverVar string, returnTarget *tryReturnTarget, ctx Ctx) ast.Stmt {
	body := []ast.Stmt{
		// Normalize native Go runtime panics (divide by zero, nil
		// dereference, index out of range, failed type assertion) into the
		// matching Java exception so catch dispatch can handle them.
		&ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: recoveredName}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.CallExpr{
				Fun:  stdjavaQualifiedExpr("NormalizePanic", ctx),
				Args: []ast.Expr{&ast.Ident{Name: recoverVar}},
			}},
		},
		&ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: didPanicName}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.Ident{Name: "true"}},
		},
	}
	// A panic raised while deferred resource cleanup is unwinding supersedes a
	// pending return/break/continue from the try or catch body. Ordinary source
	// statements cannot reach this state, but try-with-resources close calls can.
	if returnTarget != nil {
		body = append(body, &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: returnTarget.FlagName}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.Ident{Name: "false"}},
		})
		if len(returnTarget.controlList) > 0 {
			body = append(body, &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: returnTarget.ControlName}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "0"}},
			})
		}
	}
	if handledName != "" {
		body = append(body, &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: handledName}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.Ident{Name: "false"}},
		})
	}
	return &ast.IfStmt{
		Init: &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: recoverVar}},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.CallExpr{Fun: &ast.Ident{Name: "recover"}},
			},
		},
		Cond: &ast.BinaryExpr{
			X:  &ast.Ident{Name: recoverVar},
			Op: token.NEQ,
			Y:  &ast.Ident{Name: "nil"},
		},
		Body: &ast.BlockStmt{List: body},
	}
}

func buildCatchDispatchStmt(catches []*sitter.Node, recoveredName, didPanicName, handledName string, returnTarget *tryReturnTarget, source []byte, ctx Ctx) ast.Stmt {
	if len(catches) == 0 {
		return nil
	}

	var dispatch ast.Stmt
	for index := len(catches) - 1; index >= 0; index-- {
		catchNode := catches[index]
		catchName, catchTypes := parseCatchParameter(catchNode, source)
		catchCond := catchConditionExpr(catchTypes, recoveredName, ctx)

		catchCtx := ctx.Clone()
		catchCtx.localScope = cloneLocalScopeDefinition(ctx.localScope)
		catchCtx.tryReturnTarget = returnTarget
		catchBlock := catchNode.ChildByFieldName("body")
		if catchBlock == nil {
			continue
		}
		catchCtx.tryControlBoundary = catchBlock

		bodyStmts := []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: handledName}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{&ast.Ident{Name: "true"}},
			},
		}

		if catchName != "" {
			catchType := "any"
			originalType := "Exception"
			if len(catchTypes) > 0 {
				originalType = catchTypes[0]
			}
			recordLocalVariableDefinition(catchCtx, catchName, originalType, catchType)
			bodyStmts = append(bodyStmts, &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: catchName}},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{&ast.Ident{Name: recoveredName}},
			})
			bodyStmts = append(bodyStmts, &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{&ast.Ident{Name: catchName}},
			})
		}

		parsedCatchBody := ParseStmt(catchBlock, source, catchCtx).(*ast.BlockStmt)
		bodyStmts = append(bodyStmts, &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.FuncLit{
					Type: &ast.FuncType{},
					Body: parsedCatchBody,
				},
			},
		})

		cond := &ast.BinaryExpr{
			X:  &ast.Ident{Name: didPanicName},
			Op: token.LAND,
			Y:  catchCond,
		}
		if dispatch == nil {
			dispatch = &ast.IfStmt{
				Cond: cond,
				Body: &ast.BlockStmt{List: bodyStmts},
			}
		} else {
			dispatch = &ast.IfStmt{
				Cond: cond,
				Body: &ast.BlockStmt{List: bodyStmts},
				Else: dispatch,
			}
		}
	}

	return dispatch
}

func parseCatchParameter(catchNode *sitter.Node, source []byte) (string, []string) {
	if catchNode == nil {
		return "", nil
	}

	var paramNode *sitter.Node
	for _, child := range nodeutil.NamedChildrenOf(catchNode) {
		if child.Type() == "catch_formal_parameter" {
			paramNode = child
			break
		}
	}
	if paramNode == nil {
		return "", nil
	}

	nameNode := paramNode.ChildByFieldName("name")
	catchName := ""
	if nameNode != nil {
		catchName = nameNode.Content(source)
	}

	catchTypes := []string{}
	typeNode := paramNode.NamedChild(0)
	if typeNode != nil {
		if typeNode.Type() == "catch_type" {
			for _, child := range nodeutil.NamedChildrenOf(typeNode) {
				catchTypes = append(catchTypes, child.Content(source))
			}
		} else {
			catchTypes = append(catchTypes, typeNode.Content(source))
		}
	}

	return catchName, catchTypes
}

func catchConditionExpr(catchTypes []string, recoveredName string, ctx Ctx) ast.Expr {
	if len(catchTypes) == 0 {
		return &ast.Ident{Name: "true"}
	}

	checks := []ast.Expr{}
	for _, catchType := range catchTypes {
		if shouldTreatAsCatchAll(catchType, ctx) {
			return &ast.Ident{Name: "true"}
		}

		// Match by hierarchy through the stdjava runtime: a thrown
		// IllegalArgumentException is caught by `catch (RuntimeException e)`.
		// Multi-catch (catch (A | B e)) is handled by OR-ing each type's check.
		base, _ := parseJavaTypeString(catchType)
		typeName := stripJavaQualifier(base)
		checks = append(checks, &ast.CallExpr{
			Fun: stdjavaQualifiedExpr("CaughtAs", ctx),
			Args: []ast.Expr{
				&ast.Ident{Name: recoveredName},
				&ast.BasicLit{Kind: token.STRING, Value: `"` + typeName + `"`},
			},
		})
	}

	if len(checks) == 0 {
		return &ast.Ident{Name: "true"}
	}

	cond := checks[0]
	for i := 1; i < len(checks); i++ {
		cond = &ast.BinaryExpr{
			X:  cond,
			Op: token.LOR,
			Y:  checks[i],
		}
	}
	return cond
}

func shouldTreatAsCatchAll(javaType string, ctx Ctx) bool {
	base, _ := parseJavaTypeString(javaType)
	base = stripJavaQualifier(base)

	// Throwable and Object are the only true catch-alls: they match anything that
	// escaped the try body. Everything else — including Exception and
	// RuntimeException — is matched by hierarchy through stdjava.CaughtAs. This is
	// sound because stdjava.NormalizePanic converts every raw Go panic into a
	// typed exception (RuntimeException by default) at the recover boundary, so a
	// thrown Error/Throwable is correctly NOT caught by catch (Exception e), while
	// more specific clauses still win by appearing earlier in the dispatch chain.
	switch base {
	case "Throwable", "Object":
		return true
	}

	// Built-in exception types are modelled by stdjava and matched by hierarchy.
	if isBuiltinExceptionType(base) {
		return false
	}

	if resolveClassScopeByQualifiedName(ctx, javaType) != nil {
		return false
	}
	if resolveClassScopeByQualifiedName(ctx, base) != nil {
		return false
	}

	// Unknown exception classes from external libraries are treated as catch-all,
	// so generated code does not depend on unavailable type declarations.
	return true
}

func cloneLocalScopeDefinition(local *symbol.Definition) *symbol.Definition {
	if local == nil {
		return nil
	}

	cloned := *local
	if len(local.Parameters) > 0 {
		cloned.Parameters = append([]*symbol.Definition{}, local.Parameters...)
	}
	if len(local.Children) > 0 {
		cloned.Children = append([]*symbol.Definition{}, local.Children...)
	}
	return &cloned
}

type resourceDecl struct {
	name string
	node *sitter.Node
}

func parseResourceDecls(resourcesNode *sitter.Node, source []byte, ctx Ctx) []resourceDecl {
	if resourcesNode == nil {
		return nil
	}

	decls := []resourceDecl{}
	for _, child := range nodeutil.NamedChildrenOf(resourcesNode) {
		if child == nil || child.Type() != "resource" {
			continue
		}

		resourceName := ""
		if nameNode := child.ChildByFieldName("name"); nameNode != nil {
			resourceName = nameNode.Content(source)
		}
		if strings.TrimSpace(resourceName) == "" {
			continue
		}

		decls = append(decls, resourceDecl{
			name: resourceName,
			node: child,
		})
	}

	return decls
}

func buildResourceCloseDeferStmt(resourceName string, ctx Ctx) ast.Stmt {
	return &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: stdjavaQualifiedExpr("CloseResource", ctx),
			Args: []ast.Expr{&ast.FuncLit{
				Type: &ast.FuncType{},
				Body: &ast.BlockStmt{List: []ast.Stmt{
					&ast.IfStmt{
						Cond: &ast.BinaryExpr{
							X:  &ast.Ident{Name: resourceName},
							Op: token.NEQ,
							Y:  &ast.Ident{Name: "nil"},
						},
						Body: &ast.BlockStmt{List: []ast.Stmt{
							&ast.ExprStmt{X: &ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X:   &ast.Ident{Name: resourceName},
									Sel: &ast.Ident{Name: "Close"},
								},
							}},
						}},
					},
				}},
			}},
		},
	}
}
