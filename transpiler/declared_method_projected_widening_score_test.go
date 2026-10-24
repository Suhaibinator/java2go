package transpiler

import (
 "testing"
 "github.com/NickyBoy89/java2go/nodeutil"
 sitter "github.com/smacker/go-tree-sitter"
)

func TestDeclaredMethodProjectedWideningScoreControls(t *testing.T) {
 rows := []struct {name, declarations, methods string; wantFirst, wantRefinement bool; cost int; exact bool}{
  {"collection-parent", "", `static <E extends java.lang.Number> int choose(java.util.Collection<E> value){return 1;} static int choose(Object value){return 2;} static int call(java.util.List<Integer> value){return choose(value);}`, true,true,1,false},
  {"iterable-parent", "", `static <E> int choose(java.lang.Iterable<E> value){return 1;} static int choose(Object value){return 2;} static int call(java.util.List<String> value){return choose(value);}`, true,true,1,false},
  {"nested-actual-argument", "", `static <E> int choose(java.util.Collection<E> value){return 1;} static int choose(Object value){return 2;} static int call(java.util.List<java.util.List<Integer>> value){return choose(value);}`, true,true,1,false},
  {"direct-exact-retained", "", `static <E> int choose(java.util.Collection<E> value){return 1;} static int choose(Object value){return 2;} static int call(java.util.Collection<Integer> value){return choose(value);}`, true,true,0,true},
  {"same-spelled-distinct-method-binders", "", `static <E> int choose(java.util.Collection<E> value){return 1;} static int choose(Object value){return 2;} static <E> int call(java.util.List<E> value){return choose(value);}`, true,true,1,false},
  {"wrong-number-bound", "", `static <E extends java.lang.Number> int choose(java.util.Collection<E> value){return 1;} static int choose(Object value){return 2;} static int call(java.util.List<String> value){return choose(value);}`, false,false,0,false},
  {"invariant-refusal", "", `static int choose(java.util.Collection<Integer> value){return 1;} static int choose(Object value){return 2;} static int call(java.util.List<String> value){return choose(value);}`, false,false,0,false},
  {"array-rank-refusal", "", `static <E> int choose(java.util.Collection<E> value){return 1;} static int choose(Object value){return 2;} static int call(java.util.List<Integer>[] value){return choose(value);}`, false,false,0,false},
  {"source-collection-lookalike-refusal", "class Collection<E>{}", `static <E> int choose(Collection<E> value){return 1;} static int choose(Object value){return 2;} static int call(java.util.List<Integer> value){return choose(value);}`, false,false,0,false},
  {"unbounded-caller-number-is-not-native-number", "class Number{}", `static <E extends java.lang.Number> int choose(java.util.Collection<E> value){return 1;} static int choose(Object value){return 2;} static <Number> int call(java.util.List<Number> value){return choose(value);}`, false,false,0,false},
 }
 for _,row := range rows { t.Run(row.name,func(t *testing.T){
  helper:=setupParseHelper(t,"package score;"+row.declarations+"class Pick{"+row.methods+"}")
  owner:=helper.File.Symbols.FindClassScope("Pick")
  if owner==nil {t.Fatal("missing actual owner")}
  methods:=owner.FindMethod().ByOriginalName("choose")
  if len(methods)!=2 {t.Fatalf("overloads=%d",len(methods))}
  first,object:=methods[0],methods[1]
  caller:=owner.FindMethodByName("call",nil)
  if caller==nil {t.Fatal("missing actual caller")}
  ctx:=helper.Ctx.Clone();ctx.currentClass=owner;ctx.localScope=caller;ctx.expectedType="";ctx.expectedTypeRoot=nil
  invocation:=findNode(caller.DeclarationNode,"method_invocation")
  if invocation==nil {t.Fatal("missing actual invocation")}
  args:=nodeutil.SemanticNamedChildrenOf(invocation.ChildByFieldName("arguments"))
  if len(args)!=1 {t.Fatal("actual arity")}
  candidate,firstOK:=scoreMethodCandidate(first,owner,methodCandidateTypeParameterNames(owner,first),args,ctx,helper.File.Source)
  fallback,objectOK:=scoreMethodCandidate(object,owner,methodCandidateTypeParameterNames(owner,object),args,ctx,helper.File.Source)
  selected:=findBestMethodInHierarchy(owner,"choose",invocation.ChildByFieldName("arguments"),false,true,ctx,helper.File.Source)
  if selected==nil {t.Fatal("no selected declaration")}
  t.Logf("first=%p object=%p selected=%p firstOK=%v score=%+v objectOK=%v objectScore=%+v",first,object,selected.def,firstOK,candidate,objectOK,fallback)
  if !objectOK {t.Fatal("existing Object candidate lost")}
  if !row.wantFirst {if firstOK || selected.def!=object {t.Fatal("refused candidate gained applicability/selection")};return}
  if !firstOK || selected.def!=first {t.Fatal("proved generic argument conversion loses to Object")}
  if candidate.phase!=0 || candidate.totalCost!=row.cost || candidate.exactCount!=map[bool]int{true:1,false:0}[row.exact] {t.Fatalf("widening/exact/phase metadata=%+v",candidate)}
  formals:=methodFormalOrigins(first,owner,nil,ctx)
  projected,proved:=declaredMethodArgumentInferredOrigin(formals[0],first,formals,args,false,ctx,helper.File.Source)
  if proved!=row.wantRefinement {t.Fatalf("origin projection=%v",proved)}
  var actual inferredJavaTypeOrigin;typ,known:=inferExprJavaType(args[0],ctx,helper.File.Source,&actual);actual.javaType=typ
  if !known || len(actual.arguments)!=1 || len(projected.arguments)!=1 || !sameJavaTypeOrigin(actual.arguments[0],projected.arguments[0]) {t.Fatal("supplied argument identity changed")}
  if row.name=="same-spelled-distinct-method-binders" {if len(caller.TypeParameters)!=1 || caller.TypeParameters[0].Declaration==first.TypeParameters[0].Declaration || actual.arguments[0].parameter!=caller.TypeParameters[0].Declaration || projected.arguments[0].parameter!=caller.TypeParameters[0].Declaration {t.Fatal("caller binder recaptured by callee spelling")}}
 }) }
}

func TestProjectedWideningOriginalStaticImportSelectedDeclaration(t *testing.T) {
 ctx:=sourceGenericViewDemandTestContext(t,map[string]string{
 "library/Selectors.java": "package library;\nimport java.util.Collection;\npublic class Selectors {\n public static <E extends Number> int select(Collection<E> values) { return 1; }\n public static int select(Object value) { return 2; }\n}",
 "probe/Main.java": "package probe;\nimport static library.Selectors.select;\nimport java.util.ArrayList;\nimport java.util.List;\npublic class Main {\n static <Number> int fromShadowedCaller(Number ignored) {\n  List<Integer> values = new ArrayList<Integer>(); values.add(7);\n  return select(values);\n }\n public static void main(String[] args) {\n  List<Integer> values = new ArrayList<Integer>(); values.add(8);\n  System.out.println(select(values) + \":\" + fromShadowedCaller(\"caller\"));\n }\n}",
 "probe/Number.java": "package probe; public class Number {}",
 })
 owner:=findQualifiedSourceClass("library.Selectors");callerOwner:=findQualifiedSourceClass("probe.Main")
 if owner==nil||callerOwner==nil {t.Fatal("original owners missing")}
 methods:=owner.FindMethod().ByOriginalName("select");if len(methods)!=2 {t.Fatal("original overloads missing")}
 first,object:=methods[0],methods[1];if len(first.TypeParameters)==0 {first,object=object,first}
 for _,name:=range []string{"main","fromShadowedCaller"} {t.Run(name,func(t *testing.T){
  caller:=callerOwner.FindMethodByName(name,nil);if caller==nil {t.Fatal("original caller missing")}
  callCtx:=classScopeCtx(callerOwner,ctx);callCtx.localScope=caller;callCtx.expectedType="";callCtx.expectedTypeRoot=nil
  var selectedCall *sitter.Node;var visit func(*sitter.Node)
  visit=func(n *sitter.Node){if n==nil{return};if n.Type()=="method_invocation"&&n.ChildByFieldName("object")==nil {if id:=n.ChildByFieldName("name");id!=nil&&id.Content(callCtx.currentFile.Source)=="select" {if selectedCall!=nil {t.Fatal("multiple original calls")};selectedCall=n}};for _,ch:=range nodeutil.NamedChildrenOf(n){visit(ch)}}
  visit(caller.DeclarationNode);if selectedCall==nil {t.Fatal("original call missing")}
  resolution:=resolveStaticImportedMethod(selectedCall,callCtx,callCtx.currentFile.Source)
  if resolution.source==nil {t.Fatalf("no actual selected source declaration: %+v",resolution)}
  t.Logf("ORIGINAL selection first=%p object=%p actual=%p owner=%p caller=%p",first,object,resolution.source.def,resolution.source.owner,caller)
  if resolution.source.def!=first || resolution.source.owner!=owner {t.Fatal("actual original overload selects Object instead of Collection<E>")}
 })}
}
