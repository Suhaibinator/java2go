package transpiler

import "testing"

// These new assertions leave the frozen URI126 controls byte-identical. A
// panic unrelated to the declared diagnostic is always a test failure.
func uri129RequireStrictDiagnostic(t *testing.T, source string, want Diagnostic) {
	t.Helper()
	strictRoutingState(t)
	var observed any
	func() {
		defer func() { observed = recover() }()
		renderGoFileFromJava(t, source)
	}()
	failure, ok := observed.(strictModeError)
	if !ok {
		t.Fatalf("expected strictModeError, got %T: %v", observed, observed)
	}
	if failure.diagnostic != want {
		t.Fatalf("strict diagnostic = %#v, want %#v", failure.diagnostic, want)
	}
	items := Diagnostics()
	if len(items) != 1 || items[0] != want {
		t.Fatalf("collected diagnostic = %#v, want only %#v", items, want)
	}
}

func TestURI129ExistingRefusalsHaveTypedSourceDiagnostics(t *testing.T) {
	for _, entry := range []struct{ name, source, node, snippet string }{
		{"URI seven argument constructor", `class Probe {java.net.URI value() throws Exception{return new java.net.URI("http",null,"host",-1,"/a",null,null);}}`, "object_creation_expression", `new java.net.URI("http",null,"host",-1,"/a",null,null)`},
		{"ASCII wrong arity", `class Probe {String value(java.net.URI value){return value.toASCIIString(1);}}`, "method_invocation", `value.toASCIIString(1)`},
		{"syntax exception wrong arity", `class Probe {java.net.URISyntaxException value(){return new java.net.URISyntaxException("only-one");}}`, "argument_list", `("only-one")`},
	} {
		t.Run(entry.name, func(t *testing.T) {
			uri129RequireStrictDiagnostic(t, entry.source, Diagnostic{Kind: "intrinsic invocation", NodeType: entry.node, Message: entry.snippet, ClassName: "probe", Line: 1})
		})
	}
}

// URI.normalize remains outside the implemented service slice. URL(String)
// moved to URL701 positive conformance using its unchanged Java source.
func TestURI129UnadmittedJDKAPIsRequireTypedSourceRefusal(t *testing.T) {
	for _, entry := range []struct{ name, source, kind, node, snippet string }{
		{"URI normalize", `class Probe {Object value(java.net.URI value){return value.normalize();}}`, "intrinsic invocation", "method_invocation", `value.normalize()`},
	} {
		t.Run(entry.name, func(t *testing.T) {
			uri129RequireStrictDiagnostic(t, entry.source, Diagnostic{Kind: entry.kind, NodeType: entry.node, Message: entry.snippet, ClassName: "probe", Line: 1})
		})
	}
}
