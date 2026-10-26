package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

// JDK21's primitive declarations form a specificity chain. In particular,
// (long,int) is distinct from (long,long): its result is int, not long.
type mathFloorModDeclaration struct {
	parameters [2]string
	result     string
	helper     string
}

var mathFloorModDeclarations = [...]mathFloorModDeclaration{
	{parameters: [2]string{"int", "int"}, result: "int", helper: "MathFloorModInt"},
	{parameters: [2]string{"long", "int"}, result: "int", helper: "MathFloorModLongInt"},
	{parameters: [2]string{"long", "long"}, result: "long", helper: "MathFloorModLong"},
}

func init() {
	for _, declaration := range mathFloorModDeclarations {
		registerStaticIntrinsicImportSignature("Math", "floorMod", declaration.parameters[:]...)
		definitions := staticIntrinsicImportSignatures[intrinsicKey{"Math", "floorMod"}]
		definitions[len(definitions)-1].OriginalType = declaration.result
	}
	registerStaticNodeIntrinsic("Math", "floorMod", func(node *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
		declaration, ok := mathFloorModInvocationDeclaration(node, ctx, source)
		if !ok || len(args) != 2 {
			return unsupportedIntrinsicValue(node, "int", source, ctx)
		}
		return stdjavaCall(ctx, declaration.helper, args...)
	})
	// A reference has no invocation tree. Its lowering replaces this placeholder
	// only after the same declarations are selected from the actual SAM types.
	registerStaticIntrinsic("Math", "floorMod", func(ast.Expr, []ast.Expr, Ctx) ast.Expr { return nil })
	registerStaticIntrinsicExpectedArguments("Math", "floorMod", func(node *sitter.Node, ctx Ctx, source []byte) []string {
		declaration, ok := mathFloorModInvocationDeclaration(node, ctx, source)
		if !ok {
			return nil
		}
		return []string{declaration.parameters[0], declaration.parameters[1]}
	})
	registerStaticIntrinsicDerivedResultType("Math", "floorMod", func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
		declaration, ok := mathFloorModInvocationDeclaration(node, ctx, source)
		return declaration.result, ok
	})
}

func mathFloorModInvocationDeclaration(node *sitter.Node, ctx Ctx, source []byte) (mathFloorModDeclaration, bool) {
	if node == nil || invocationArgumentCount(node) != 2 {
		return mathFloorModDeclaration{}, false
	}
	// Preserve strict-before-loose Java invocation phases. The declaration list
	// is most-specific first within either phase, so no score or result guessing
	// can promote the mixed overload's int result to long.
	for phase := 0; phase < 2; phase++ {
		for _, declaration := range mathFloorModDeclarations {
			applicable := true
			for index, expected := range declaration.parameters {
				argument := invocationArgumentNode(node, index)
				_, _, ok := javaInvocationConversionCost(argument, expected, inScopeTypeParameters(ctx), ctx, source)
				if phase == 1 {
					_, _, ok = javaLooseInvocationConversionCost(argument, expected, inScopeTypeParameters(ctx), ctx, source)
				}
				// Loose invocation includes every strict conversion as well as unboxing.
				if phase == 1 && !ok {
					_, _, ok = javaInvocationConversionCost(argument, expected, inScopeTypeParameters(ctx), ctx, source)
				}
				if !ok {
					applicable = false
					break
				}
			}
			if applicable {
				return declaration, true
			}
		}
	}
	return mathFloorModDeclaration{}, false
}

// The caller has already proved a canonical static type qualifier. handled is
// true only for this registered family, including inapplicable signatures which
// must not fall back to an unadapted native callback.
func mathFloorModReferenceDeclaration(class, method string, actual []string, ctx Ctx) (declaration mathFloorModDeclaration, handled bool) {
	if class != "Math" || method != "floorMod" {
		return declaration, false
	}
	if len(actual) != 2 {
		return declaration, true
	}
	for phase := 0; phase < 2; phase++ {
		for _, candidate := range mathFloorModDeclarations {
			applicable := true
			for index, expected := range candidate.parameters {
				if _, _, ok := methodReferenceConversionCost(actual[index], expected, phase == 1, ctx); !ok {
					applicable = false
					break
				}
			}
			if applicable {
				return candidate, true
			}
		}
	}
	return declaration, true
}
