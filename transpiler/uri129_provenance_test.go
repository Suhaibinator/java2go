package transpiler

import (
	"encoding/json"
	"os"
	"testing"
)

func TestURI129DeclarationProvenance(t *testing.T) {
	var rows []struct {
		Name, Caller, Kind string
		Files              map[string]string
		Snippet            *string
	}
	raw, err := os.ReadFile("testdata/uri129_contracts/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 22 {
		t.Fatalf("need all 22 provenance controls: %d", len(rows))
	}
	for _, row := range rows {
		t.Run(row.Name, func(t *testing.T) {
			ctx := ordinaryDemandTestContext(t, row.Files, row.Caller)
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
			if row.Snippet == nil {
				if observed != nil || len(Diagnostics()) != 0 {
					t.Fatalf("declining/admitted control acquired refusal: %T %v; %v", observed, observed, Diagnostics())
				}
				return
			}
			nodeType := "method_invocation"
			if row.Kind == "JDK owner java.net.URL" {
				nodeType = "object_creation_expression"
			}
			want := Diagnostic{Kind: row.Kind, NodeType: nodeType, Message: *row.Snippet, ClassName: "probe", Line: 1}
			failure, ok := observed.(strictModeError)
			if !ok || failure.diagnostic != want {
				t.Fatalf("strict failure = %T %#v, want %#v", observed, observed, want)
			}
			if items := Diagnostics(); len(items) != 1 || items[0] != want {
				t.Fatalf("collected %#v, want only %#v", items, want)
			}
		})
	}
}
