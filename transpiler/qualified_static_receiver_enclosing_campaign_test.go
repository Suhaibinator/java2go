package transpiler

import (
 "os"
 "testing"
 sitter "github.com/smacker/go-tree-sitter"
)

// The unchanged source is independently observed on JDK21 in all three modes.
// Inspect owner binding directly so a build failure cannot hide wrong clinit.
func TestQualifiedSourceReceiverEnclosingValueWitness(t *testing.T) {
 for _, mode := range []string{"instance", "static", "priorlocal"} {
  t.Run(mode, func(t *testing.T) {
   source, err := os.ReadFile("testdata/enclosing_qualifier_witness/Main.java")
   if err != nil { t.Fatal(err) }
   helper := setupParseHelper(t, string(source))
   ctx := helper.Ctx
   owner, method := "Main", "priorLocal"
   if mode == "instance" { owner, method = "InstanceOuter", "read" }
   if mode == "static" { owner, method = "StaticOuter", "read" }
   ctx.currentClass = helper.File.Symbols.FindClassScope(owner)
   if ctx.currentClass == nil { t.Fatal("missing witness owner") }
   if mode != "priorlocal" {
    if len(ctx.currentClass.Subclasses) != 1 { t.Fatal("missing sole inner owner") }
    ctx.currentClass = ctx.currentClass.Subclasses[0]
   }
   ctx.localScope = ctx.currentClass.FindMethodByName(method, nil)
   if ctx.localScope == nil || ctx.localScope.DeclarationNode == nil { t.Fatal("missing witness method") }
   var receivers []*sitter.Node
   var visit func(*sitter.Node)
   visit = func(n *sitter.Node) {
    if n.Type() == "field_access" && n.Content(source) == "Normalizer.Form" { receivers = append(receivers,n) }
    for i:=0;i<int(n.NamedChildCount());i++ { visit(n.NamedChild(i)) }
   }
   visit(ctx.localScope.DeclarationNode)
   expected := 1
   if mode == "priorlocal" { expected = 4 }
   if len(receivers) != expected { t.Fatalf("witness receiver count got%d want%d",len(receivers),expected) }
   inferred, ok := inferIdentifierJavaType("Normalizer",ctx)
   if !ok || inferred != "Holder" { t.Fatalf("lexical root type got%q ok%v wantHolder",inferred,ok) }
   for i, receiver := range receivers {
    wantType := mode == "priorlocal" && i == 0
    resolved := qualifiedSourceClassReceiver(ctx,source,receiver)
    if (resolved != nil) != wantType {
     t.Errorf("ENCLOSING_QUALIFIER_BINDING: mode=%s receiver=%d source-type=%v want=%v",mode,i,resolved!=nil,wantType)
    }
    _, isType := staticFieldQualifierScope(receiver,source,ctx)
    if isType != wantType {
     t.Errorf("ENCLOSING_QUALIFIER_BINDING: mode=%s receiver=%d static-type=%v want=%v",mode,i,isType,wantType)
    }
   }
  })
 }
}
