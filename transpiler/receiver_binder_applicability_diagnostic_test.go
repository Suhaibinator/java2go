package transpiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func receiverBinderDiagnosticOrigin(origin inferredJavaTypeOrigin, class, method *symbol.TypeParamDeclaration) string {
	parameter := ""
	switch origin.parameter {
	case nil:
	case class:
		parameter = "CLASS"
	case method:
		parameter = "METHOD"
	default:
		parameter = fmt.Sprintf("FOREIGN(%p)", origin.parameter)
	}
	scope := ""
	if origin.nominalScope != nil {
		scope = origin.nominalScope.Class.OriginalName
	}
	args := make([]string, len(origin.arguments))
	for i, arg := range origin.arguments {
		args[i] = receiverBinderDiagnosticOrigin(arg, class, method)
	}
	return fmt.Sprintf("{java=%q source=%q external=%q parameter=%s rank=%d unresolved=%t args=[%s]}", origin.javaType, scope, origin.externalName, parameter, origin.arrayRank, origin.unresolved, strings.Join(args, ","))
}

func TestReceiverBinderApplicabilityOriginalShadowCalls(t *testing.T) {
	for _, tc := range []struct{ name, source, owner, method string }{
		{"class-inferred", innerCarrierIdentitySource, "Probe", "run"},
		{"interface-explicit", hiddenLexicalReceiverSource + `class Use { static int run(Carrier<String> owner, Carrier<String>.Cell<Child> same) { return owner.<Child>shadow(same); } }`, "Use", "run"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, tc.source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = helper.File.Symbols.FindClassScope(tc.owner)
			if ctx.currentClass == nil {
				t.Fatal("caller declaration missing")
			}
			ctx.className = ctx.currentClass.Class.Name
			for _, m := range ctx.currentClass.Methods {
				if m.OriginalName == tc.method {
					ctx.localScope = m
				}
			}
			carrier := helper.File.Symbols.FindClassScope("Carrier")
			if carrier == nil || len(carrier.TypeParameters) != 1 {
				t.Fatal("carrier declaration missing")
			}
			var generic *symbol.Definition
			for _, m := range carrier.Methods {
				if m.OriginalName == "shadow" && len(m.TypeParameters) == 1 {
					generic = m
				}
			}
			if generic == nil {
				t.Fatal("generic declaration missing")
			}
			classParameter := carrier.TypeParameters[0].Declaration
			methodParameter := generic.TypeParameters[0].Declaration
			if classParameter == methodParameter || generic.Parameters[0].TypeParameterBindings["T"] != methodParameter {
				t.Fatal("actual distinct class/method binder declaration contract lost")
			}
			describe := func(o inferredJavaTypeOrigin) string {
				return receiverBinderDiagnosticOrigin(o, classParameter, methodParameter)
			}
			count := 0
			var walk func(*sitter.Node)
			walk = func(n *sitter.Node) {
				if n.Type() == "method_invocation" && n.ChildByFieldName("name").Content(helper.File.Source) == "shadow" {
					count++
					target := resolveInvocationTarget(n.ChildByFieldName("object"), ctx, helper.File.Source)
					if target == nil {
						t.Fatal("actual receiver target missing")
					}
					resolution, _ := findBestMethodForInvocationTarget(target, "shadow", n.ChildByFieldName("arguments"), true, false, ctx, helper.File.Source)
					selected := "<nil>"
					if resolution != nil {
						selected = resolution.def.OriginalType + " " + resolution.def.Name
					}
					arguments := nodeutil.SemanticNamedChildrenOf(n.ChildByFieldName("arguments"))
					if len(arguments) != 1 {
						t.Fatal("original one-argument call changed")
					}
					var actual inferredJavaTypeOrigin
					actualType, known := inferExprJavaType(arguments[0], ctx, helper.File.Source, &actual)
					actual.javaType = actualType
					bindings := sourceClassOriginBindings(target.classScope, target.classTypeArgumentOrigins, carrier, ctx)
					formals := methodFormalOrigins(generic, carrier, bindings, ctx)
					ownerArgs := invocationOwnerTypeArguments(target, &methodResolution{def: generic, owner: carrier}, ctx)
					cost, exact, applies, handled := declaredOriginConversion(actual, formals[0], ctx)
					names := methodCandidateTypeParameterNames(nil, generic)
					score, scoreApplies := scoreDeclaredMethodCandidate(generic, carrier, names, arguments, ctx, helper.File.Source, ownerArgs, bindings)
					boundBindings, boundApplies := methodInvocationConstraintBindings(generic, carrier, arguments, false, ctx, helper.File.Source)
					formalString := methodParameterReferenceType(generic.Parameters[0], generic, carrier, ctx)
					instantiated := receiverInstantiatedMethodFormal(generic.Parameters[0], generic, carrier, ownerArgs, ctx)
					t.Logf("call=%s selected=%s known=%t actual=%s receiverArgs=%v classBinding=%s formalOrigin=%s conversion=%d/%t/%t/%t names=%v formalString=%s instantiated=%s boundBindings=%v boundApplicable=%t score=%+v scoreApplicable=%t", n.Content(helper.File.Source), selected, known, describe(actual), ownerArgs, describe(bindings[classParameter]), describe(formals[0]), cost, exact, applies, handled, names, formalString, instantiated, boundBindings, boundApplies, score, scoreApplies)
					wantGeneric := arguments[0].Content(helper.File.Source) != "otherBound"
					if wantGeneric && (resolution == nil || resolution.def != generic) {
						t.Errorf("actual same-owner member must select actual generic declaration; got %s", selected)
					}
					if !wantGeneric && (resolution == nil || resolution.def == generic || resolution.def.OriginalType != "long") {
						t.Errorf("different enclosing owner must preserve long Object fallback; got %s", selected)
					}
				}
				for i := 0; i < int(n.NamedChildCount()); i++ {
					walk(n.NamedChild(i))
				}
			}
			walk(ctx.localScope.DeclarationNode)
			want := 2
			if tc.name == "interface-explicit" {
				want = 1
			}
			if count != want {
				t.Fatalf("actual calls=%d want%d", count, want)
			}
		})
	}
}
