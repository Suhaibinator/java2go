package transpiler

import (
	"regexp"
	"strings"
	"testing"
)

func TestComplexProgram_WorkflowEndToEnd(t *testing.T) {
	src := `
package complex.workflow;

public interface Mapper<T, R> { R apply(T value); }

public enum Mode {
    FAST,
    SAFE
}

public abstract class Task {
    String id;
    public Task(String id) { this.id = id; }
    public String name() { return this.id; }
    public abstract int run(String input);
}

public class ParseTask extends Task {
    public ParseTask(String id) { super(id); }

    public int run(String input) {
        Mapper<String, String> trim = s -> s;
        String normalized = trim.apply(input);
        if (normalized instanceof String) {
            return normalized.length();
        }
        return 0;
    }
}

public class App {
    public static int execute(Task task, Mapper<String, String> mapper) {
        String out = mapper.apply(task.name());
        if (task instanceof ParseTask) {
            return out.length();
        }
        return 0;
    }

    public static void main(String[] args) {
        Task task = new ParseTask("x");
        int n = execute(task, v -> v);
        Mode mode = Mode.valueOf("FAST");
        for (Mode each : Mode.values()) {
            System.out.println(each.name());
        }
        System.out.println(n + mode.ordinal());
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "type MapperFuncAdapter[T any, R any] struct") {
		t.Fatalf("expected functional interface adapter generation, got:\n%s", out)
	}
	if !regexp.MustCompile(`NewMapperFuncAdapterJava2goExecution\[\*stdjava\.JavaString, \*stdjava\.JavaString\]\(func\([A-Za-z_][A-Za-z0-9_]* \*stdjava\.Execution, v \*stdjava\.JavaString\) \*stdjava\.JavaString`).MatchString(flat) {
		t.Fatalf("expected lambda in main to be wrapped and typed, got:\n%s", out)
	}
	if !strings.Contains(flat, `stdjava.ObjectInstanceOf(task, stdjava.TypeID("complex.workflow.ParseTask"))`) {
		t.Fatalf("expected class instanceof conversion in execute(), got:\n%s", out)
	}
	if !strings.Contains(flat, "stdjava.ObjectInstanceOf(normalized, stdjava.StringTypeID)") {
		t.Fatalf("expected String instanceof conversion in run(), got:\n%s", out)
	}
	if !regexp.MustCompile(`ModeValueOfJava2goExecution\([A-Za-z_][A-Za-z0-9_]*, stdjava\.JavaStringLiteralUTF16\(\[\]uint16\{70, 65, 83, 84\}\)\)`).MatchString(flat) {
		t.Fatalf("expected enum valueOf rewrite to generated helper, got:\n%s", out)
	}
	if !regexp.MustCompile(`ModeValuesJava2goExecution\([A-Za-z_][A-Za-z0-9_]*\)`).MatchString(flat) {
		t.Fatalf("expected enum values() rewrite to generated helper, got:\n%s", out)
	}
}

func TestComplexProgram_VoidLambdaAdapterInMainProgram(t *testing.T) {
	src := `
package complex.handler;

public interface Handler<T> { void handle(T value); }

public class Logger {
    public static void writeAll(Handler<String> handler) {
        handler.handle("first");
        handler.handle("second");
    }

    public static void main(String[] args) {
        writeAll(v -> { System.out.println(v); });
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "func WriteAll(handler Handler[*stdjava.JavaString])") {
		t.Fatalf("expected interface method parameter type without pointer indirection, got:\n%s", out)
	}
	if !strings.Contains(flat, "type HandlerFuncAdapter[T any] struct") {
		t.Fatalf("expected functional adapter type for Handler<T>, got:\n%s", out)
	}
	if !strings.Contains(flat, "WriteAllJava2goExecution(__java2goExecution, NewHandlerFuncAdapterJava2goExecution[*stdjava.JavaString](func(__java2goExecution *stdjava.Execution, v *stdjava.JavaString)") {
		t.Fatalf("expected void lambda to be wrapped with typed adapter in main(), got:\n%s", out)
	}
}

func TestComplexProgram_GenericFactoryAndDiamondInLambda(t *testing.T) {
	src := `
package complex.factory;

public interface Factory<T> { T create(String seed); }

public class Box<T> {
    T value;
    public Box(T value) { this.value = value; }
    public T get() { return this.value; }
}

public class FactoryApp {
    public static Box<String> build(Factory<Box<String>> factory) {
        return factory.create("seed");
    }

    public static void main(String[] args) {
        Box<String> box = build(seed -> new Box<>(seed));
        System.out.println(box.get());
    }
}
`

	out := renderGoFileFromJava(t, src)
	assertComplexFactoryContract(t, out)
}
