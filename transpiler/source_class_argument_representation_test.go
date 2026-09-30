package transpiler

import (
    "bytes"
    "go/ast"
    "go/printer"
    "go/token"
    "reflect"
    "testing"
)

func boundedArgumentTestGoType(t *testing.T, expr ast.Expr) string {
    t.Helper()
    var buffer bytes.Buffer
    if err := printer.Fprint(&buffer, token.NewFileSet(), expr); err != nil { t.Fatal(err) }
    return buffer.String()
}

// The original JVM-proven shadow_guards program retains T extends List<String>
// and casts null to BadBound<?>. This general source-declaration matrix tests
// the corresponding Go instantiations without changing Java erasure or bounds.
func TestBoundedSourceClassGoArguments(t *testing.T) {
    ctx := sourceGenericViewDemandTestContext(t, map[string]string{
        "p/Holder.java": `package p;public class Holder<T extends java.util.List<String>>{public T value;}`,
        "p/Dependent.java": `package p;public class Dependent<T extends java.util.List<String>,U extends T>{public U value;}`,
        "p/Free.java": `package p;public class Free<T>{public T value;}`,
        "app/Use.java": `package app;public class Use{p.Holder<?> value;}`,
        "app/String.java": `package app;class String{}`,
        "app/List.java": `package app;class List<T>{}`,
    })
    ctx = classScopeCtx(findQualifiedSourceClass("app.Use"), ctx)
    for _, test := range []struct {name, source, want string}{
        {"bounded wildcard", "p.Holder<?>", "*p.Holder[*stdjava.List[*stdjava.JavaString]]"},
        {"bounded raw", "p.Holder", "*p.Holder[*stdjava.List[*stdjava.JavaString]]"},
        {"bounded lower wildcard", "p.Holder<? super java.util.ArrayList<java.lang.String>>", "*p.Holder[*stdjava.List[*stdjava.JavaString]]"},
        {"bounded upper wildcard", "p.Holder<? extends java.util.ArrayList<java.lang.String>>", "*p.Holder[*stdjava.List[*stdjava.JavaString]]"},
        {"concrete argument", "p.Holder<java.util.List<java.lang.String>>", "*p.Holder[*stdjava.List[*stdjava.JavaString]]"},
        {"dependent raw", "p.Dependent", "*p.Dependent[*stdjava.List[*stdjava.JavaString], *stdjava.List[*stdjava.JavaString]]"},
        {"dependent wildcard", "p.Dependent<java.util.List<java.lang.String>,?>", "*p.Dependent[*stdjava.List[*stdjava.JavaString], *stdjava.List[*stdjava.JavaString]]"},
        {"unbounded wildcard", "p.Free<?>", "*p.Free[any]"},
    } {
        t.Run(test.name, func(t *testing.T) {
            got := boundedArgumentTestGoType(t, javaTypeStringToGoTypeExpr(test.source, nil, ctx))
            if got != test.want { t.Errorf("Go source-class argument representation = %s, want %s", got, test.want) }
        })
    }
    holder := findQualifiedSourceClass("p.Holder")
    if canonicalGenericFamily(holder, ctx) != nil { t.Error("bounded runtime family bypassed complete plan audit") }
    if holder.TypeParameters[0].Bounds[0].Original != "java.util.List<String>" { t.Error("authoritative Java parameterized bound was changed") }
    if got := normalizeClassTypeArguments(holder, nil, nil, nil); !reflect.DeepEqual(got, []string{"java.util.List"}) { t.Errorf("Java raw erasure was changed: %#v", got) }
    fields := makeTypeParamFieldsInContext(holder.TypeParameters, classScopeCtx(holder, ctx))
    if got := boundedArgumentTestGoType(t, fields[0].Type); got != "*stdjava.List[*stdjava.JavaString]" { t.Errorf("declared Go constraint was weakened: %s", got) }
}
