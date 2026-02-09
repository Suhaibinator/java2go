package transpiler

import (
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
	if !strings.Contains(flat, "NewMapperFuncAdapter[string, string](func(v string) string") {
		t.Fatalf("expected lambda in main to be wrapped and typed, got:\n%s", out)
	}
	if !strings.Contains(flat, "any(task).(*ParseTask)") {
		t.Fatalf("expected class instanceof conversion in execute(), got:\n%s", out)
	}
	if !strings.Contains(flat, "any(normalized).(string)") {
		t.Fatalf("expected String instanceof conversion in run(), got:\n%s", out)
	}
	if !strings.Contains(flat, "ModeValueOf(\"FAST\")") {
		t.Fatalf("expected enum valueOf rewrite to generated helper, got:\n%s", out)
	}
	if !strings.Contains(flat, "ModeValues()") {
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

	if !strings.Contains(flat, "func WriteAll(handler Handler[string])") {
		t.Fatalf("expected interface method parameter type without pointer indirection, got:\n%s", out)
	}
	if !strings.Contains(flat, "type HandlerFuncAdapter[T any] struct") {
		t.Fatalf("expected functional adapter type for Handler<T>, got:\n%s", out)
	}
	if !strings.Contains(flat, "WriteAll(NewHandlerFuncAdapter[string](func(v string)") {
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
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "func Build(factory Factory[*Box[string]]) *Box[string]") {
		t.Fatalf("expected nested generic interface + return types in build(), got:\n%s", out)
	}
	if !strings.Contains(flat, "Build(NewFactoryFuncAdapter[*Box[string]](func(seed string) *Box[string]") {
		t.Fatalf("expected lambda to infer generic return type and use adapter, got:\n%s", out)
	}
	if !strings.Contains(flat, "NewBox[string](seed)") && !strings.Contains(flat, "NewBox[*Box[string]](seed)") && !strings.Contains(flat, "ConstructBox[*Box[string]](seed)") {
		t.Fatalf("expected constructor call with inferred generic type inside lambda body, got:\n%s", out)
	}
}
