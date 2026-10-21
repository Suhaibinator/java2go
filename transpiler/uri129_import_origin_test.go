package transpiler

import "testing"

func TestURI129ImportedOwnerDoesNotReparseSyntheticQualifiedPrefix(t *testing.T) {
	for _, member := range []string{"box.value", "box.get()"} {
		t.Run(member, func(t *testing.T) {
			ctx := ordinaryDemandTestContext(t, map[string]string{
				"p/Box.java": `package p;import java.net.URI;public class Box {static class java {}public URI value;public URI get(){return value;}}`,
				"Probe.java": `class Probe {<URI>Object value(p.Box box){return ` + member + `.normalize();}}`,
			}, "Probe.java")
			strictRoutingState(t)
			var observed any
			func() {
				defer func() { observed = recover() }()
				node := ctx.currentClass.Class.DeclarationNode
				for node.Parent() != nil {
					node = node.Parent()
				}
				ParseNode(node, ctx.currentFile.Source, ctx)
			}()
			want := Diagnostic{Kind: "intrinsic invocation", NodeType: "method_invocation", Message: member + ".normalize()", ClassName: "probe", Line: 1}
			failure, ok := observed.(strictModeError)
			if !ok || failure.diagnostic != want {
				t.Fatalf("failure %T %#v, want %#v", observed, observed, want)
			}
			if items := Diagnostics(); len(items) != 1 || items[0] != want {
				t.Fatalf("diagnostics %#v, want only %#v", items, want)
			}
		})
	}
}
