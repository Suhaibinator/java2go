package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Constructor functions keep their real Go binders and parameter ABI. This
// certificate concerns only declaration-owned locals and casts in their body.
// Its Enum view is the JVM erasure, not a check of an instantiated argument.
type genericFamilyConstructorBodyCertificate struct {
	owner        *symbol.ClassScope
	constructor  *symbol.Definition
	declarations map[*symbol.TypeParamDeclaration]bool
}

func certifyGenericFamilyEnumConstructorBody(plan *genericFamilyPlan, owner *symbol.ClassScope, constructor *symbol.Definition, declaration *symbol.TypeParamDeclaration, ctx Ctx) bool {
	if plan == nil || owner == nil || owner.Class == nil || owner.IsInner || constructor == nil || !constructor.Constructor || declaration == nil || len(constructor.TypeParameters) != 0 || constructor.DeclarationNode == nil {
		return false
	}
	declared := false
	for _, candidate := range owner.Methods {
		declared = declared || candidate == constructor
	}
	ownedBinder := false
	for _, parameter := range owner.OwnTypeParameters() {
		ownedBinder = ownedBinder || parameter.Declaration == declaration
	}
	representation, represented := plan.representations[declaration]
	if !declared || !ownedBinder || !represented || representation.erasure != "java.lang.Enum" || representation.goName != "JavaEnum" || representation.sourceBound != nil {
		return false
	}
	if owner.Class.DeclarationNode == nil || !sameSourceNode(constructor.DeclarationNode.Parent(), owner.Class.DeclarationNode.ChildByFieldName("body")) {
		return false
	}
	var bareDefinitions func(*symbol.Definition) bool
	bareDefinitions = func(definition *symbol.Definition) bool {
		if definition == nil {
			return true
		}
		if definitionReferencesTypeParameterDeclaration(definition, declaration) {
			component, rank := javaArrayTypeParts(strings.TrimSpace(definition.OriginalType))
			_, arguments := parseJavaTypeString(component)
			if definition.DirectTypeParameter != declaration || rank != 0 || len(arguments) != 0 {
				return false
			}
		}
		for _, child := range append(append([]*symbol.Definition(nil), definition.Parameters...), definition.Children...) {
			if !bareDefinitions(child) {
				return false
			}
		}
		return true
	}
	for _, child := range constructor.Children {
		if !bareDefinitions(child) {
			return false
		}
	}
	file := findFileScopeForClassScope(owner)
	if file == nil {
		return false
	}
	methodCtx := classScopeCtx(owner, ctx)
	methodCtx.localScope = constructor
	body := constructor.DeclarationNode.ChildByFieldName("body")
	if body == nil {
		return false
	}
	var supportedSyntax func(*sitter.Node) bool
	supportedSyntax = func(node *sitter.Node) bool {
		if node == nil {
			return true
		}
		switch node.Type() {
		case "array_type":
			component, rank := javaArrayTypeParts(node.Content(file.Source))
			if rank != 0 && visibleTypeParameterDeclarationForJavaType(component, methodCtx) == declaration {
				return false
			}
		case "ternary_expression", "lambda_expression", "explicit_constructor_invocation", "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration", "class_body":
			return false
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			if !supportedSyntax(child) {
				return false
			}
		}
		return true
	}
	if !supportedSyntax(body) || !methodBodyTypeSyntaxSupportsCallableErasure(owner, constructor, declaration, true, ctx) {
		return false
	}
	if plan.constructorBodies == nil {
		plan.constructorBodies = make(map[*symbol.Definition]*genericFamilyConstructorBodyCertificate)
	}
	certificate := plan.constructorBodies[constructor]
	if certificate == nil {
		certificate = &genericFamilyConstructorBodyCertificate{owner: owner, constructor: constructor, declarations: make(map[*symbol.TypeParamDeclaration]bool)}
		plan.constructorBodies[constructor] = certificate
	}
	if certificate.owner != owner || certificate.constructor != constructor {
		return false
	}
	certificate.declarations[declaration] = true
	return true
}

func genericFamilyConstructorCertificateMatches(certificate *genericFamilyConstructorBodyCertificate, plan *genericFamilyPlan, ctx Ctx) bool {
	return certificate != nil && plan != nil && ctx.currentClass == certificate.owner && ctx.localScope == certificate.constructor && ctx.localScope.Constructor && plan.constructorBodies[ctx.localScope] == certificate
}

func withGenericFamilyConstructorBody(ctx Ctx) Ctx {
	ctx.genericFamilyConstructorBody = nil
	if ctx.localScope == nil || !ctx.localScope.Constructor {
		return ctx
	}
	// Low-level render helpers deliberately leave their caller uncached. A
	// successful constructor body still needs one complete, body-local analysis
	// so a fresh query cannot exchange the declaration certificate underneath it.
	incomingAnalysis := ctx.genericFamilies
	if incomingAnalysis == nil {
		ctx.genericFamilies = &genericFamilyAnalysis{}
	}
	plan := canonicalGenericFamily(ctx.currentClass, ctx)
	if plan != nil {
		certificate := plan.constructorBodies[ctx.localScope]
		if genericFamilyConstructorCertificateMatches(certificate, plan, ctx) {
			ctx.genericFamilyConstructorBody = certificate
			return ctx
		}
	}
	ctx.genericFamilies = incomingAnalysis
	return ctx
}

func genericFamilyConstructorLocalErasure(javaType string, ctx Ctx) (string, bool) {
	certificate := ctx.genericFamilyConstructorBody
	if certificate == nil || ctx.currentClass != certificate.owner || ctx.localScope != certificate.constructor {
		return "", false
	}
	declaration := visibleTypeParameterDeclarationForJavaType(javaType, ctx)
	if declaration == nil || !certificate.declarations[declaration] {
		return "", false
	}
	plan := canonicalGenericFamily(ctx.currentClass, ctx)
	if !genericFamilyConstructorCertificateMatches(certificate, plan, ctx) {
		return "", false
	}
	representation, known := plan.representations[declaration]
	return representation.erasure, known
}

// Java var may infer the constructor's class declaration even though its
// written syntax contains no binder. Preserve that declared scalar identity;
// a same-spelled method binder, nominal type, array or incomplete origin cannot
// donate the constructor's erased local representation.
func genericFamilyConstructorLocalType(written, inferred string, origin inferredJavaTypeOrigin, ctx Ctx) (string, bool) {
	if _, known := genericFamilyConstructorLocalErasure(written, ctx); known {
		return written, true
	}
	if !isVarKeywordType(strings.TrimSpace(written)) || inferred == "" || !completeJavaTypeOrigin(origin) || origin.parameter == nil || origin.arrayRank != 0 || origin.wildcard || len(origin.arguments) != 0 || origin.nominalScope != nil || origin.externalName != "" {
		return "", false
	}
	if visibleTypeParameterDeclarationForJavaType(inferred, ctx) != origin.parameter {
		return "", false
	}
	if _, known := genericFamilyConstructorLocalErasure(inferred, ctx); !known {
		return "", false
	}
	return inferred, true
}
