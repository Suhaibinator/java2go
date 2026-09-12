package transpiler

import (
	"os"
	"strings"
	"testing"
)

func TestFuturesApplications(t *testing.T) {
	// Instrument the generated module as well as the transpiler process.
	t.Setenv("GOFLAGS", strings.TrimSpace(os.Getenv("GOFLAGS")+" -race"))
	for _, app := range []struct{ name, fixture, want string }{
		{"ThreadLifecycle", "thread_lifecycle", "false:false:restarted:true:false"},
		{"CallableForms", "callable_forms", "13:17:13:13:17:19"},
		{"FutureBatch", "future_batch", "42:saved:2:task failed:7:true:true:true:rejected"},
		{"FutureCancellation", "future_cancellation", "timeout:true:true:true:cancelled:false:false:11:true"},
	} {
		t.Run(app.name, func(t *testing.T) {
			src, err := os.ReadFile("../testfiles/applications/" + app.fixture + "/src/parity/" + app.fixture + "/" + app.name + ".java")
			if err != nil {
				t.Fatal(err)
			}
			out := renderGoFileFromJava(t, strings.SplitN(string(src), "\n", 2)[1])
			runGeneratedWithStdjava(t, out, "package main\nimport \"testing\"\nfunc TestApplication(t *testing.T) { if got := Run(); got != "+"`"+app.want+"`"+" { t.Fatalf(\"got %q\", got) } }\n")
		})
	}
}
