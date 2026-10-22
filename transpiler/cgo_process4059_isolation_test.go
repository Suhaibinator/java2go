package transpiler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sitter "github.com/smacker/go-tree-sitter"
)

const superclass4059ProcessMarker = "JAVA2GO_SUPERCLASS4059_MEASUREMENT_CHILD"
const superclass4059OutputBound = 4 * 1024 * 1024

type superclass4059Request struct {
	Parent string
	Mode   string
}

type superclass4059ProcessOptions struct {
	limit       time.Duration
	sharedDir   string
	textProfile string
	marker      string
	withoutMark bool
	outputBound int
}

type superclass4059ProcessResult struct {
	Stdout, Stderr string
	Error          string
	PID            int
	Reaped         bool
	TimedOut       bool
	Overflow       bool
}

type superclass4059BoundedOutput struct {
	mu       sync.Mutex
	buffer   bytes.Buffer
	limit    int
	overflow bool
	cancel   context.CancelFunc
}

func (output *superclass4059BoundedOutput) Write(p []byte) (int, error) {
	output.mu.Lock()
	defer output.mu.Unlock()
	available := output.limit - output.buffer.Len()
	if len(p) > available {
		output.buffer.Write(p[:available])
		output.overflow = true
		output.cancel()
	} else {
		output.buffer.Write(p)
	}
	return len(p), nil
}

func (output *superclass4059BoundedOutput) result() (string, bool) {
	output.mu.Lock()
	defer output.mu.Unlock()
	return output.buffer.String(), output.overflow
}

func superclass4059Flag(name string) string {
	if value := flag.Lookup(name); value != nil {
		return value.Value.String()
	}
	return ""
}

func superclass4059ChildRequest(t *testing.T) (superclass4059Request, bool) {
	t.Helper()
	encoded := os.Getenv(superclass4059ProcessMarker)
	if encoded == "" {
		return superclass4059Request{}, false
	}
	var request superclass4059Request
	if err := json.Unmarshal([]byte(encoded), &request); err != nil || request.Parent != t.Name() || request.Mode == "" ||
		superclass4059Flag("test.run") != "^"+regexp.QuoteMeta(request.Parent)+"$" ||
		superclass4059Flag("test.count") != "1" || superclass4059Flag("test.parallel") != "1" {
		t.Fatal("invalid measurement child marker or selector")
	}
	return request, true
}

func superclass4059CanIsolate(coverMode, sharedDir string) bool {
	return coverMode == "" || sharedDir != ""
}

func superclass4059RunChild(t *testing.T, request superclass4059Request, options superclass4059ProcessOptions) superclass4059ProcessResult {
	t.Helper()
	limit := 55 * time.Second
	if options.limit > 0 && options.limit < limit {
		limit = options.limit
	}
	if deadline, ok := t.Deadline(); ok {
		remaining := time.Until(deadline) - 250*time.Millisecond
		if remaining < limit {
			limit = remaining
		}
	}
	if limit <= 0 {
		return superclass4059ProcessResult{Error: "no parent deadline remains for measurement child"}
	}
	executable, err := os.Executable()
	if err != nil {
		return superclass4059ProcessResult{Error: err.Error()}
	}
	args := []string{"-test.v", "-test.run=^" + regexp.QuoteMeta(request.Parent) + "$", "-test.count=1", "-test.parallel=1", "-test.timeout=" + limit.String()}
	if options.sharedDir != "" {
		args = append(args, "-test.gocoverdir="+options.sharedDir)
	}
	// Measurement children never inherit the parent's text coverage filename.
	// Only explicit coverage-contract control processes request a separate file.
	if options.textProfile != "" {
		args = append(args, "-test.coverprofile="+options.textProfile)
	}
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	command := exec.CommandContext(ctx, executable, args...)
	command.WaitDelay = 250 * time.Millisecond
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, superclass4059ProcessMarker+"=") {
			command.Env = append(command.Env, item)
		}
	}
	if !options.withoutMark {
		encoded, marshalError := json.Marshal(request)
		if marshalError != nil {
			return superclass4059ProcessResult{Error: marshalError.Error()}
		}
		marker := string(encoded)
		if options.marker != "" {
			marker = options.marker
		}
		command.Env = append(command.Env, superclass4059ProcessMarker+"="+marker)
	}
	bound := options.outputBound
	if bound <= 0 {
		bound = superclass4059OutputBound
	}
	stdout := &superclass4059BoundedOutput{limit: bound, cancel: cancel}
	stderr := &superclass4059BoundedOutput{limit: bound, cancel: cancel}
	command.Stdout, command.Stderr = stdout, stderr
	runError := command.Run()
	result := superclass4059ProcessResult{Reaped: command.ProcessState != nil, TimedOut: ctx.Err() == context.DeadlineExceeded}
	if command.Process != nil {
		result.PID = command.Process.Pid
	}
	result.Stdout, result.Overflow = stdout.result()
	var stderrOverflow bool
	result.Stderr, stderrOverflow = stderr.result()
	result.Overflow = result.Overflow || stderrOverflow
	if runError != nil {
		result.Error = runError.Error()
	}
	return result
}

func superclass4059LogResult(t *testing.T, result superclass4059ProcessResult) {
	t.Helper()
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	// JSON escapes child failure lines, retaining complete bounded evidence
	// without presenting controlled child failures as outer test failures.
	t.Logf("MEASUREMENT_CHILD4059 %s", encoded)
}

func superclass4059Completed(result superclass4059ProcessResult, parent string) error {
	if result.Error != "" || result.TimedOut || result.Overflow || !result.Reaped {
		return fmt.Errorf("measurement child failed: %s timeout=%t overflow=%t reaped=%t", result.Error, result.TimedOut, result.Overflow, result.Reaped)
	}
	if strings.Contains(result.Stdout+result.Stderr, "--- FAIL:") || strings.Contains(result.Stdout+result.Stderr, "--- SKIP:") || strings.Contains(result.Stdout+result.Stderr, "panic:") || strings.Contains(result.Stdout+result.Stderr, "DATA RACE") {
		return fmt.Errorf("measurement child reported a failed or incomplete assertion")
	}
	completed := regexp.MustCompile(`(?m)^--- PASS: ` + regexp.QuoteMeta(parent) + ` \(`)
	if len(completed.FindAllString(result.Stdout, -1)) != 1 {
		return fmt.Errorf("measurement child lacks exactly one completed parent")
	}
	return nil
}

func superclass4059CompleteWork(result superclass4059ProcessResult, parent string) error {
	if err := superclass4059Completed(result, parent); err != nil {
		return err
	}
	cases := map[string]bool{"m1-b4-f1-nfalse": true, "m16-b4-f1-nfalse": true, "m64-b4-f1-nfalse": true, "m1-b24-f1-nfalse": true, "m16-b24-f1-nfalse": true, "m64-b24-f1-nfalse": true, "m64-b24-f0-nfalse": true, "m16-b4-f1-ntrue": true}
	rows := map[string]bool{}
	for _, line := range strings.Split(result.Stdout, "\n") {
		if _, encoded, found := strings.Cut(line, "SUPERCLASS_QUERY_WORK "); found {
			var row superclassQuery4059WorkRow
			if err := json.Unmarshal([]byte(encoded), &row); err != nil {
				return fmt.Errorf("invalid measurement row: %w", err)
			}
			key := fmt.Sprintf("m%d-b%d-f%d-n%t", row.Methods, row.Statements, row.Formals, row.Nested)
			identity := row.Label + "/" + key
			if !cases[key] || (row.Label != "scanner" && row.Label != "allocator") || rows[identity] || row.Output == "" {
				return fmt.Errorf("unexpected or duplicate measurement row %s", identity)
			}
			rows[identity] = true
		}
	}
	if len(rows) != 16 {
		return fmt.Errorf("measurement child completed %d of16 required work rows", len(rows))
	}
	for name := range cases {
		pattern := regexp.MustCompile(`(?m)^    --- PASS: ` + regexp.QuoteMeta(parent+"/"+name) + ` \(`)
		if len(pattern.FindAllString(result.Stdout, -1)) != 1 {
			return fmt.Errorf("measurement child lacks completed fixture %s", name)
		}
	}
	return nil
}

func superclass4059IsolatedEntry(t *testing.T) bool {
	t.Helper()
	if request, child := superclass4059ChildRequest(t); child {
		if request.Mode != "work" {
			t.Fatal("invalid measurement child mode")
		}
		return false
	}
	sharedDir := superclass4059Flag("test.gocoverdir")
	if !superclass4059CanIsolate(testing.CoverMode(), sharedDir) {
		t.Log("MEASUREMENT4059_UNISOLATED_COVERAGE_FALLBACK: original in-process assertions retained")
		return false
	}
	result := superclass4059RunChild(t, superclass4059Request{t.Name(), "work"}, superclass4059ProcessOptions{sharedDir: sharedDir})
	superclass4059LogResult(t, result)
	if err := superclass4059CompleteWork(result, t.Name()); err != nil {
		t.Fatal(err)
	}
	return true
}

// Controls dispatch only inside a marked child of the same real parent. Each
// normally registered parent below executes actual checks, never a skip/no-op.
func superclass4059ControlChild(t *testing.T) bool {
	t.Helper()
	request, child := superclass4059ChildRequest(t)
	if !child {
		return false
	}
	switch request.Mode {
	case "stable", "malformed":
		t.Logf("MEASUREMENT_CONTROL4059_OK pid=%d", os.Getpid())
	case "failure":
		t.Fatal("controlled real child assertion failure4059")
	case "timeout":
		time.Sleep(5 * time.Second)
		t.Fatal("timeout control was not stopped")
	case "overflow":
		t.Log(strings.Repeat("x", 8192))
	case "ownership":
		cgo179OwnershipFixture(t)
	case "noise":
		cgo179NoiseFixture(t)
	case "budget":
		cgo179BudgetFixture(t)
	case "coverage", "coverage-direct":
		superclass4059ProductionCoverageProbe(t)
	case "coverage-parent":
		if superclass4059Flag("test.coverprofile") == "" || superclass4059Flag("test.gocoverdir") == "" {
			t.Fatal("coverage parent has no explicit aggregation ownership")
		}
		result := superclass4059RunChild(t, superclass4059Request{t.Name(), "coverage"}, superclass4059ProcessOptions{sharedDir: superclass4059Flag("test.gocoverdir")})
		superclass4059LogResult(t, result)
		if err := superclass4059Completed(result, t.Name()); err != nil {
			t.Fatal(err)
		}
	case "fallback-noise":
		if err := os.Unsetenv(superclass4059ProcessMarker); err != nil {
			t.Fatal(err)
		}
		if superclass4059IsolatedEntry(t) {
			t.Fatal("manual empty-directory coverage unexpectedly isolated")
		}
		cgo179NoiseFixture(t)
	default:
		t.Fatal("unknown measurement control child mode")
	}
	return true
}

func superclass4059ExpectFailure(t *testing.T, result superclass4059ProcessResult, diagnostic string) {
	t.Helper()
	superclass4059LogResult(t, result)
	if result.Error == "" || !result.Reaped || result.TimedOut || result.Overflow || !strings.Contains(result.Stdout+result.Stderr, diagnostic) {
		t.Fatalf("expected real bounded child assertion failure %q", diagnostic)
	}
}

func TestSuperclassQuery4059ProcessLifecycle(t *testing.T) {
	if superclass4059ControlChild(t) {
		return
	}
	result := superclass4059RunChild(t, superclass4059Request{t.Name(), "stable"}, superclass4059ProcessOptions{sharedDir: superclass4059Flag("test.gocoverdir")})
	superclass4059LogResult(t, result)
	if err := superclass4059Completed(result, t.Name()); err != nil {
		t.Fatal(err)
	}
	if result.PID == os.Getpid() || !strings.Contains(result.Stdout, fmt.Sprintf("pid=%d", result.PID)) || os.Getenv(superclass4059ProcessMarker) != "" {
		t.Fatal("measurement child was not a separate reaped invocation")
	}
}

func TestSuperclassQuery4059ProcessChildFailure(t *testing.T) {
	if superclass4059ControlChild(t) {
		return
	}
	result := superclass4059RunChild(t, superclass4059Request{t.Name(), "failure"}, superclass4059ProcessOptions{sharedDir: superclass4059Flag("test.gocoverdir")})
	superclass4059ExpectFailure(t, result, "controlled real child assertion failure4059")
	if superclass4059Completed(result, t.Name()) == nil {
		t.Fatal("child assertion failure was admitted")
	}
}

func TestSuperclassQuery4059ProcessTimeout(t *testing.T) {
	if superclass4059ControlChild(t) {
		return
	}
	result := superclass4059RunChild(t, superclass4059Request{t.Name(), "timeout"}, superclass4059ProcessOptions{limit: 100 * time.Millisecond})
	superclass4059LogResult(t, result)
	if result.Error == "" || !result.Reaped || !result.TimedOut || superclass4059Completed(result, t.Name()) == nil {
		t.Fatal("child timeout did not fail closed and reap")
	}
}

func TestSuperclassQuery4059ProcessIncompleteOutput(t *testing.T) {
	if superclass4059ControlChild(t) {
		return
	}
	result := superclass4059RunChild(t, superclass4059Request{t.Name(), "malformed"}, superclass4059ProcessOptions{sharedDir: superclass4059Flag("test.gocoverdir")})
	superclass4059LogResult(t, result)
	if err := superclass4059Completed(result, t.Name()); err != nil {
		t.Fatal(err)
	}
	if superclass4059CompleteWork(result, t.Name()) == nil {
		t.Fatal("successful child without complete actual fixtures was admitted")
	}
	result.Stdout += "\nSUPERCLASS_QUERY_WORK {invalid JSON}\n"
	if superclass4059CompleteWork(result, t.Name()) == nil {
		t.Fatal("malformed measurement row was admitted")
	}
}

func TestSuperclassQuery4059ProcessOutputBound(t *testing.T) {
	if superclass4059ControlChild(t) {
		return
	}
	result := superclass4059RunChild(t, superclass4059Request{t.Name(), "overflow"}, superclass4059ProcessOptions{outputBound: 128})
	superclass4059LogResult(t, result)
	if !result.Overflow || !result.Reaped || superclass4059Completed(result, t.Name()) == nil {
		t.Fatal("truncated child output was admitted or child not reaped")
	}
}

func TestSuperclassQuery4059ProcessMarker(t *testing.T) {
	if superclass4059ControlChild(t) {
		return
	}
	result := superclass4059RunChild(t, superclass4059Request{t.Name(), "stable"}, superclass4059ProcessOptions{marker: `{"Parent":"different-parent","Mode":"stable"}`})
	superclass4059ExpectFailure(t, result, "invalid measurement child marker or selector")
}

func TestSuperclassQuery4059ProcessNativeNoise(t *testing.T) {
	if superclass4059ControlChild(t) {
		return
	}
	for _, mode := range []string{"ownership", "noise"} {
		result := superclass4059RunChild(t, superclass4059Request{t.Name(), mode}, superclass4059ProcessOptions{sharedDir: superclass4059Flag("test.gocoverdir")})
		if !strings.Contains(result.Stdout, `CONTROLLED_NATIVE_CLEANUP179 {"CloseInvocations":1,"NativeCalls":1}`) {
			t.Fatal("native cleanup was not actually observed once")
		}
		if mode == "ownership" {
			superclass4059LogResult(t, result)
			if err := superclass4059Completed(result, t.Name()); err != nil {
				t.Fatal(err)
			}
		} else {
			superclass4059ExpectFailure(t, result, "unexpected native finalizer/concurrent noise: delta=1")
		}
	}
}

func TestSuperclassQuery4059ProcessBudgetFailure(t *testing.T) {
	if superclass4059ControlChild(t) {
		return
	}
	result := superclass4059RunChild(t, superclass4059Request{t.Name(), "budget"}, superclass4059ProcessOptions{sharedDir: superclass4059Flag("test.gocoverdir")})
	superclass4059ExpectFailure(t, result, "ACTUAL_NATIVE_WORK_RED:")
	if !strings.Contains(result.Stdout, "ACTUAL_ALLOCATION_WORK_RED:") || superclass4059CompleteWork(result, t.Name()) == nil {
		t.Fatal("measured old scanner budget failure lost its allocation shield or was admitted")
	}
}

func superclass4059ProductionCoverageProbe(t *testing.T) {
	t.Helper()
	for _, name := range []string{"type", "ordinary"} {
		want := name
		if name == "type" {
			want += "_"
		}
		if actual := sanitizeGoIdent(name); actual != want {
			t.Fatalf("real production identifier behavior changed: %q", actual)
		}
	}
	t.Log("MEASUREMENT_PRODUCTION_COVERAGE4059_OK")
}

func superclass4059SanitizerCoverage(t *testing.T, profile string) map[string]uint64 {
	t.Helper()
	files := token.NewFileSet()
	parsed, err := parser.ParseFile(files, "imports.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	first, last := 0, 0
	for _, declaration := range parsed.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == "sanitizeGoIdent" {
			first, last = files.Position(function.Pos()).Line, files.Position(function.End()).Line
		}
	}
	if first == 0 {
		t.Fatal("actual production function not found")
	}
	encoded, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	positive := map[string]uint64{}
	for _, line := range strings.Split(string(encoded), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		path, position, found := strings.Cut(fields[0], ":")
		if !found || !strings.HasSuffix(path, "/transpiler/imports.go") {
			continue
		}
		start, _, found := strings.Cut(position, ".")
		lineNumber, parseError := strconv.Atoi(start)
		if !found || parseError != nil || lineNumber < first || lineNumber > last {
			continue
		}
		count, parseError := strconv.ParseUint(fields[2], 10, 64)
		if parseError != nil || count == 0 {
			t.Fatalf("production block %s lacks positive coverage", position)
		}
		positive[position+" "+fields[1]] = count
	}
	if len(positive) == 0 {
		t.Fatal("no actual production function coverage blocks found")
	}
	return positive
}

func TestSuperclassQuery4059ProcessCoverage(t *testing.T) {
	if superclass4059ControlChild(t) {
		return
	}
	if testing.CoverMode() == "" {
		if !superclass4059CanIsolate("", "") {
			t.Fatal("non-covered process unexpectedly requires a coverage directory")
		}
		result := superclass4059RunChild(t, superclass4059Request{t.Name(), "stable"}, superclass4059ProcessOptions{})
		if err := superclass4059Completed(result, t.Name()); err != nil {
			t.Fatal(err)
		}
		superclass4059LogResult(t, result)
		return
	}
	profiles := map[string]map[string]uint64{}
	metadata := map[string]string{}
	counterFiles := map[string]int{}
	profileHashes := map[string]string{}
	for _, mode := range []string{"coverage-parent", "coverage-direct"} {
		directory := t.TempDir()
		profile := filepath.Join(t.TempDir(), mode+".cover")
		result := superclass4059RunChild(t, superclass4059Request{t.Name(), mode}, superclass4059ProcessOptions{sharedDir: directory, textProfile: profile})
		superclass4059LogResult(t, result)
		if err := superclass4059Completed(result, t.Name()); err != nil {
			t.Fatal(err)
		}
		profiles[mode] = superclass4059SanitizerCoverage(t, profile)
		profileBytes, readError := os.ReadFile(profile)
		if readError != nil {
			t.Fatal(readError)
		}
		profileHashes[mode] = fmt.Sprintf("%x", sha256.Sum256(profileBytes))
		entries, readError := os.ReadDir(directory)
		if readError != nil {
			t.Fatal(readError)
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "covmeta.") {
				if metadata[mode] != "" {
					t.Fatal("multiple binary metadata hashes in coverage directory")
				}
				metadata[mode] = entry.Name()
			}
			if strings.HasPrefix(entry.Name(), "covcounters.") {
				counterFiles[mode]++
			}
		}
	}
	for block, directCount := range profiles["coverage-direct"] {
		if count := profiles["coverage-parent"][block]; count != directCount {
			t.Fatalf("shared same-binary child block %s count=%d differs from direct in-process control %d", block, count, directCount)
		}
	}
	if len(profiles["coverage-parent"]) != len(profiles["coverage-direct"]) {
		t.Fatal("shared same-binary child production block inventory differs")
	}
	if metadata["coverage-parent"] == "" || metadata["coverage-parent"] != metadata["coverage-direct"] || counterFiles["coverage-parent"] < 2 || counterFiles["coverage-direct"] != 1 {
		t.Fatalf("coverage did not aggregate same-hash child and parent counters: meta=%v counters=%v", metadata, counterFiles)
	}
	encoded, err := json.Marshal(struct {
		Blocks        map[string]map[string]uint64
		Metadata      map[string]string
		CounterFiles  map[string]int
		ProfileSHA256 map[string]string
	}{profiles, metadata, counterFiles, profileHashes})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ACTUAL_PRODUCTION_COVERAGE4059 %s", encoded)
}

func TestSuperclassQuery4059ProcessManualCoveredFallback(t *testing.T) {
	if superclass4059ControlChild(t) {
		return
	}
	if testing.CoverMode() == "" {
		result := superclass4059RunChild(t, superclass4059Request{t.Name(), "stable"}, superclass4059ProcessOptions{})
		if err := superclass4059Completed(result, t.Name()); err != nil {
			t.Fatal(err)
		}
		superclass4059LogResult(t, result)
		return
	}
	parent := "TestSuperclassQuery4059FrozenWorkBudget"
	result := superclass4059RunChild(t, superclass4059Request{parent, "work"}, superclass4059ProcessOptions{withoutMark: true, textProfile: filepath.Join(t.TempDir(), "manual.cover")})
	superclass4059LogResult(t, result)
	if err := superclass4059CompleteWork(result, parent); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Stdout, "MEASUREMENT4059_UNISOLATED_COVERAGE_FALLBACK") {
		t.Fatal("covered empty-directory manual invocation did not retain original body")
	}
	failed := superclass4059RunChild(t, superclass4059Request{t.Name(), "fallback-noise"}, superclass4059ProcessOptions{textProfile: filepath.Join(t.TempDir(), "manual-failure.cover")})
	superclass4059ExpectFailure(t, failed, "unexpected native finalizer/concurrent noise: delta=1")
	if !strings.Contains(failed.Stdout, "MEASUREMENT4059_UNISOLATED_COVERAGE_FALLBACK") || !strings.Contains(failed.Stdout, `CONTROLLED_NATIVE_CLEANUP179 {"CloseInvocations":1,"NativeCalls":1}`) {
		t.Fatal("manual fallback did not preserve actual noisy old-helper failure")
	}
}

type cgo179CleanupState struct {
	invocations atomic.Int32
	done        chan cgo179CleanupReceipt
}

type cgo179CleanupReceipt struct {
	CloseInvocations int32
	NativeCalls      int64
}

type cgo179NativeHolder struct {
	parser *sitter.Parser
	state  *cgo179CleanupState
}

// A package root keeps ownership explicit until the controlled release. No
// closure relies on compiler-local liveness to keep the native parser alive.
var cgo179CleanupRoot *cgo179NativeHolder

func cgo179ArmCleanup(t *testing.T) *cgo179CleanupState {
	t.Helper()
	if cgo179CleanupRoot != nil {
		t.Fatal("previous controlled native owner is still registered")
	}
	parser := sitter.NewParser()
	// Transfer finalizer ownership before the holder owns Close. The library's
	// original parser finalizer must not race this holder's cleanup.
	runtime.SetFinalizer(parser, nil)
	state := &cgo179CleanupState{done: make(chan cgo179CleanupReceipt, 1)}
	holder := &cgo179NativeHolder{parser: parser, state: state}
	runtime.SetFinalizer(holder, func(value *cgo179NativeHolder) {
		invocations := value.state.invocations.Add(1)
		if invocations != 1 {
			value.state.done <- cgo179CleanupReceipt{CloseInvocations: invocations, NativeCalls: -1}
			return
		}
		before := runtime.NumCgoCall()
		value.parser.Close()
		delta := runtime.NumCgoCall() - before
		value.state.done <- cgo179CleanupReceipt{CloseInvocations: invocations, NativeCalls: delta}
	})
	cgo179CleanupRoot = holder
	return state
}

func cgo179ReleaseCleanup(t *testing.T, state *cgo179CleanupState) cgo179CleanupReceipt {
	t.Helper()
	if cgo179CleanupRoot == nil || cgo179CleanupRoot.state != state {
		t.Fatal("controlled native owner was lost before explicit release")
	}
	cgo179CleanupRoot = nil
	// Forced collection belongs only to this controlled contamination fixture;
	// no production query or original measurement helper is altered.
	runtime.GC()
	select {
	case receipt := <-state.done:
		if receipt.CloseInvocations != 1 || receipt.NativeCalls != 1 || state.invocations.Load() != 1 {
			t.Fatalf("native cleanup prerequisite failed: %+v", receipt)
		}
		encoded, err := json.Marshal(receipt)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("CONTROLLED_NATIVE_CLEANUP179 %s", encoded)
		return receipt
	case <-time.After(2 * time.Second):
		t.Fatal("controlled native cleanup did not complete")
		return cgo179CleanupReceipt{}
	}
}

func cgo179OwnershipFixture(t *testing.T) {
	state := cgo179ArmCleanup(t)
	cgo179ReleaseCleanup(t, state)
	// The parser's old finalizer was removed before holder registration. Further
	// collection cannot invoke holder Close a second time.
	runtime.GC()
	runtime.GC()
	if state.invocations.Load() != 1 {
		t.Fatal("native Close was invoked more than once")
	}
	runtime.KeepAlive(state)
}

func cgo179NoiseFixture(t *testing.T) {
	state := cgo179ArmCleanup(t)
	calls := 0
	query := func() string {
		calls++
		if calls == 4 {
			// The unchanged helper calls once for output and twice for warmup.
			// Release native cleanup in the first of its three measured calls.
			cgo179ReleaseCleanup(t, state)
		}
		return "stable-output-without-native-query-work"
	}
	// Expected raw RED is accepted only with observed single native cleanup and
	// the unchanged helper's actual nondivisible counter diagnostic.
	superclassQuery4059Measure(t, query)
	t.Fatal("old measurement unexpectedly accepted controlled native cleanup")
}

func cgo179BudgetFixture(t *testing.T) {
	fixture := superclassQuery4059CostFixture(t, 1, 4, 1, false)
	for _, file := range fixture.files {
		_ = superclassQuery4059Nodes(file.Ast, "identifier")
	}
	allocs, cgo, output := superclassQuery4059Measure(t, func() string {
		if superclassQuery4059OldOracle("UnrelatedAllocatedEntry", fixture.ctx) {
			panic("old scanner reserved an unrelated name")
		}
		return "absent"
	})
	// These are the unchanged existing scanner/m1/b4/f1/nfalse limits from
	// TestSuperclassQuery4059FrozenWorkBudget, not a new or relaxed budget.
	const frozenAllocs, frozenCgo = 268, 411
	t.Logf("OLD_SCANNER_MEASURED179 allocs=%.0f cgo=%d output=%s", allocs, cgo, output)
	if allocs <= frozenAllocs || cgo <= frozenCgo {
		t.Fatalf("old scanner shield prerequisite failed: allocs=%.0f cgo=%d", allocs, cgo)
	}
	t.Errorf("ACTUAL_ALLOCATION_WORK_RED: scanner/m1/b4/f1/nfalse actual=%.0f frozen_max=%d", allocs, frozenAllocs)
	t.Errorf("ACTUAL_NATIVE_WORK_RED: scanner/m1/b4/f1/nfalse actual=%d frozen_max=%d", cgo, frozenCgo)
	runtime.KeepAlive(fixture.files)
}
