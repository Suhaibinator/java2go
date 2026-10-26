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
	"strings"
	"testing"
	"time"
)

const ci203BoundOriginalJava = `
package parity.analytics.bounds;

public interface Ranked {
    int primaryScore();
    String stableKey();
}

public class StableRanker<T extends Ranked> {
    public boolean before(T left, T right) {
        if (left.primaryScore() != right.primaryScore()) {
            return left.primaryScore() > right.primaryScore();
        }
        return left.stableKey().compareTo(right.stableKey()) < 0;
    }
}
`

// This is the original before() expression in the canonical JavaString ABI.
// Compare syntax trees so whitespace is irrelevant while receiver/argument
// staging, invocation execution, null-check timing and result use stay exact.
const ci203BoundExpectedReturn = `	return func() int32 {
		__java2goInvocationReceiver := func() *stdjava.JavaString {
			__java2goInvocationReceiver := left
			__java2goVirtualInterfaceReceiver, __java2goHasVirtualInterfaceReceiver := interface {
			}(__java2goInvocationReceiver).(interface {
				Java2goInterfaceVirtual_7061726974792E616E616C79746963732E626F756E64732E52616E6B656400737461626C654B657901537472696E67(__java2goExecution *stdjava.Execution) *stdjava.JavaString
			})
			if __java2goHasVirtualInterfaceReceiver {
				return __java2goVirtualInterfaceReceiver.Java2goInterfaceVirtual_7061726974792E616E616C79746963732E626F756E64732E52616E6B656400737461626C654B657901537472696E67(__java2goExecution)
			}
			__java2goExecutionReceiver, __java2goHasExecutionReceiver := interface {
			}(__java2goInvocationReceiver).(RankedJava2goExecution)
			if __java2goHasExecutionReceiver {
				return __java2goExecutionReceiver.StableKeyJava2goExecution(__java2goExecution)
			}
			return interface {
			}(stdjava.ReferenceRequireNonNull(__java2goInvocationReceiver)).(Ranked).StableKey()
		}()
		__java2goInvocationArg0 := func() *stdjava.JavaString {
			__java2goInvocationReceiver := right
			__java2goVirtualInterfaceReceiver, __java2goHasVirtualInterfaceReceiver := interface {
			}(__java2goInvocationReceiver).(interface {
				Java2goInterfaceVirtual_7061726974792E616E616C79746963732E626F756E64732E52616E6B656400737461626C654B657901537472696E67(__java2goExecution *stdjava.Execution) *stdjava.JavaString
			})
			if __java2goHasVirtualInterfaceReceiver {
				return __java2goVirtualInterfaceReceiver.Java2goInterfaceVirtual_7061726974792E616E616C79746963732E626F756E64732E52616E6B656400737461626C654B657901537472696E67(__java2goExecution)
			}
			__java2goExecutionReceiver, __java2goHasExecutionReceiver := interface {
			}(__java2goInvocationReceiver).(RankedJava2goExecution)
			if __java2goHasExecutionReceiver {
				return __java2goExecutionReceiver.StableKeyJava2goExecution(__java2goExecution)
			}
			return interface {
			}(stdjava.ReferenceRequireNonNull(__java2goInvocationReceiver)).(Ranked).StableKey()
		}()
		return stdjava.RequireJavaString(__java2goInvocationReceiver).CompareTo(__java2goInvocationArg0)
	}() < 0`

func ci203BoundStringContract(out string) error {
	file, err := ci199ParseContract(out)
	if err != nil {
		return err
	}
	target, err := ci199Function(file, "BeforeJava2goExecution")
	if err != nil {
		return err
	}
	expected := ci199ExpectedFunction(`func (sr *StableRanker[T]) BeforeJava2goExecution(__java2goExecution *stdjava.Execution, left Ranked, right Ranked) bool {` + ci203BoundExpectedReturn + `}`)
	if target.Recv == nil || len(target.Recv.List) != 1 || ci199ASTText(target.Recv.List[0].Type) != ci199ASTText(expected.Recv.List[0].Type) || ci199ASTText(target.Type) != ci199ASTText(expected.Type) {
		return ci199Reject("original bound receiver, formals or result changed")
	}
	if !ci199NoShadow(target, "stdjava", "Ranked", "RankedJava2goExecution") {
		return ci199Reject("bound dispatch/runtime binding shadowed")
	}
	if target.Body == nil || len(target.Body.List) == 0 {
		return ci199Reject("missing bound comparison")
	}
	result, ok := target.Body.List[len(target.Body.List)-1].(*ast.ReturnStmt)
	if !ok || ci199ASTText(result) != ci199ASTText(expected.Body.List[0]) {
		return ci199Reject("bound String staging, current Execution, null check, once-only compare or result changed")
	}
	return nil
}

func ci203AssertBoundStringContract(t *testing.T, out string) {
	t.Helper()
	if err := ci203BoundStringContract(out); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestCI203BoundStringGuardNegativeControls(t *testing.T) {
	out := renderGoFileFromJava(t, ci203BoundOriginalJava)
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, "generated.go", out, 0)
	if err != nil {
		t.Fatal(err)
	}
	fn, err := ci199Function(file, "BeforeJava2goExecution")
	if err != nil {
		t.Fatal(err)
	}
	statement := fn.Body.List[len(fn.Body.List)-1]
	tail := out[positions.Position(statement.Pos()).Offset:positions.Position(statement.End()).Offset]
	const checked = "stdjava.RequireJavaString(__java2goInvocationReceiver).CompareTo(__java2goInvocationArg0)"
	negatives := []struct{ name, before, after string }{
		{"host-string-result", "func() *stdjava.JavaString", "func() string"},
		{"wrong-left-receiver", "__java2goInvocationReceiver := left", "__java2goInvocationReceiver := right"},
		{"wrong-right-receiver", "__java2goInvocationReceiver := right", "__java2goInvocationReceiver := left"},
		{"wrong-current-execution", ".StableKeyJava2goExecution(__java2goExecution)", ".StableKeyJava2goExecution(stdjava.NewExecution())"},
		{"wrong-bound-method", ".StableKeyJava2goExecution(__java2goExecution)", ".OtherKeyJava2goExecution(__java2goExecution)"},
		{"missing-null-check", checked, "__java2goInvocationReceiver.CompareTo(__java2goInvocationArg0)"},
		{"reversed-operands", checked, "stdjava.RequireJavaString(__java2goInvocationArg0).CompareTo(__java2goInvocationReceiver)"},
		{"duplicated-compare", "return " + checked, "_ = " + checked + "\nreturn " + checked},
		{"constant-result", checked, "int32(0)"},
		{"ignored-compare-result", "}() < 0", "}() > 0"},
		{"early-null-check", "__java2goInvocationArg0 :=", "_ = stdjava.RequireJavaString(__java2goInvocationReceiver)\n__java2goInvocationArg0 :="},
	}
	mutations := make([]ci199Mutation, 0, len(negatives)+1)
	for _, negative := range negatives {
		if !strings.Contains(tail, negative.before) {
			t.Fatalf("negative %s missing preimage", negative.name)
		}
		mutations = append(mutations, ci199Mutation{name: negative.name, function: fn.Name.Name, before: tail, after: strings.Replace(tail, negative.before, negative.after, 1)})
	}
	mutations = append(mutations, ci199Mutation{name: "runtime-import-spoof", before: `"github.com/NickyBoy89/java2go/stdjava"`, after: `"example.invalid/stdjava"`})
	ci199CheckNegatives(t, out, ci203BoundStringContract, mutations)
}

// Keep the two original public declarations intact, placing each in its own
// Java source file, and exercise before() through a concrete implementation.
func TestCI203BoundStringOriginalJDK21Behavior(t *testing.T) {
	const marker = "public class StableRanker"
	parts := strings.Split(ci203BoundOriginalJava, marker)
	if len(parts) != 2 {
		t.Fatal("original StableRanker declaration missing")
	}
	const probe = `
public class BoundStringProbe {
 static String events = "";
 static class Value implements Ranked {
  int score; String key; String tag;
  Value(int score, String key, String tag) { this.score=score; this.key=key; this.tag=tag; }
  public int primaryScore() { return score; }
  public String stableKey() { events += tag; return key; }
 }
 public static String run() {
  String result = "";
  StableRanker<Value> ranker = new StableRanker<Value>();
  Value left = new Value(1, "a", "L");
  Value right = new Value(1, "b", "R");
  result += ranker.before(left,right)+":"+events+"\n";
  events="";
  result += ranker.before(right,left)+":"+events+"\n";
  events="";
  result += ranker.before(new Value(2,"a","H"),right)+":"+events+"\n";
  events="";
  try { ranker.before(new Value(1,null,"N"),right); }
  catch (NullPointerException expected) { result += "null:"+events+"\n"; }
  events="";
  result += ranker.before(new Value(1,"\uD800\uDC00","U"), new Value(1,"\uE000","B"))+":"+events+"\n";
  return result;
 }
}
`
	const want = "true:LR\nfalse:RL\ntrue:\nnull:NR\ntrue:UB\n"
	root := t.TempDir()
	files := map[string]string{
		"Ranked.java":            parts[0],
		"StableRanker.java":      "package parity.analytics.bounds;\n" + marker + parts[1],
		"BoundStringProbe.java":  "package parity.analytics.bounds;\n" + probe,
		"BoundStringDriver.java": `package parity.analytics.bounds; class BoundStringDriver { public static void main(String[] args) { System.out.print(BoundStringProbe.run()); } }`,
	}
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(tool string, args ...string) string {
		t.Helper()
		executable, err := campaignCompilerJavaTool(tool)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, executable, args...)
		cmd.Dir = root
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		t.Logf("direct JDK21: %v", cmd.Args)
		if err := cmd.Run(); err != nil || stderr.Len() != 0 {
			t.Fatalf("JDK21 %s failed: %v stdout=%q stderr=%q", tool, err, stdout.String(), stderr.String())
		}
		return stdout.String()
	}
	if out := run("javac", "-J-XX:ActiveProcessorCount=2", "--release", "21", "-encoding", "UTF-8", "-d", root, "Ranked.java", "StableRanker.java", "BoundStringProbe.java", "BoundStringDriver.java"); out != "" {
		t.Fatalf("javac stdout=%q", out)
	}
	measured := run("java", "-XX:ActiveProcessorCount=2", "-Dstdout.encoding=UTF-8", "-Dstderr.encoding=UTF-8", "-cp", root, "parity.analytics.bounds.BoundStringDriver")
	if measured != want {
		t.Fatalf("original JDK21 behavior=%q want=%q", measured, want)
	}
	t.Logf("fresh JDK21 oracle stdout=%q stderr=empty", measured)
	runGeneratedWithStdjava(t, renderGoFileFromJava(t, ci203BoundOriginalJava+probe), fmt.Sprintf(`
package bounds
import (
 "slices"
 "testing"
 "unicode/utf16"
)
func TestOriginalBoundStringParity(t *testing.T) {
 got := Run()
 want := utf16.Encode([]rune(%q))
 if got == nil || !slices.Equal(got.UTF16Copy(),want) { t.Fatalf("bound String result=%%v, want UTF16 %%v",got,want) }
}
`, measured))
}
