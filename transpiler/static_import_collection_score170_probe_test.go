package transpiler

import (
 "testing"
 "github.com/NickyBoy89/java2go/nodeutil"
 sitter "github.com/smacker/go-tree-sitter"
)

// Diagnostic/TDD-only addition: actual original Java inputs and existing APIs.
// No expected JVM output, Java declaration or production selector is replaced.
func TestStaticImportCollectionScore170OriginalDeclarationProbe(t *testing.T) {
 ctx := sourceGenericViewDemandTestContext(t, map[string]string{
  "library/Selectors.java": "package library;\nimport java.util.Collection;\npublic class Selectors {\n public static <E extends Number> int select(Collection<E> values) { return 1; }\n public static int select(Object value) { return 2; }\n}",
  "probe/Main.java": "package probe;\nimport static library.Selectors.select;\nimport java.util.ArrayList;\nimport java.util.List;\npublic class Main {\n static <Number> int fromShadowedCaller(Number ignored) {\n  List<Integer> values = new ArrayList<Integer>(); values.add(7);\n  return select(values);\n }\n public static void main(String[] args) {\n  List<Integer> values = new ArrayList<Integer>(); values.add(8);\n  System.out.println(select(values) + \":\" + fromShadowedCaller(\"caller\"));\n }\n}",
  "probe/Number.java": "package probe; public class Number {}",
 })
 library := findQualifiedSourceClass("library.Selectors")
 callerOwner := findQualifiedSourceClass("probe.Main")
 sourceNumber := findQualifiedSourceClass("probe.Number")
 if library == nil || callerOwner == nil || sourceNumber == nil { t.Fatal("original declarations missing") }
 var generic, object = library.FindMethod().ByOriginalName("select")[0], library.FindMethod().ByOriginalName("select")[1]
 if len(generic.TypeParameters) == 0 { generic, object = object, generic }
 if len(generic.TypeParameters) != 1 || generic.TypeParameters[0].Declaration == nil || len(generic.Parameters) != 1 || len(object.TypeParameters) != 0 || len(object.Parameters) != 1 { t.Fatal("original overload declarations missing") }
 e := generic.TypeParameters[0].Declaration
 declaration := classScopeCtx(library, ctx)
 declaration.localScope = generic
 declaration.syntheticTypeParameters = nil
 qualified := qualifyTypeParameterBounds(generic.TypeParameters, declaration)
 if len(qualified) != 1 || len(qualified[0].Bounds) != 1 { t.Fatal("actual E bound missing") }
 bound := declaredJavaTypeOrigin(qualified[0].Bounds[0], declaration)
 t.Logf("DECL generic=%p object=%p owner=%p E=%p bound=%+v sourceNumber=%p", generic, object, library, e, bound, sourceNumber)
 if bound.parameter != nil || bound.nominalScope != nil || bound.externalName != "java.lang.Number" || bound.unresolved { t.Fatal("callee Number rebound to source/caller namespace") }
 for _, callerName := range []string{"main", "fromShadowedCaller"} {
  t.Run(callerName, func(t *testing.T) {
   caller := callerOwner.FindMethodByName(callerName, nil)
   if caller == nil { t.Fatal("original caller missing") }
   inference := classScopeCtx(callerOwner, ctx)
   inference.localScope = caller
   inference.expectedType, inference.expectedTypeRoot = "", nil
   if callerName == "fromShadowedCaller" {
    if len(caller.TypeParameters) != 1 || caller.TypeParameters[0].Declaration == nil || caller.TypeParameters[0].Declaration == e || caller.TypeParameters[0].Name != "Number" { t.Fatal("distinct caller Number binder missing") }
    t.Logf("CALLER Number declaration=%p != E=%p", caller.TypeParameters[0].Declaration, e)
   }
   var calls []*sitter.Node
   var visit func(*sitter.Node)
   source := inference.currentFile.Source
   visit = func(node *sitter.Node) {
    if node == nil { return }
    if node.Type() == "method_invocation" && node.ChildByFieldName("object") == nil {
     if name := node.ChildByFieldName("name"); name != nil && name.Content(source) == "select" { calls = append(calls, node) }
    }
    for _, child := range nodeutil.NamedChildrenOf(node) { visit(child) }
   }
   visit(caller.DeclarationNode)
   if len(calls) != 1 { t.Fatalf("original select calls=%d", len(calls)) }
   invocation := calls[0]
   args := nodeutil.SemanticNamedChildrenOf(invocation.ChildByFieldName("arguments"))
   if len(args) != 1 { t.Fatal("original argument arity changed") }
   var actual inferredJavaTypeOrigin
   typ, known := inferExprJavaType(args[0], inference, source, &actual)
   actual.javaType = typ
   formals := methodFormalOrigins(generic, library, nil, inference)
   if len(formals) != 1 || len(formals[0].arguments) != 1 || formals[0].arguments[0].parameter != e { t.Fatal("formal lost exact callee-owned E") }
   initialCost, initialExact, initialApplies, initialHandled := declaredOriginConversion(actual, formals[0], inference)
   projected, proved := declaredMethodArgumentInferredOrigin(formals[0], generic, formals, args, false, inference, source)
   projectedCost, projectedExact, projectedApplies, projectedHandled := declaredOriginConversion(actual, projected, inference)
   fallbackExpected := receiverInstantiatedMethodFormal(generic.Parameters[0], generic, library, nil, inference)
   fallbackCost, fallbackExact, fallbackApplies := javaInvocationConversionCost(args[0], fallbackExpected, methodCandidateTypeParameterNames(library, generic), inference, source)
   constraints, boundsOK := methodInvocationConstraintBindings(generic, library, args, false, inference, source)
   genericScore, genericOK := scoreMethodCandidate(generic, library, methodCandidateTypeParameterNames(library, generic), args, inference, source)
   objectScore, objectOK := scoreMethodCandidate(object, library, methodCandidateTypeParameterNames(library, object), args, inference, source)
   resolved := resolveStaticImportedMethod(invocation, inference, source)
   t.Logf("ACTUAL known=%v type=%q origin=%+v FORMAL=%+v", known, typ, actual, formals[0])
   t.Logf("INITIAL cost=%d exact=%v applies=%v handled=%v FALLBACK expected=%q cost=%d exact=%v applies=%v", initialCost, initialExact, initialApplies, initialHandled, fallbackExpected, fallbackCost, fallbackExact, fallbackApplies)
   t.Logf("PROJECT proved=%v origin=%+v cost=%d exact=%v applies=%v handled=%v BOUNDS=%v bindings=%v", proved, projected, projectedCost, projectedExact, projectedApplies, projectedHandled, boundsOK, constraints)
   t.Logf("SCORES genericOK=%v generic=%+v objectOK=%v object=%+v SELECTED=%+v", genericOK, genericScore, objectOK, objectScore, resolved)
   if !known || actual.externalName != "java.util.List" || len(actual.arguments) != 1 || actual.arguments[0].externalName != "java.lang.Integer" { t.Fatal("actual original List<Integer> origin not proved") }
   if !proved || projected.externalName != "java.util.Collection" || len(projected.arguments) != 1 || !sameJavaTypeOrigin(projected.arguments[0], actual.arguments[0]) { t.Fatal("projection did not preserve exact supplied Integer origin") }
   if !genericOK || !objectOK || !boundsOK { t.Fatal("candidate refusal precedes score hypothesis; retain actual trace before proposing a scorer repair") }
   if !projectedHandled || !projectedApplies || projectedExact || projectedCost != 1 { t.Fatal("actual ordinary generic-parent widening does not match proposed score hypothesis") }
   if !methodCandidateScoreBetter(genericScore, objectScore) || resolved.source == nil || resolved.source.def != generic { t.Fatal("original applicable Collection<E> widening loses to Object overload") }
  })
 }
}
