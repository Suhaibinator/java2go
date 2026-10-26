package transpiler

import "testing"

func TestThreadSourceAncestryDeclarationCertificate(t *testing.T) {
	for _, test := range []struct {
		source string
		want   bool
	}{
		{`class Probe extends java.lang.Thread {}`, true},
		{`class Base extends java.lang.Thread {} class Probe extends Base {}`, true},
		{`import foreign.Thread; class Probe extends Thread {}`, false},
		{`class Probe extends foreign.Thread {}`, false},
		{`class Thread {} class Probe extends Thread {}`, false},
		{`class Probe<Thread> extends Thread {}`, false},
		{`class Probe<java> extends java.lang.Thread {}`, false},
		{`class Probe extends Missing {}`, false},
	} {
		t.Run(test.source, func(t *testing.T) {
			h := setupParseHelper(t, test.source)
			scope := resolveClassScopeByQualifiedName(h.Ctx, "Probe")
			if actual := classScopeExtendsThread(h.Ctx, scope); actual != test.want {
				t.Fatalf("declaration certificate=%v want%v", actual, test.want)
			}
		})
	}
}
