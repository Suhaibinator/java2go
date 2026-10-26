package transpiler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
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

// The original JVM application covers literal String and Locale.US calls.
// These compiler controls exercise typed/null arguments and owner isolation at
// the same boundary, without adding invented Java behavioral expectations.
func dateFormatConstructorCalls46(t *testing.T, class, source string) []*ast.CallExpr {
	t.Helper()
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	input := filepath.Join(work, class+".java")
	emptyInput := filepath.Join(work, "empty-input-path")
	classes := filepath.Join(work, "classes")
	if err := os.WriteFile(input, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(emptyInput, 0755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, javac, "--release", "21", "-encoding", "UTF-8", "-proc:none", "-classpath", emptyInput, "-sourcepath", emptyInput, "-d", classes, input)
	command.Dir = work
	command.WaitDelay = 5 * time.Second
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	started := time.Now()
	runErr := command.Run()
	exit := 0
	if runErr != nil {
		exit = -1
		if failure, ok := runErr.(*exec.ExitError); ok {
			exit = failure.ExitCode()
		}
	}
	if artifacts := os.Getenv("JAVA2GO_DATEFORMAT_CTOR_ARTIFACTS"); artifacts != "" {
		directory := filepath.Join(artifacts, class)
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
		for name, raw := range map[string][]byte{class + ".java": []byte(source), "javac.stdout": stdout.Bytes(), "javac.stderr": stderr.Bytes()} {
			if err := os.WriteFile(filepath.Join(directory, name), raw, 0644); err != nil {
				t.Fatal(err)
			}
		}
		record, err := json.MarshalIndent(map[string]any{"stage": "constructor-control-javac", "command": command.Args, "cwd": work, "exit_code": exit, "timed_out": ctx.Err() != nil, "elapsed_seconds": time.Since(started).Seconds(), "source_sha256": fmt.Sprintf("%x", sha256.Sum256([]byte(source)))}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "javac.json"), append(record, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
		hashes := map[string]string{}
		if runErr == nil {
			if err := filepath.WalkDir(classes, func(path string, item os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if item.IsDir() {
					return nil
				}
				relative, err := filepath.Rel(classes, path)
				if err != nil {
					return err
				}
				raw, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				hashes[relative] = fmt.Sprintf("%x", sha256.Sum256(raw))
				target := filepath.Join(directory, "classes", relative)
				if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
					return err
				}
				return os.WriteFile(target, raw, 0644)
			}); err != nil {
				t.Fatal(err)
			}
		}
		record, err = json.MarshalIndent(hashes, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "classes-hashes.json"), append(record, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if runErr != nil || ctx.Err() != nil {
		t.Fatalf("constructor Java control invalid or timed out: %v %v\nstdout:%s\nstderr:%s", runErr, ctx.Err(), stdout.Bytes(), stderr.Bytes())
	}
	output := renderGoFileFromJava(t, source)
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", output, 0)
	if err != nil {
		t.Fatal(err)
	}
	var calls []*ast.CallExpr
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "NewSimpleDateFormatJavaString" {
			return true
		}
		owner, ok := selector.X.(*ast.Ident)
		if ok && owner.Name == "stdjava" {
			calls = append(calls, call)
		}
		return true
	})
	return calls
}

func TestCampaignDateFormatConstructor46TypedAndNullArguments(t *testing.T) {
	calls := dateFormatConstructorCalls46(t, "DateFormatConstructorTyped", `import java.text.SimpleDateFormat; import java.util.Locale;
public class DateFormatConstructorTyped {
 static void make(String pattern, Locale locale) {
  new SimpleDateFormat(pattern);
  new SimpleDateFormat(pattern, locale);
  new SimpleDateFormat(null);
  new SimpleDateFormat("yyyy", (Locale)null);
 }
}`)
	if len(calls) != 4 {
		t.Fatalf("want four runtime constructors, got %d", len(calls))
	}
	for index, call := range calls {
		want := 1
		if index == 1 || index == 3 {
			want = 2
		}
		if len(call.Args) != want {
			t.Fatalf("constructor %d arity %d want %d", index, len(call.Args), want)
		}
		if value, ok := call.Args[0].(*ast.Ident); ok && value.Name == "nil" {
			t.Fatalf("constructor %d lost nullable String target", index)
		}
	}
}

func TestCampaignDateFormatConstructor46SourceShadow(t *testing.T) {
	calls := dateFormatConstructorCalls46(t, "DateFormatConstructorShadow", `class SimpleDateFormat { SimpleDateFormat(String pattern) {} }
public class DateFormatConstructorShadow {
 static void make() {
  new SimpleDateFormat("local");
  new java.text.SimpleDateFormat("yyyy");
 }
}`)
	if len(calls) != 1 {
		t.Fatalf("want only qualified JDK constructor lowered to runtime, got %d", len(calls))
	}
}
