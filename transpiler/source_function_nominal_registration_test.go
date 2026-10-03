package transpiler

import (
 "go/ast"
 "strconv"
 "testing"
)

func TestSourceFunctionNominalRegistration(t *testing.T) {
 cases := []struct { name, source string; function bool }{
  {"qualified", `class Length implements java.util.function.Function<String,Integer> { public Integer apply(String text) { return 1; } }`, true},
  {"imported", `import java.util.function.Function; class Length implements Function<String,Integer> { public Integer apply(String text) { return 1; } }`, true},
  {"extended-interface", `interface Length extends java.util.function.Function<String,Integer> {}`, true},
  {"shape-only", `class Length { public Integer apply(String text) { return 1; } }`, false},
  {"source-shadow", `interface Function<T,R> { R apply(T value); } class Length implements Function<String,Integer> { public Integer apply(String text) { return 1; } }`, false},
  {"external-name-only", `class Length implements other.Function<String,Integer> { public Integer apply(String text) { return 1; } }`, false},
 }
 for _, test := range cases {
  t.Run(test.name, func(t *testing.T) {
   h := setupParseHelper(t, test.source)
   ctx := h.Ctx.Clone()
   for _, scope := range h.File.Symbols.TopLevelClasses { if scope.Class.OriginalName == "Length" { ctx.currentClass = scope; ctx.className = scope.Class.Name } }
   if ctx.currentClass.Class.OriginalName != "Length" { t.Fatal("missing source declaration") }
   declarations := map[string]ast.Decl{
    "named": sourceClassRegistrationDecl(ctx.currentClass, ctx),
    "synthetic": syntheticReferenceRegistrationDecl(ctx.className, "fixture.Length", nil, nil, ctx),
   }
   for _, kind := range []string{"named", "synthetic"} {
    t.Run(kind, func(t *testing.T) {
     found := false
     ast.Inspect(declarations[kind], func(node ast.Node) bool {
      call, ok := node.(*ast.CallExpr); if !ok { return true }
      selector, ok := call.Fun.(*ast.SelectorExpr); if !ok || selector.Sel.Name != "RegisterJavaType" { return true }
      for _, argument := range call.Args {
       id, ok := argument.(*ast.CallExpr); if !ok || len(id.Args)!=1 { continue }
       literal, ok := id.Args[0].(*ast.BasicLit); if !ok { continue }
       value, err := strconv.Unquote(literal.Value); if err==nil && value=="java.util.function.Function" { found=true }
      }
      return false
     })
     if found != test.function { t.Fatalf("canonical Function edge = %v, want %v; only exact declared membership may authorize ObjectView", found, test.function) }
    })
   }
  })
 }
}
