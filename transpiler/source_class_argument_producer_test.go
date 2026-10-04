package transpiler

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// These producers bypass javaTypeStringToGoTypeExpr for the complete class.
// They must instantiate the same preserved List<String> constraint as fields.
func TestBoundedSourceClassGoArgumentProducers(t *testing.T) {
	t.Run("raw_subobject", func(t *testing.T) {
		ctx := sourceGenericViewDemandTestContext(t, map[string]string{
			"p/Holder.java": `package p;public class Holder<T extends java.util.List<String>>{public T value;}`,
			"app/Use.java": `package app;public class Use{}`,
		})
		ctx = classScopeCtx(findQualifiedSourceClass("app.Use"), ctx)
		got := boundedArgumentTestGoType(t, classSubobjectPointerTypeExpr(findQualifiedSourceClass("p.Holder"), nil, ctx.currentClass, ctx))
		want := "*p.Holder[*stdjava.List[*stdjava.JavaString]]"
		if got != want {
			t.Errorf("Go raw superclass subobject representation = %s, want %s", got, want)
		}
	})
	for _, test := range []struct{ name, declaration string }{
		{"raw_construction", `public class Use{public Object make(){return new p.Holder();}}`},
		{"raw_implicit_super", `public class Use extends p.Holder{}`},
		{"raw_explicit_super", `public class Use extends p.Holder{public Use(){super();}}`},
		{"concrete_construction", `public class Use{public Object make(){return new p.Holder<java.util.List<java.lang.String>>();}}`},
		{"diamond_target_construction", `public class Use{public p.Holder<java.util.List<java.lang.String>> make(){return new p.Holder<>();}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			for name, source := range map[string]string{
				"p/Holder.java": `package p;public class Holder<T extends java.util.List<String>>{public T value;public Holder(){}}`,
				"app/Use.java": "package app;" + test.declaration,
			} {
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { t.Fatal(err) }
				if err := os.WriteFile(path, []byte(source), 0o644); err != nil { t.Fatal(err) }
			}
			out := convertJavaProjectDir(t, root)["app/Use.go"]
			pattern := `p\.NewHolder(?:Java2goWithSelf)?(?:Java2goExecution)?\[\*stdjava\.List\[\*stdjava\.JavaString\]\]`
			if !regexp.MustCompile(pattern).MatchString(out) {
				t.Errorf("Go constructor class argument does not preserve parameterized bound:\n%s", out)
			}
		})
	}
}
