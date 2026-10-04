package transpiler

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Exercise Maven discovery, production conversion, package lowering and the
// generated process launcher using the original unnamed-package Java witness.
func TestMavenUnnamedGenericMainOriginalJVMParity(t *testing.T) {
	source, err := os.ReadFile("testdata/generic_legacy_main_witness/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	generated := filepath.Join(root, "generated")
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeProjectFile(filepath.Join(root, "pom.xml"), []byte(`<project><groupId>example</groupId><artifactId>generic</artifactId><version>1</version></project>`)); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(root, "src/main/java/Main.java")
	if err := writeProjectFile(input, source); err != nil {
		t.Fatal(err)
	}
	retainMavenUnnamedArtifacts(t, root, "maven-generic")
	var compilerOutput bytes.Buffer
	if err := run([]string{"-strict", "-maven", root, "-main-class", "Main", "-runtime", repo, "-module", "example.test/generic", "-output", generated}, &compilerOutput); err != nil {
		t.Fatalf("MAVEN_UNNAMED_GENERIC_MAIN_STAGE: production: %v; output=%s", err, compilerOutput.Bytes())
	}
	actual, err := os.ReadFile(input)
	if err != nil || !bytes.Equal(actual, source) {
		t.Fatalf("original Java changed: %v", err)
	}
	launcher, err := parser.ParseFile(token.NewFileSet(), filepath.Join(generated, "cmd/app/main.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	ast.Inspect(launcher, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		owner, ok := selector.X.(*ast.Ident)
		if ok && owner.Name == "app" {
			calls++
			if len(call.Args) != 0 {
				t.Fatal("generic process launcher must call zero-argument producer boundary")
			}
		}
		return true
	})
	if calls != 1 {
		t.Fatalf("expected one application process-boundary call, got %d", calls)
	}
	mavenUnnamedRun(t, generated, "generic-allpackages", 5*time.Minute, "go", "build", "-race", "-mod=mod", "./...")
	binary := filepath.Join(root, "generic-program")
	mavenUnnamedRun(t, generated, "generic-entry", 5*time.Minute, "go", "build", "-race", "-mod=mod", "-o", binary, "./cmd/app")
	for _, seed := range []string{"17", "41", "97"} {
		for repeat := 1; repeat <= 3; repeat++ {
			want, err := os.ReadFile("testdata/generic_legacy_main_witness/" + seed + ".stdout")
			if err != nil {
				t.Fatal(err)
			}
			wantErr, err := os.ReadFile("testdata/generic_legacy_main_witness/" + seed + ".stderr")
			if err != nil {
				t.Fatal(err)
			}
			got, stderr := mavenUnnamedRun(t, generated, fmt.Sprintf("maven-generic-%s-%d", seed, repeat), time.Minute, binary, seed)
			if !bytes.Equal(got, want) || !bytes.Equal(stderr, wantErr) {
				t.Fatalf("generic Maven seed%s repeat%d stdout=%q stderr=%q differs from original JVM stdout=%q stderr=%q", seed, repeat, got, stderr, want, wantErr)
			}
			t.Logf("MAVEN_UNNAMED_GENERIC_MAIN_ORIGINAL_JVM_PARITY: seed%s repeat%d stdout=%q stderr=%q", seed, repeat, got, stderr)
		}
	}
}

func TestMavenUnnamedPackageOrdinaryMainAndImports(t *testing.T) {
	root := t.TempDir()
	generated := filepath.Join(root, "generated")
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"pom.xml":                         `<project><groupId>example</groupId><artifactId>unnamed</artifactId><version>1</version></project>`,
		"src/main/java/Main.java":         `import probe.Helper; public class Main { public static void main(String[] args) { System.out.println(args.length + ":" + args[0] + ":" + Local.value() + ":" + Helper.value()); } }`,
		"src/main/java/other/Local.java":  `class Local { static int value() { return 17; } }`,
		"src/main/java/probe/Helper.java": `package probe; public class Helper { public static int value() { return 41; } }`,
	}
	for path, data := range files {
		if err := writeProjectFile(filepath.Join(root, path), []byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	retainMavenUnnamedArtifacts(t, root, "maven-ordinary")
	var output bytes.Buffer
	if err := run([]string{"-strict", "-maven", root, "-main-class", "Main", "-runtime", repo, "-output", generated}, &output); err != nil {
		t.Fatalf("unnamed ordinary Maven production: %v output=%s", err, output.Bytes())
	}
	binary := filepath.Join(root, "ordinary-program")
	mavenUnnamedRun(t, generated, "ordinary-allpackages", 5*time.Minute, "go", "build", "-race", "-mod=mod", "./...")
	mavenUnnamedRun(t, generated, "ordinary-entry", 5*time.Minute, "go", "build", "-race", "-mod=mod", "-o", binary, "./cmd/app")
	got, stderr := mavenUnnamedRun(t, generated, "ordinary-run", time.Minute, binary, "argument")
	if !bytes.Equal(got, []byte("1:argument:17:41\n")) || len(stderr) != 0 {
		t.Fatalf("ordinary main/import/unnamed sibling semantics stdout=%q stderr=%q", got, stderr)
	}
}

func TestMavenUnnamedPackageDuplicateAndInvalidMain(t *testing.T) {
	for _, test := range []struct {
		name  string
		files map[string]string
		main  string
		want  string
	}{
		{"duplicate", map[string]string{"Main.java": `public class Main { public static void main(String[] args) {} }`, "sub/Main.java": `public class Main {}`}, "Main", "duplicate Java type Main"},
		{"invalid-main", map[string]string{"Main.java": `public class Main { public static int value() { return 1; } }`}, "Main", "must declare public static void main"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			repo, err := filepath.Abs("..")
			if err != nil {
				t.Fatal(err)
			}
			if err := writeProjectFile(filepath.Join(root, "pom.xml"), []byte(`<project><groupId>example</groupId><artifactId>unnamed</artifactId><version>1</version></project>`)); err != nil {
				t.Fatal(err)
			}
			for path, data := range test.files {
				if err := writeProjectFile(filepath.Join(root, "src/main/java", path), []byte(data)); err != nil {
					t.Fatal(err)
				}
			}
			output := filepath.Join(root, "out")
			err = run([]string{"-strict", "-maven", root, "-main-class", test.main, "-runtime", repo, "-output", output}, &bytes.Buffer{})
			if err == nil || !bytes.Contains([]byte(err.Error()), []byte(test.want)) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatal("failed unnamed application published output")
			}
		})
	}
}

func retainMavenUnnamedArtifacts(t *testing.T, root, label string) {
	t.Helper()
	artifact := os.Getenv("JAVA2GO_NORMALIZER_ARTIFACTS")
	if artifact == "" {
		return
	}
	t.Cleanup(func() {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
			if e != nil || d.IsDir() {
				return e
			}
			ext := filepath.Ext(path)
			if ext != ".go" && ext != ".java" && filepath.Base(path) != "go.mod" && filepath.Base(path) != "pom.xml" {
				return nil
			}
			relative, e := filepath.Rel(root, path)
			if e != nil {
				return e
			}
			raw, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			return writeProjectFile(filepath.Join(artifact, label, relative), raw)
		})
		if err != nil {
			t.Error(err)
		}
	})
}
func mavenUnnamedRun(t *testing.T, dir, stage string, bound time.Duration, name string, args ...string) ([]byte, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), bound)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	cmd.WaitDelay = 5 * time.Second
	var out, errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	err := cmd.Run()
	if artifact := os.Getenv("JAVA2GO_NORMALIZER_ARTIFACTS"); artifact != "" {
		for _, stream := range []struct {
			name string
			data []byte
		}{{"stdout", out.Bytes()}, {"stderr", errout.Bytes()}} {
			if err := writeProjectFile(filepath.Join(artifact, stage+"."+stream.name), stream.data); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err != nil || ctx.Err() != nil {
		t.Fatalf("MAVEN_UNNAMED_STAGE: %s: %v timeout=%v stdout=%s stderr=%s", stage, err, ctx.Err(), out.Bytes(), errout.Bytes())
	}
	return out.Bytes(), errout.Bytes()
}
