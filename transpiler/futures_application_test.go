package transpiler

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"unicode/utf16"
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
			runGeneratedWithStdjava(t, out, fmt.Sprintf(`package main
import ("testing"; "slices")
func TestApplication(t *testing.T) { if got := Run(); got == nil || !slices.Equal(got.UTF16Copy(), %#v) { t.Fatalf("want %%q, got JavaString %%#v", %q, got) } }
`, utf16.Encode([]rune(app.want)), app.want))
		})
	}
}
