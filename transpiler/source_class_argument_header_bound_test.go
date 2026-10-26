package transpiler

import (
	"go/ast"
	"os"
	"path/filepath"
	"testing"
)

func boundedHeaderOracleTestContext(t *testing.T) Ctx {
	t.Helper()
	sources := map[string]string{}
	for _, name := range []string{"app/Holder.java", "app/StringHolder.java", "app/Main.java", "origin/Limit.java"} {
		contents, err := os.ReadFile(filepath.Join("testdata", "source_bounded_header_identity", name))
		if err != nil { t.Fatal(err) }
		sources[name] = string(contents)
	}
	return sourceGenericViewDemandTestContext(t, sources)
}

// The independent JDK21 header-bound program proves these class-header
// identities. Body method/constructor and record guards retain their scopes.
func TestBoundedSourceClassGoHeaderBoundIdentity(t *testing.T) {
	for _, test := range []struct{ name, owner, want string }{
		{"imported_limit_constraint", "app.Holder", "*origin.Limit"},
		{"java_string_constraint", "app.StringHolder", "*stdjava.List[*stdjava.JavaString]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := boundedHeaderOracleTestContext(t)
			owner := findQualifiedSourceClass(test.owner)
			fields := makeTypeParamFieldsInContext(owner.TypeParameters, classHeaderTypeCtx(owner, ctx))
			got := boundedArgumentTestGoType(t, fields[0].Type)
			if got != test.want { t.Errorf("Go class-header bound identity = %s, want %s", got, test.want) }
		})
	}
	for _, test := range []struct{ name, source, want string }{
		{"imported_limit_wildcard", "app.Holder<?>", "*Holder[*origin.Limit]"},
		{"imported_limit_raw", "app.Holder", "*Holder[*origin.Limit]"},
		{"java_string_wildcard", "app.StringHolder<?>", "*StringHolder[*stdjava.List[*stdjava.JavaString]]"},
		{"java_string_raw", "app.StringHolder", "*StringHolder[*stdjava.List[*stdjava.JavaString]]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := boundedHeaderOracleTestContext(t)
			ctx = classScopeCtx(findQualifiedSourceClass("app.Main"), ctx)
			got := boundedArgumentTestGoType(t, javaTypeStringToGoTypeExpr(test.source, nil, ctx))
			if got != test.want { t.Errorf("Go class-header raw/wildcard identity = %s, want %s", got, test.want) }
		})
	}
	for _, test := range []struct{ name, kind string }{
		{"method_body_bound_guard", "method"},
		{"constructor_body_bound_guard", "constructor"},
		{"record_header_member_guard", "record"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := sourceGenericViewDemandTestContext(t, map[string]string{
				"origin/Limit.java": `package origin;public class Limit{}`,
				"app/BodyOwner.java": `package app;import origin.Limit;public class BodyOwner<T extends origin.Limit>{public static class Limit{} public <M extends Limit> void body(M value){} public <C extends Limit> BodyOwner(C value){}}`,
				"app/RecordOwner.java": `package app;import origin.Limit;public record RecordOwner<T extends Limit>(T value){public static class Limit{}}`,
			})
			if test.kind == "record" {
				owner := findQualifiedSourceClass("app.RecordOwner")
				member := findQualifiedSourceClass("app.RecordOwner.Limit")
				fields := makeTypeParamFieldsInContext(owner.TypeParameters, classHeaderTypeCtx(owner, ctx))
				want := boundedArgumentTestGoType(t, &ast.StarExpr{X: &ast.Ident{Name: member.Class.Name}})
				if got := boundedArgumentTestGoType(t, fields[0].Type); got != want { t.Errorf("record header member namespace changed: %s, want %s", got, want) }
				return
			}
			owner := findQualifiedSourceClass("app.BodyOwner")
			member := findQualifiedSourceClass("app.BodyOwner.Limit")
			ctx = classScopeCtx(owner, ctx)
			for _, method := range owner.Methods {
				if len(method.TypeParameters) == 0 || method.Constructor != (test.kind == "constructor") { continue }
				ctx.localScope = method
				fields := makeTypeParamFieldsInContext(method.TypeParameters, ctx)
				want := boundedArgumentTestGoType(t, &ast.StarExpr{X: &ast.Ident{Name: member.Class.Name}})
				if got := boundedArgumentTestGoType(t, fields[0].Type); got != want { t.Errorf("method/constructor body member namespace changed: %s, want %s", got, want) }
				return
			}
			t.Fatal("missing method/constructor binder control")
		})
	}
}
