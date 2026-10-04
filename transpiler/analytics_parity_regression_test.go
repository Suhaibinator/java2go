package transpiler

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFieldAccessReceiverTypeDrivesGeneratedMethodNamesAndCollectionIntrinsics(t *testing.T) {
	src := `
package parity.analytics.fields;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public interface Parser {
    String parse();
}

public class Worker {
    public String name() { return "worker"; }
}

public class FieldBackedCalls {
    private Parser parser;
    private Worker worker;
    private List<String> values;
    private Map<String, String> byKey;

    public FieldBackedCalls(Parser parser, Worker worker) {
        this.parser = parser;
        this.worker = worker;
        this.values = new ArrayList<String>();
        this.byKey = new HashMap<String, String>();
    }

    public String use() {
        this.values.add(this.parser.parse());
        this.byKey.put("worker", this.worker.name());
        return this.values.get(0) + this.byKey.get("worker") + this.values.size();
    }
}
`

	out := renderGoFileFromJava(t, src)
	checks := []string{
		"__java2goInvocationReceiver := fs.parser",
		".ParseJava2goExecution(__java2goExecution)",
		".worker.NameJava2goExecution(__java2goExecution)",
		"stdjava.EvaluationValue(fs.values).Add(",
		".values.Get(0)",
		".values.Size()",
		"stdjava.MapPutExecution(__java2goExecution, stdjava.EvaluationValue(fs.byKey), stdjava.JavaStringLiteralUTF16([]uint16{119, 111, 114, 107, 101, 114}), fs.worker.NameJava2goExecution(__java2goExecution))",
		"stdjava.ObjectView[*stdjava.JavaString](fs.byKey.GetObject(stdjava.JavaStringLiteralUTF16([]uint16{119, 111, 114, 107, 101, 114}), __java2goExecution), stdjava.StringTypeID)",
	}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("generated field receiver is missing %q:\n%s", check, out)
		}
	}
	add := strings.Index(out, "stdjava.EvaluationValue(fs.values).Add(")
	put := strings.Index(out, "stdjava.MapPutExecution(__java2goExecution, stdjava.EvaluationValue(fs.byKey),")
	getList := strings.Index(out, "fs.values.Get(0)")
	getMap := strings.Index(out, "fs.byKey.GetObject(stdjava.JavaStringLiteralUTF16([]uint16{119, 111, 114, 107, 101, 114}), __java2goExecution)")
	size := strings.Index(out, "fs.values.Size()")
	if add < 0 || put <= add || getList <= put || getMap <= getList || size <= getMap {
		t.Errorf("generated calls changed Java statement or return-expression evaluation order:\n%s", out)
	}
	for _, stale := range []string{".parser.parse(", ".worker.name(", ".values.add(", ".values.get(", ".values.size(", ".byKey.put(", ".byKey.get("} {
		if strings.Contains(out, stale) {
			t.Errorf("generated field receiver retained Java method spelling %q:\n%s", stale, out)
		}
	}
}

func TestAbstractClassFieldUsesCompanionInterface(t *testing.T) {
	src := `
package parity.analytics.abstractfield;

public abstract class ScorePolicy {
    public abstract int score(int value);
}

public class ConcretePolicy extends ScorePolicy {
    public int score(int value) { return value + 1; }
}

public class Engine {
    private ScorePolicy policy;

    public Engine(ScorePolicy policy) {
        this.policy = policy;
    }

    public int run() {
        return this.policy.score(6);
    }
}
`

	flat := normalizeSpaces(renderGoFileFromJava(t, src))
	if !strings.Contains(flat, "policy ScorePolicyI") {
		t.Fatalf("expected abstract-class field to use its companion interface:\n%s", flat)
	}
	if !strings.Contains(flat, "func NewEngine(policy ScorePolicyI)") {
		t.Fatalf("expected constructor parameter to use the same companion interface:\n%s", flat)
	}
	if !strings.Contains(flat, "__java2goInvocationReceiver := ee.policy") ||
		!strings.Contains(flat, "ScoreJava2goExecution(__java2goExecution, __java2goInvocationArg0)") {
		t.Fatalf("expected method resolution through the abstract-class field:\n%s", flat)
	}
}

func TestInterfaceBoundUsesInterfaceConstraintAndResolvesBoundMethods(t *testing.T) {
	src := `
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

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "type StableRanker[T Ranked] struct") {
		t.Fatalf("expected interface upper bound to be emitted as an interface constraint:\n%s", out)
	}
	if !strings.Contains(flat, "func NewStableRanker[T Ranked]") {
		t.Fatalf("expected constructor to preserve the interface constraint:\n%s", out)
	}
	if strings.Contains(flat, "T *Ranked") {
		t.Fatalf("interface upper bound must not be pointer-wrapped:\n%s", out)
	}
	if strings.Count(flat, ".PrimaryScoreJava2goExecution(__java2goExecution)") != 4 {
		t.Fatalf("expected type-parameter receiver calls to use Ranked's generated method names:\n%s", out)
	}
	// The String intrinsic stages its receiver and argument before the null
	// check, preserving Java's invocation evaluation order.
	receiverStage := strings.Index(flat, "__java2goInvocationReceiver := func() string")
	argumentStage := strings.Index(flat, "__java2goInvocationArg0 := func() string")
	compareCall := strings.Index(flat, "stdjava.StringCompareTo(stdjava.StringRequireNonNull(__java2goInvocationReceiver), __java2goInvocationArg0)")
	if receiverStage < 0 || argumentStage <= receiverStage || compareCall <= argumentStage ||
		!strings.Contains(flat, "return func() int32 {") || !strings.Contains(flat, "}() < 0") ||
		strings.Count(flat, ".StableKeyJava2goExecution(__java2goExecution)") != 2 {
		t.Fatalf("expected String return from the bound method to drive compareTo intrinsic lowering:\n%s", out)
	}
}

func TestExplicitNullLocalsKeepCrossPackageAndGenericQualification(t *testing.T) {
	root := t.TempDir()
	writeJavaTestSource(t, root, "parity/nulls/model/Event.java", `
package parity.nulls.model;
public class Event {}
`)
	writeJavaTestSource(t, root, "parity/nulls/app/NullLocals.java", `
package parity.nulls.app;

import java.util.ArrayList;
import java.util.List;
import parity.nulls.model.Event;

public class NullLocals {
    public Event choose(boolean populate) {
        Event selected = null;
        List<Event> staged = null;
        if (populate) {
            staged = new ArrayList<Event>();
            staged.add(new Event());
            selected = staged.get(0);
        }
        return selected;
    }
}
`)

	outputs := convertJavaProjectDir(t, root)
	out := outputs[filepath.ToSlash("parity/nulls/app/NullLocals.go")]
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "var selected *model.Event = nil") {
		t.Fatalf("expected null local to keep its generated-package qualifier:\n%s", out)
	}
	if !strings.Contains(flat, "var staged *stdjava.List[*model.Event] = nil") {
		t.Fatalf("expected generic null local to keep runtime and element qualifiers:\n%s", out)
	}
	if !strings.Contains(flat, `model "parity/nulls/model"`) {
		t.Fatalf("expected generated model import for explicitly typed locals:\n%s", out)
	}
}
