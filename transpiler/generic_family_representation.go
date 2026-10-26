package transpiler

import (
	"fmt"
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
)

// A family keeps Java binder identity and bounds separate from its physical
// reference representation. Different declarations in one inheritance chain
// can have different erasures: Base<T> uses Object while Child<N extends
// Number> uses Number, with a checked override bridge between them.
type genericFamilyBinderRepresentation struct {
	erasure           string
	goName            string
	sourceBound       *symbol.ClassScope
	sourceFile        *symbol.FileScope
	sourceCertificate *genericFamilySourceBoundCertificate
}

func genericFamilyBinderErasure(parameter symbol.TypeParam, ctx Ctx, visiting map[typeParameterIdentityKey]bool) (string, error) {
	key := identityKeyForTypeParameter(parameter)
	if parameter.Declaration == nil || visiting[key] {
		return "", fmt.Errorf("generic family has an unresolved or cyclic binder")
	}
	if len(parameter.Bounds) == 0 {
		return "java.lang.Object", nil
	}
	if len(parameter.Bounds) != 1 {
		if source, _, ok := genericFamilyConcreteSourceBound(parameter, ctx, map[*symbol.TypeParamDeclaration]bool{}); ok {
			return javaInferenceTypeName(source), nil
		}
		return "", fmt.Errorf("generic family intersection bound requires a representation proof")
	}
	visiting[key] = true
	defer delete(visiting, key)
	if source, _, ok := genericFamilyConcreteSourceBound(parameter, ctx, map[*symbol.TypeParamDeclaration]bool{}); ok {
		return javaInferenceTypeName(source), nil
	}
	bound := parameter.Bounds[0]
	if dependency, found := resolveReferenceTypeParameter(bound, ctx); found {
		return genericFamilyBinderErasure(dependency.parameter, dependency.context, visiting)
	}
	base, _ := parseJavaTypeString(qualifyDeclaredReferenceType(bound, ctx))
	if resolveClassScopeByQualifiedName(ctx, base) != nil {
		return "", fmt.Errorf("generic family source bound requires a representation proof: %s", base)
	}
	return base, nil
}

func (plan *genericFamilyPlan) addBinderRepresentations(scope *symbol.ClassScope, ctx Ctx) error {
	return plan.addBinderParameterRepresentations(scope.TypeParameters, classScopeCtx(scope, ctx))
}

// Carried binders keep the source namespace of their original declaration,
// even when their anonymous body inherits a different member TYPE namespace.
func (plan *genericFamilyPlan) addBinderParameterRepresentations(parameters []symbol.TypeParam, declarationCtx Ctx) error {
	for _, parameter := range parameters {
		parameterCtx := declarationCtx
		// Carried enclosing declarations retain their original lexical context.
		for _, binding := range referenceTypeParameterBindings(declarationCtx) {
			if binding.parameter.Declaration == parameter.Declaration {
				parameterCtx = binding.context
				break
			}
		}
		erasure, err := genericFamilyBinderErasure(parameter, parameterCtx, map[typeParameterIdentityKey]bool{})
		if err != nil {
			return err
		}
		representation := genericFamilyBinderRepresentation{erasure: erasure}
		if source, file, ok := genericFamilyConcreteSourceBound(parameter, parameterCtx, map[*symbol.TypeParamDeclaration]bool{}); ok {
			representation.sourceBound = source
			representation.sourceFile = file
		}
		switch erasure {
		case "java.lang.Object":
			representation.goName = "any"
		case "java.lang.Number":
			representation.goName = "JavaNumber"
		case "java.lang.Enum":
			representation.goName = "JavaEnum"
		case "java.util.Date":
			owner, known := canonicalIntrinsicOwner(erasure, parameterCtx)
			if !known || owner != erasure || !intrinsicOwnerSupported(owner) {
				return fmt.Errorf("generic family bound requires an unsupported physical representation: %s", erasure)
			}
			// The existing util.Date reference protocol retains SQL subtype identity.
			representation.goName = "DateValue"
		default:
			if representation.sourceBound == nil {
				return fmt.Errorf("generic family bound requires an unsupported physical representation: %s", erasure)
			}
		}
		if prior, exists := plan.representations[parameter.Declaration]; exists && prior != representation {
			return fmt.Errorf("generic family binder has inconsistent declaration erasure")
		}
		plan.representations[parameter.Declaration] = representation
		plan.binders[parameter.Declaration] = struct{}{}
	}
	return nil
}

func genericFamilyBinderGoType(parameter symbol.TypeParam, ctx Ctx) ast.Expr {
	plan := canonicalGenericFamily(ctx.currentClass, ctx)
	if plan == nil {
		return nil
	}
	representation, ok := plan.representations[parameter.Declaration]
	if !ok {
		return nil
	}
	if representation.sourceBound != nil {
		return genericFamilySourceBoundGoType(representation, ctx)
	}
	if representation.goName == "any" {
		return ast.NewIdent("any")
	}
	return stdjavaQualifiedExpr(representation.goName, ctx)
}
