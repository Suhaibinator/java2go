package transpiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Generated test modules use the exact runtime dependency graph and checksums
// so readonly builds need no network resolution or module-file mutation.
func prepareCampaignReadonlyRuntimeModule(t *testing.T, generated, repo string) {
	t.Helper()
	runtimeModule, err := os.ReadFile(filepath.Join(repo, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	directive, blocks, ok := strings.Cut(string(runtimeModule), "\n")
	if !ok || !strings.HasPrefix(directive, "module ") {
		t.Fatal("runtime module directive missing")
	}
	moduleName := strings.TrimSpace(strings.TrimPrefix(directive, "module "))
	generatedDirective := "module generated"
	if existing, err := os.ReadFile(filepath.Join(generated, "go.mod")); err == nil {
		count := 0
		for _, line := range strings.Split(string(existing), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 0 || fields[0] != "module" {
				continue
			}
			count++
			if count != 1 || len(fields) != 2 {
				t.Fatal("generated module requires exactly one well-formed module directive")
			}
			generatedDirective = strings.TrimSpace(line)
		}
		if count != 1 {
			t.Fatal("generated module directive missing")
		}
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	module := generatedDirective + "\n" + blocks + fmt.Sprintf("\nrequire %s v0.0.0\nreplace %s => %s\n", moduleName, moduleName, strconv.Quote(filepath.ToSlash(repo)))
	if err := writeProjectFile(filepath.Join(generated, "go.mod"), []byte(module)); err != nil {
		t.Fatal(err)
	}
	sums, err := os.ReadFile(filepath.Join(repo, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if err := writeProjectFile(filepath.Join(generated, "go.sum"), sums); err != nil {
		t.Fatal(err)
	}
}
