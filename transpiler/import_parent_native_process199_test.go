package transpiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

const importParent199OuterReady = "JAVA2GO_IMPORT_PARENT199_OUTER_READY"

type importParent199WorkRow struct {
	Queries                   int    `json:"queries"`
	Padding                   int    `json:"padding"`
	Anchor                    string `json:"anchor"`
	ParentEdges               int    `json:"parent_edges"`
	ActualNativeCalls         int64  `json:"actual_native_calls"`
	NativeBudget              int64  `json:"native_budget"`
	OldSourceNativePrediction int    `json:"old_source_native_prediction"`
	TotalAllocBytes           uint64 `json:"total_alloc_bytes"`
	Mallocs                   uint64 `json:"mallocs"`
	RealExistingAPI           string `json:"real_existing_API"`
	FreshSemanticResultCache  bool   `json:"fresh_semantic_result_cache"`
}

func importParent199Complete(result superclass4059ProcessResult, parent string) error {
	if err := superclass4059Completed(result, parent); err != nil {
		return err
	}
	if result.Stderr != "" {
		return fmt.Errorf("import-parent child stderr was not empty")
	}
	rows := map[string]bool{}
	for _, line := range strings.Split(result.Stdout, "\n") {
		_, encoded, found := strings.Cut(line, "IMPORT_PARENT_NATIVE_WORK199 ")
		if !found {
			continue
		}
		var fields map[string]json.RawMessage
		var row importParent199WorkRow
		decoder := json.NewDecoder(strings.NewReader(encoded))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&row); err != nil {
			return fmt.Errorf("invalid import-parent row: %w", err)
		}
		if err := json.Unmarshal([]byte(encoded), &fields); err != nil || len(fields) != 11 {
			return fmt.Errorf("incomplete import-parent fields")
		}
		for _, key := range []string{"queries", "padding", "anchor", "parent_edges", "actual_native_calls", "native_budget", "old_source_native_prediction", "total_alloc_bytes", "mallocs", "real_existing_API", "fresh_semantic_result_cache"} {
			if len(fields[key]) == 0 || bytes.Equal(fields[key], []byte("null")) {
				return fmt.Errorf("missing import-parent field %s", key)
			}
		}
		depth, known := map[string]int{"top": 1, "nested": 3, "root": 0}[row.Anchor]
		key := fmt.Sprintf("padding%d/%s", row.Padding, row.Anchor)
		if !known || (row.Padding != 0 && row.Padding != 64 && row.Padding != 512) || rows[key] || row.Queries != 1024 || row.ParentEdges != depth ||
			row.NativeBudget != int64(1024*(depth+3)) || row.ActualNativeCalls < 0 || row.ActualNativeCalls > row.NativeBudget || row.OldSourceNativePrediction != 1024*(2*depth+3) ||
			row.RealExistingAPI != "importSourceFacts" || row.FreshSemanticResultCache {
			return fmt.Errorf("unexpected or over-budget import-parent row %s", key)
		}
		rows[key] = true
	}
	if len(rows) != 9 {
		return fmt.Errorf("import-parent child completed %d of9 rows", len(rows))
	}
	for key := range rows {
		for _, event := range []string{"=== RUN   " + parent + "/" + key, "    --- PASS: " + parent + "/" + key + " ("} {
			if strings.Count(result.Stdout, "\n"+event) != 1 {
				return fmt.Errorf("missing or duplicate import-parent event %s", event)
			}
		}
	}
	expected := map[string]bool{parent: true}
	for key := range rows {
		expected[parent+"/"+key] = true
	}
	runs, passes := map[string]bool{}, map[string]bool{}
	passPattern := regexp.MustCompile(`^( *)--- PASS: ([^ ]+) \([0-9]+(?:\.[0-9]+)?s\)$`)
	for _, line := range strings.Split(result.Stdout, "\n") {
		if name, found := strings.CutPrefix(line, "=== RUN   "); found {
			if !expected[name] || runs[name] {
				return fmt.Errorf("unexpected or duplicate child RUN %s", name)
			}
			runs[name] = true
		}
		if strings.Contains(line, "--- PASS:") {
			match := passPattern.FindStringSubmatch(line)
			if match == nil || !expected[match[2]] || passes[match[2]] || (match[2] == parent && match[1] != "") || (match[2] != parent && match[1] != "    ") {
				return fmt.Errorf("unexpected or duplicate child PASS %s", line)
			}
			passes[match[2]] = true
		}
	}
	if len(runs) != 10 || len(passes) != 10 {
		return fmt.Errorf("incomplete import-parent RUN/PASS census")
	}
	return nil
}

func importParent199OuterHandshake(t *testing.T) {
	t.Helper()
	ready := os.Getenv(importParent199OuterReady)
	if !filepath.IsAbs(ready) {
		t.Fatal("outer native cleanup has no absolute handshake path199")
	}
	file, err := os.OpenFile(ready, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		if encoded, err := os.ReadFile(ready + ".ack"); err == nil {
			if string(encoded) != "one-native-cleanup-acknowledged\n" {
				t.Fatal("invalid outer cleanup acknowledgement199")
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("outer native cleanup acknowledgement timeout199")
}

func importParent199IsolatedEntry(t *testing.T) bool {
	t.Helper()
	if request, child := superclass4059ChildRequest(t); child {
		switch request.Mode {
		case "import-parent-work199":
			if os.Getenv(importParent199OuterReady) != "" {
				t.Fatal("unexpected import-parent outer handshake199")
			}
		case "import-parent-outer-cleanup199":
			importParent199OuterHandshake(t)
		default:
			t.Fatal("invalid import-parent measurement child mode199")
		}
		return false
	}
	if os.Getenv(importParent199OuterReady) != "" {
		t.Fatal("unowned import-parent outer handshake199")
	}
	sharedDir := superclass4059Flag("test.gocoverdir")
	if !superclass4059CanIsolate(testing.CoverMode(), sharedDir) {
		t.Log("IMPORT_PARENT199_UNISOLATED_COVERAGE_FALLBACK: original in-process assertions retained")
		return false
	}
	result := superclass4059RunChild(t, superclass4059Request{t.Name(), "import-parent-work199"}, superclass4059ProcessOptions{sharedDir: sharedDir})
	superclass4059LogResult(t, result)
	if err := importParent199Complete(result, t.Name()); err != nil {
		t.Fatal(err)
	}
	return true
}

// This separate control window uses the real warmed API and acknowledges one
// native parser finalizer. No GC or control hook enters the original nine cases.
func importParent199Contamination(t *testing.T) {
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())
	defer parser.Close()
	var trees []*sitter.Tree
	defer func() { runtime.KeepAlive(trees) }()
	ctx := importParent199OwnedContext(t, parser, campaignImportSource("alpha"), &trees)
	ctx.callableSubclasses = &callableSubclassSourceInventory{}
	original := ctx.currentFile.BaseClass.Class.DeclarationNode
	ctx.currentFile.BaseClass.Class.DeclarationNode = importParent199OldRoot(original)
	defer func() { ctx.currentFile.BaseClass.Class.DeclarationNode = original }()
	campaignAssertImportText(t, ctx, "alpha")
	inventory, want, ok := importSourceFacts(ctx)
	if !ok || !want.importsReady || !want.ordinaryReady {
		t.Fatal("root control did not warm actual import facts199")
	}
	clone := ctx.Clone()
	state := cgo179ArmCleanup(t)
	calls := runtime.NumCgoCall()
	for query := 0; query < 1024; query++ {
		if query == 512 {
			cgo179ReleaseCleanup(t, state)
		}
		gotInventory, got, known := importSourceFacts(clone)
		if !known || gotInventory != inventory || got.root != want.root || got.declaration != want.declaration || !got.importsReady || !got.ordinaryReady {
			t.Fatal("controlled query changed source facts199")
		}
	}
	calls = runtime.NumCgoCall() - calls
	runtime.KeepAlive(ctx)
	if calls != 3073 {
		t.Fatalf("one-native-cleanup control requires3073 actual calls, got %d", calls)
	}
	t.Logf("IMPORT_PARENT199_CONTROLLED_NOISE actual_native_calls=%d native_budget=3072", calls)
	if calls > 3072 {
		t.Errorf("IMPORT_PARENT_DUPLICATE_NATIVE_READ199: calls %d exceed one-parent-per-edge guard budget 3072", calls)
	}
}

func importParent199CanonicalTranscript(parent string) superclass4059ProcessResult {
	var out strings.Builder
	fmt.Fprintf(&out, "=== RUN   %s\n", parent)
	for _, padding := range []int{0, 64, 512} {
		for _, anchor := range []string{"top", "nested", "root"} {
			depth := map[string]int{"top": 1, "nested": 3, "root": 0}[anchor]
			row := importParent199WorkRow{Queries: 1024, Padding: padding, Anchor: anchor, ParentEdges: depth, ActualNativeCalls: int64(1024 * (depth + 3)), NativeBudget: int64(1024 * (depth + 3)), OldSourceNativePrediction: 1024 * (2*depth + 3), RealExistingAPI: "importSourceFacts"}
			encoded, _ := json.Marshal(row)
			fmt.Fprintf(&out, "=== RUN   %s/padding%d/%s\n    probe.go:1: IMPORT_PARENT_NATIVE_WORK199 %s\n", parent, padding, anchor, encoded)
		}
	}
	fmt.Fprintf(&out, "--- PASS: %s (0.01s)\n", parent)
	for _, padding := range []int{0, 64, 512} {
		for _, anchor := range []string{"top", "nested", "root"} {
			fmt.Fprintf(&out, "    --- PASS: %s/padding%d/%s (0.00s)\n", parent, padding, anchor)
		}
	}
	out.WriteString("PASS\n")
	return superclass4059ProcessResult{Stdout: out.String(), Reaped: true}
}

func TestImportParentNativeProcess199Controls(t *testing.T) {
	if request, child := superclass4059ChildRequest(t); child {
		if request.Mode != "import-parent-noise-control199" {
			t.Fatal("invalid import-parent control mode199")
		}
		importParent199Contamination(t)
		return
	}
	parent := t.Name()
	t.Run("native-finalizer-red", func(t *testing.T) {
		result := superclass4059RunChild(t, superclass4059Request{parent, "import-parent-noise-control199"}, superclass4059ProcessOptions{sharedDir: superclass4059Flag("test.gocoverdir")})
		superclass4059ExpectFailure(t, result, "IMPORT_PARENT_DUPLICATE_NATIVE_READ199: calls 3073 exceed one-parent-per-edge guard budget 3072")
		if !strings.Contains(result.Stdout, `CONTROLLED_NATIVE_CLEANUP179 {"CloseInvocations":1,"NativeCalls":1}`) || !strings.Contains(result.Stdout, "IMPORT_PARENT199_CONTROLLED_NOISE actual_native_calls=3073 native_budget=3072") {
			t.Fatal("controlled one-native finalizer receipt missing199")
		}
	})
	t.Run("outer-cleanup-isolated-green", func(t *testing.T) {
		ready := filepath.Join(t.TempDir(), "child.ready")
		t.Setenv(importParent199OuterReady, ready)
		state := cgo179ArmCleanup(t)
		done := make(chan bool, 1)
		go func() {
			defer close(done)
			until := time.Now().Add(5 * time.Second)
			for time.Now().Before(until) {
				if _, err := os.Stat(ready); err == nil {
					cgo179ReleaseCleanup(t, state)
					if err := os.WriteFile(ready+".ack", []byte("one-native-cleanup-acknowledged\n"), 0600); err != nil {
						t.Error(err)
						return
					}
					done <- true
					return
				}
				time.Sleep(time.Millisecond)
			}
			t.Error("isolated child readiness timeout199")
		}()
		options := superclass4059ProcessOptions{sharedDir: superclass4059Flag("test.gocoverdir")}
		if testing.CoverMode() != "" {
			options.sharedDir = t.TempDir()
			options.textProfile = filepath.Join(t.TempDir(), "import.cover")
		}
		result := superclass4059RunChild(t, superclass4059Request{"TestImportParentNativeWork199", "import-parent-outer-cleanup199"}, options)
		superclass4059LogResult(t, result)
		if acknowledged := <-done; !acknowledged || state.invocations.Load() != 1 {
			t.Fatal("outer cleanup was not acknowledged exactly once199")
		}
		if err := importParent199Complete(result, "TestImportParentNativeWork199"); err != nil {
			t.Fatal(err)
		}
		if options.textProfile != "" {
			encoded, err := os.ReadFile(options.textProfile)
			if err != nil {
				t.Fatal(err)
			}
			positive := 0
			for _, line := range strings.Split(string(encoded), "\n") {
				fields := strings.Fields(line)
				if len(fields) == 3 && strings.Contains(fields[0], "/transpiler/import_source_inventory.go:") {
					count, err := strconv.ParseUint(fields[2], 10, 64)
					if err != nil {
						t.Fatal(err)
					}
					if count > 0 {
						positive++
					}
				}
			}
			if positive == 0 {
				t.Fatal("isolated child lost production import guard coverage199")
			}
			t.Logf("IMPORT_PARENT199_PRODUCTION_COVERAGE positive_blocks=%d", positive)
		}
	})
	t.Run("invalid-child-mode", func(t *testing.T) {
		result := superclass4059RunChild(t, superclass4059Request{"TestImportParentNativeWork199", "wrong-mode199"}, superclass4059ProcessOptions{})
		superclass4059ExpectFailure(t, result, "invalid import-parent measurement child mode199")
	})
	t.Run("complete-child-census", func(t *testing.T) {
		parent := "TestImportParentNativeWork199"
		good := importParent199CanonicalTranscript(parent)
		if err := importParent199Complete(good, parent); err != nil {
			t.Fatal(err)
		}
		mutants := map[string]superclass4059ProcessResult{}
		for name, text := range map[string]string{
			"unknown-run":        good.Stdout + "=== RUN   " + parent + "/foreign\n",
			"unknown-pass":       good.Stdout + "    --- PASS: " + parent + "/foreign (0.00s)\n",
			"duplicate-run":      good.Stdout + "=== RUN   " + parent + "/padding0/root\n",
			"duplicate-pass":     good.Stdout + "    --- PASS: " + parent + "/padding0/root (0.00s)\n",
			"missing-parent-run": strings.Replace(good.Stdout, "=== RUN   "+parent+"\n", "", 1),
		} {
			value := good
			value.Stdout = text
			mutants[name] = value
		}
		for name, text := range map[string]string{"missing-row": strings.Replace(good.Stdout, "IMPORT_PARENT_NATIVE_WORK199 ", "MISSING_ROW ", 1), "unknown-anchor": strings.Replace(good.Stdout, `"anchor":"root"`, `"anchor":"foreign"`, 1), "over-budget": strings.Replace(good.Stdout, `"actual_native_calls":3072`, `"actual_native_calls":3073`, 1), "changed-budget": strings.Replace(good.Stdout, `"native_budget":3072`, `"native_budget":3073`, 1), "wrong-query-count": strings.Replace(good.Stdout, `"queries":1024`, `"queries":1023`, 1), "missing-field": strings.Replace(good.Stdout, `"mallocs":0,`, "", 1), "null-field": strings.Replace(good.Stdout, `"mallocs":0`, `"mallocs":null`, 1), "missing-subtest": strings.Replace(good.Stdout, "    --- PASS: "+parent+"/padding0/root", "    --- LOST: "+parent+"/padding0/root", 1), "failure": good.Stdout + "--- FAIL: other (0.00s)\n", "skip": good.Stdout + "--- SKIP: other (0.00s)\n", "panic": good.Stdout + "panic: actual failure\n", "race": good.Stdout + "DATA RACE\n"} {
			value := good
			value.Stdout = text
			mutants[name] = value
		}
		first := regexp.MustCompile(`(?m)^.*IMPORT_PARENT_NATIVE_WORK199 .*$`).FindString(good.Stdout)
		duplicate := good
		duplicate.Stdout += first + "\n"
		mutants["duplicate-row"] = duplicate
		for name, edit := range map[string]func(*superclass4059ProcessResult){"stderr": func(r *superclass4059ProcessResult) { r.Stderr = "unexpected\n" }, "exit": func(r *superclass4059ProcessResult) { r.Error = "exit status 1" }, "timeout": func(r *superclass4059ProcessResult) { r.TimedOut = true }, "overflow": func(r *superclass4059ProcessResult) { r.Overflow = true }, "unreaped": func(r *superclass4059ProcessResult) { r.Reaped = false }} {
			value := good
			edit(&value)
			mutants[name] = value
		}
		for name, value := range mutants {
			t.Run(name, func(t *testing.T) {
				if importParent199Complete(value, parent) == nil {
					t.Fatal("incomplete or failed child was admitted199")
				}
			})
		}
	})
}
