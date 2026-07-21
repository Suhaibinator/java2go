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
		".parser.Parse()",
		".worker.Name()",
		".values.Add(",
		".values.Get(0)",
		".values.Size()",
		".byKey.Put(",
		".byKey.Get(",
	}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("generated field receiver is missing %q:\n%s", check, out)
		}
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
	if !strings.Contains(flat, ".policy.Score(6)") {
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
	if !strings.Contains(flat, "left.PrimaryScore()") || !strings.Contains(flat, "right.PrimaryScore()") {
		t.Fatalf("expected type-parameter receiver calls to use Ranked's generated method names:\n%s", out)
	}
	if !strings.Contains(flat, "stdjava.StringCompareTo(left.StableKey(), right.StableKey())") {
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
