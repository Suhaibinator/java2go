package transpiler

import (
    "go/ast"
    "strconv"
    "testing"
)

// This complete Java declaration was independently compiled by JDK21; six
// stateful Main runs resolve its header to the real JDK Function and its body
// to Length.Function. The existing callback campaign remains unchanged.
func TestSourceFunctionHeaderShadowNominalRegistration(t *testing.T) {
    h := setupParseHelper(t, `package app;
import java.util.function.Function;
public final class Length implements Function<String,Integer> {
    private final int bias;
    private int calls;
    private int total;
    public Length(int bias) { this.bias = bias; }
    public Integer apply(String value) {
        calls++;
        total += value.length();
        return bias + value.length() + calls;
    }
    public int calls() { return calls; }
    public int total() { return total; }
    public String bodyMarker() {
        Function<String,Integer> marker = new Function<>();
        return marker.label();
    }
    public static final class Function<T,R> {
        public String label() { return "body"; }
    }
}
`)
    ctx := h.Ctx.Clone()
    for _, scope := range h.File.Symbols.TopLevelClasses {
        if scope.Class.OriginalName == "Length" {
            ctx.currentClass = scope
            ctx.className = scope.Class.Name
        }
    }
    if ctx.currentClass == nil || ctx.currentClass.Class.OriginalName != "Length" {
        t.Fatal("missing independently validated Length declaration")
    }
    declarations := map[string]ast.Decl{
        "named": sourceClassRegistrationDecl(ctx.currentClass, ctx),
        "synthetic": syntheticReferenceRegistrationDecl(ctx.className, "app.Length", nil, nil, ctx),
    }
    for _, kind := range []string{"named", "synthetic"} {
        t.Run(kind, func(t *testing.T) {
            found := false
            ast.Inspect(declarations[kind], func(node ast.Node) bool {
                call, ok := node.(*ast.CallExpr)
                if !ok { return true }
                selector, ok := call.Fun.(*ast.SelectorExpr)
                if !ok || selector.Sel.Name != "RegisterJavaType" { return true }
                for _, argument := range call.Args {
                    id, ok := argument.(*ast.CallExpr)
                    if !ok || len(id.Args) != 1 { continue }
                    literal, ok := id.Args[0].(*ast.BasicLit)
                    if !ok { continue }
                    value, err := strconv.Unquote(literal.Value)
                    if err == nil && value == "java.util.function.Function" { found = true }
                }
                return false
            })
            if !found { t.Fatal("valid JDK Function header lost nominal edge to body-only nested Function") }
        })
    }
}
