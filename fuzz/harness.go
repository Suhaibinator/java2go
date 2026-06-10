package fuzz

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	java2go "github.com/NickyBoy89/java2go"
)

// Category classifies the outcome of running one program through both pipelines.
type Category string

const (
	// OK means Java and Go produced identical output — no divergence.
	OK Category = "OK"
	// TranspileCrash means java2go panicked or returned an error converting the
	// program (it should degrade gracefully, never crash).
	TranspileCrash Category = "TRANSPILE_CRASH"
	// GoCompileError means the transpiled Go did not compile.
	GoCompileError Category = "GO_COMPILE_ERROR"
	// OutputMismatch means both sides ran but printed different stdout — the
	// highest-value category (a real behavioral divergence).
	OutputMismatch Category = "OUTPUT_MISMATCH"
	// GoRuntimeError means the transpiled Go compiled but panicked/exited non-zero
	// at runtime while the Java side ran cleanly — a behavioral divergence (Java
	// produced a value, Go crashed).
	GoRuntimeError Category = "GO_RUNTIME_ERROR"
	// JavaError means the generated program is not valid Java (compile error or
	// runtime exception on the JDK). This is a generator bug, not a transpiler
	// bug; such programs are discarded, never reported.
	JavaError Category = "JAVA_ERROR"
)

// Result is the outcome of one differential run.
type Result struct {
	Seed     int64
	Source   string
	Category Category
	// JavaOut / GoOut hold the captured stdout of each side (when reached).
	JavaOut string
	GoOut   string
	// Detail carries the compiler/transpiler error text for the *_ERROR
	// categories, for triage.
	Detail string
}

// Diverged reports whether the result is a genuine transpiler divergence worth
// recording (i.e. not OK and not a generator bug).
func (r Result) Diverged() bool {
	switch r.Category {
	case TranspileCrash, GoCompileError, GoRuntimeError, OutputMismatch:
		return true
	}
	return false
}

// Harness runs programs through both the JDK and java2go inside a scratch dir
// that lives under the module root so the stdjava import resolves.
type Harness struct {
	root     string // module root (contains go.mod)
	buildDir string // scratch dir under root for generated Go + Java
	timeout  time.Duration
}

// NewHarness creates a harness rooted at the java2go module directory. buildDir
// is created (and reused) under root; it must be inside the module so the
// generated Go can import github.com/NickyBoy89/java2go/stdjava.
func NewHarness(root string) (*Harness, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		return nil, fmt.Errorf("module root %s has no go.mod: %w", root, err)
	}
	buildDir := filepath.Join(root, "fuzz", ".fuzzbuild")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return nil, err
	}
	return &Harness{root: root, buildDir: buildDir, timeout: 20 * time.Second}, nil
}

// Cleanup removes the scratch build directory.
func (h *Harness) Cleanup() {
	_ = os.RemoveAll(h.buildDir)
}

// Run executes one source program through both pipelines and classifies the
// outcome. The Java side is the oracle: if it does not compile and run cleanly,
// the program is a generator bug (JavaError) and the transpiler is not consulted.
func (h *Harness) Run(seed int64, source string) Result {
	res := Result{Seed: seed, Source: source}

	// Per-run scratch dir, isolated by seed so concurrent runs never collide.
	work := filepath.Join(h.buildDir, fmt.Sprintf("seed_%d", seed))
	if err := os.RemoveAll(work); err != nil {
		res.Category = TranspileCrash
		res.Detail = "clearing workdir: " + err.Error()
		return res
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		res.Category = TranspileCrash
		res.Detail = "mkdir workdir: " + err.Error()
		return res
	}
	defer os.RemoveAll(work)

	// 1. Oracle: run on the real JDK via single-file source launch.
	javaPath := filepath.Join(work, "Gen.java")
	if err := os.WriteFile(javaPath, []byte(source), 0o644); err != nil {
		res.Category = TranspileCrash
		res.Detail = "writing java: " + err.Error()
		return res
	}
	javaOut, javaErr := h.runCmd(work, "java", javaPath)
	if javaErr != nil {
		res.Category = JavaError
		res.Detail = javaErr.Error() + "\n" + javaOut
		return res
	}
	res.JavaOut = javaOut

	// 2. Transpile via the library API. A panic here is a transpiler crash, not a
	//    fatal harness error, so recover and categorize it.
	goOutDir := filepath.Join(work, "out")
	if err := h.transpile(javaPath, goOutDir); err != nil {
		res.Category = TranspileCrash
		res.Detail = err.Error()
		return res
	}

	// The transpiler renames main() to Main(); add a driver so the package runs.
	driver := "package main\n\nfunc main() { Main() }\n"
	if err := os.WriteFile(filepath.Join(goOutDir, "zz_driver.go"), []byte(driver), 0o644); err != nil {
		res.Category = TranspileCrash
		res.Detail = "writing driver: " + err.Error()
		return res
	}

	// 3. Compile the generated Go from the module root so imports resolve. Build
	//    and run are split so a compile failure (GO_COMPILE_ERROR) is distinguished
	//    from a clean-compile-but-panic-at-runtime (GO_RUNTIME_ERROR).
	rel, err := filepath.Rel(h.root, goOutDir)
	if err != nil {
		res.Category = GoCompileError
		res.Detail = err.Error()
		return res
	}
	pkgPath := "./" + filepath.ToSlash(rel) + "/"
	bin := filepath.Join(goOutDir, "prog")
	if buildOut, buildErr := h.runCmd(h.root, "go", "build", "-o", bin, pkgPath); buildErr != nil {
		res.Category = GoCompileError
		res.Detail = buildErr.Error() + "\n" + buildOut
		return res
	}

	// 4. Run the compiled binary.
	goOut, goErr := h.runCmd(h.root, bin)
	res.GoOut = goOut
	if goErr != nil {
		// Compiled fine but exited non-zero (panic, etc.) — a runtime divergence
		// since the Java oracle ran cleanly to completion.
		res.Category = GoRuntimeError
		res.Detail = goErr.Error() + "\n" + goOut
		return res
	}

	// 5. Diff stdout.
	if normalize(javaOut) != normalize(goOut) {
		res.Category = OutputMismatch
		return res
	}
	res.Category = OK
	return res
}

// transpile converts javaPath into Go under outDir, recovering panics so a
// transpiler crash is reported rather than killing the fuzzer.
func (h *Harness) transpile(javaPath, outDir string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("transpiler panic: %v", r)
		}
	}()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if runErr := java2go.Run([]string{"-w", "-output", outDir, javaPath}); runErr != nil {
		return fmt.Errorf("transpile error: %w", runErr)
	}
	return nil
}

// runCmd runs name with args in dir, capturing combined stdout+stderr, enforcing
// the harness timeout. A non-zero exit (or timeout) returns a non-nil error.
func (h *Harness) runCmd(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	done := make(chan struct{})
	var out []byte
	var runErr error
	go func() {
		out, runErr = cmd.CombinedOutput()
		close(done)
	}()
	select {
	case <-done:
		return string(out), runErr
	case <-time.After(h.timeout):
		_ = cmd.Process.Kill()
		<-done
		return string(out), errors.New("timeout after " + h.timeout.String())
	}
}

// normalize trims trailing whitespace/newlines so an oracle's final newline does
// not register as a spurious mismatch.
func normalize(s string) string {
	return strings.TrimRight(s, "\n \t")
}
