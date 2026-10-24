package transpiler

import (
	"go/ast"
	"testing"
)

func TestSourceReflectionDeclaredNestHostTDD(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{
		"p/Host.java": `package p; public class Host<T> {
   public Class<?> metadata(){return getClass();}
   public static class Holder { private volatile int value; public static class Deep {} }
   public static class Sibling {}
   public class Member {}
  }`,
		"p/Unrelated.java":  `package p; public class Unrelated {}`,
		"p/Host$Spoof.java": `package p; public class Host$Spoof {}`,
	})
	for _, test := range []struct {
		name, source, host string
		nested             bool
	}{
		{"outer_self", "p.Host", "p.Host", false},
		{"static_holder", "p.Host.Holder", "p.Host", true},
		{"static_sibling", "p.Host.Sibling", "p.Host", true},
		{"deep_static", "p.Host.Holder.Deep", "p.Host", true},
		{"generic_enclosing_member", "p.Host.Member", "p.Host", true},
		{"unrelated_self", "p.Unrelated", "p.Unrelated", false},
		{"literal_dollar_top_level_self", "p.Host$Spoof", "p.Host$Spoof", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			scope := findQualifiedSourceClass(test.source)
			if scope == nil {
				t.Fatal("parsed declaration missing", test.source)
			}
			if (scope.Enclosing != nil) != test.nested {
				t.Fatal("parsed nesting control differs")
			}
			if test.name == "static_holder" && scope.IsInner {
				t.Fatal("static scope unexpectedly requires outer instance")
			}
			statement := sourceClassMetadataStmt(scope, classScopeCtx(scope, ctx))
			expression, ok := statement.(*ast.ExprStmt)
			if !ok {
				t.Fatalf("class metadata statement %T", statement)
			}
			call, ok := expression.X.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				t.Fatal("registration shape")
			}
			descriptor := reflectCoreComposite(t, call.Args[0])
			nest := reflectCoreOptionalKey(descriptor, "NestHost")
			if nest == nil {
				t.Fatal("declared NestHost metadata missing")
			}
			if got := reflectCoreTypeID(t, nest); got != test.host {
				t.Fatalf("declared NestHost=%q want=%q", got, test.host)
			}
		})
	}
}
