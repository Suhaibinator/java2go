package transpiler

import (
	"fmt"
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"strings"
)

func genericFamilyAnonymousAuditScope(event genericFamilySourceEvent) (*symbol.ClassScope, []capturedLocal, error) {
	if event.node == nil || event.node.Type() != "object_creation_expression" || event.body == nil || event.file == nil || event.lexical.currentFile != event.file || event.owner == nil || event.parent == nil || event.parent.Class == nil {
		return nil, nil, fmt.Errorf("anonymous generic family requires its original source event")
	}
	root := event.node
	for root.Parent() != nil {
		root = root.Parent()
	}
	if event.file.ParsedRoot() == nil || !root.Equal(event.file.ParsedRoot()) {
		return nil, nil, fmt.Errorf("anonymous generic family source root is not current")
	}
	bodyKnown := false
	for _, child := range nodeutil.NamedChildrenOf(event.node) {
		bodyKnown = bodyKnown || child.Equal(event.body)
	}
	if !bodyKnown || event.body.Type() != "class_body" {
		return nil, nil, fmt.Errorf("anonymous generic family body is not owned")
	}
	if anonymousMemberTypeDeclaration(event.body) != nil {
		return nil, nil, fmt.Errorf("anonymous generic family contains an unsupported member type declaration")
	}
	// The creation header occurs outside the anonymous body. Its actual source
	// parent must be resolved before entering inherited member TYPE namespaces.
	base, _ := parseJavaTypeString(event.javaType)
	if resolveClassScopeByQualifiedName(event.lexical, base) != event.parent {
		return nil, nil, fmt.Errorf("anonymous generic family parent is not the original header declaration")
	}
	captures := collectCapturedLocalDefinitions(event.body, event.file.Source, event.lexical)
	for _, capture := range captures {
		if capture.javaDef == nil || strings.TrimSpace(capture.javaDef.OriginalType) == "" || isVarKeywordType(strings.TrimSpace(capture.javaDef.OriginalType)) {
			return nil, nil, fmt.Errorf("anonymous generic family capture needs an original declared logical type")
		}
	}

	inner := event.lexical.currentClass != nil && event.lexical.localScope != nil && !event.lexical.localScope.IsStatic
	scope := &symbol.ClassScope{Class: &symbol.Definition{DeclarationNode: event.node}, Enclosing: event.owner, IsInner: inner}
	scope.TypeParameters = anonymousClassTypeParameters(event.node, captures, inner, event.file.Source, event.lexical)
	if event.parent.IsInterface {
		scope.ImplementedInterfaces = []string{event.javaType}
	} else {
		scope.Superclass = event.javaType
	}

	// Methods deliberately remain absent: auditTypeSyntax reconstructs each
	// method's own binders from its original AST and preserves their shadowing.
	return scope, captures, nil
}
func genericFamilyCaptureTypeOccurrence(capture capturedLocal, event genericFamilySourceEvent) *sitter.Node {
	method := event.lexical.localScope
	if method == nil || method.DeclarationNode == nil || capture.javaDef == nil {
		return nil
	}
	var found *sitter.Node
	ambiguous := false
	selectType := func(typ, name *sitter.Node) {
		if typ != nil && name != nil && name.Content(event.file.Source) == capture.javaDef.OriginalName && typ.Content(event.file.Source) == capture.javaDef.OriginalType {
			if found != nil && !found.Equal(typ) {
				ambiguous = true
			}
			found = typ
		}
	}
	if parameters := method.DeclarationNode.ChildByFieldName("parameters"); parameters != nil {
		for _, parameter := range nodeutil.NamedChildrenOf(parameters) {
			typ, name := nodeutil.JavaParameterNodes(parameter)
			selectType(typ, name)
		}
	}
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "class_declaration" || node.Type() == "interface_declaration" || node.Type() == "object_creation_expression" {
			return
		}
		if node.Type() == "local_variable_declaration" {
			for _, declarator := range nodeutil.VariableDeclarators(node) {
				selectType(node.ChildByFieldName("type"), declarator.ChildByFieldName("name"))
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(method.DeclarationNode.ChildByFieldName("body"))
	if ambiguous {
		return nil
	}
	return found
}
func genericFamilyAnonymousEmissionMatches(frame genericFamilyAnonymous, scope *symbol.ClassScope, plan *genericFamilyPlan, ctx Ctx) bool {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil || frame.declarationCtx.currentFile == nil || frame.node == nil || !frame.node.Equal(scope.Class.DeclarationNode) || len(frame.carried) != len(scope.TypeParameters) {
		return false
	}
	currentCaptures := collectCapturedLocalDefinitions(objectCreationClassBody(frame.node), frame.declarationCtx.currentFile.Source, frame.declarationCtx)
	if !genericFamilyCaptureListMatches(frame.captures, currentCaptures, frame.declarationCtx) {
		return false
	}
	for _, proof := range frame.captures {
		var field *symbol.Definition
		for _, candidate := range scope.Fields {
			if candidate != nil && candidate.OriginalName == proof.name {
				if field != nil {
					return false
				}
				field = candidate
			}
		}
		if field == nil || !sameJavaTypeOrigin(proof.logical, declaredJavaTypeOrigin(definitionDeclaredJavaType(field), frame.declarationCtx)) {
			return false
		}
	}
	for _, info := range ctx.anonymousClasses {
		if info != nil && info.scope == scope && !genericFamilyCaptureListMatches(frame.captures, info.captured, frame.declarationCtx) {
			return false
		}
	}
	parentKnown := false
	if frame.parent.IsInterface {
		for _, typ := range scope.ImplementedInterfaces {
			parentKnown = parentKnown || typ == frame.javaType
		}
	} else {
		parentKnown = scope.Superclass == frame.javaType
	}
	if !parentKnown {
		return false
	}
	ownerKnown := false
	for enclosing := scope.Enclosing; enclosing != nil; enclosing = enclosing.Enclosing {
		ownerKnown = ownerKnown || enclosing == frame.owner
	}
	if !ownerKnown {
		return false
	}
	for index, parameter := range scope.TypeParameters {
		if parameter.Declaration == nil || parameter.Declaration != frame.carried[index].Declaration {
			return false
		}
	}
	check := &genericFamilyPlan{binders: map[*symbol.TypeParamDeclaration]struct{}{}, representations: map[*symbol.TypeParamDeclaration]genericFamilyBinderRepresentation{}}
	if err := check.addBinderParameterRepresentations(scope.TypeParameters, frame.declarationCtx); err != nil {
		return false
	}
	for declaration, representation := range check.representations {
		audited, present := plan.representations[declaration]
		if !present || audited.erasure != representation.erasure || audited.goName != representation.goName || audited.sourceBound != representation.sourceBound || audited.sourceFile != representation.sourceFile {
			return false
		}
	}
	return true
}

// Anonymous lowering currently models fields, captures and methods, not member
// type declarations. Refuse the unsupported namespace instead of resolving its
// members through an unrelated enclosing or top-level declaration.
func anonymousMemberTypeDeclaration(body *sitter.Node) *sitter.Node {
	for _, member := range nodeutil.NamedChildrenOf(body) {
		if sourceGenericViewTypeDeclaration(member) {
			return member
		}
	}
	return nil
}

// Store the source declaration identity and a resolved value tree, never only
// the mutable variable Definition or its mutable TypeParameterBindings map.
type genericFamilyCaptureSnapshot struct {
	origin  *symbol.Definition
	name    string
	logical inferredJavaTypeOrigin
}

func genericFamilyCaptureOrigin(capture capturedLocal) *symbol.Definition {
	if capture.origin != nil {
		return capture.origin
	}
	return capture.javaDef
}
func genericFamilyCaptureSnapshots(captures []capturedLocal, ctx Ctx) ([]genericFamilyCaptureSnapshot, error) {
	var result []genericFamilyCaptureSnapshot
	for _, capture := range captures {
		if capture.javaDef == nil {
			return nil, fmt.Errorf("anonymous capture has no source declaration")
		}
		logical := declaredJavaTypeOrigin(definitionDeclaredJavaType(capture.javaDef), ctx)
		if !completeJavaTypeOrigin(logical) {
			return nil, fmt.Errorf("anonymous capture has an unresolved declared type")
		}
		result = append(result, genericFamilyCaptureSnapshot{origin: genericFamilyCaptureOrigin(capture), name: capture.javaDef.OriginalName, logical: logical})
	}
	return result, nil
}
func genericFamilyCaptureListMatches(proofs []genericFamilyCaptureSnapshot, captures []capturedLocal, ctx Ctx) bool {
	if len(proofs) != len(captures) {
		return false
	}
	for index, capture := range captures {
		proof := proofs[index]
		if capture.javaDef == nil || genericFamilyCaptureOrigin(capture) != proof.origin || capture.javaDef.OriginalName != proof.name || !sameJavaTypeOrigin(proof.logical, declaredJavaTypeOrigin(definitionDeclaredJavaType(capture.javaDef), ctx)) {
			return false
		}
	}
	return true
}
