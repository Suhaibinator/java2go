package transpiler

import (
	"strings"
	"testing"
)

func TestWorkflowResolution_ChainedReturnKeepsDeclaringPackage(t *testing.T) {
	root := t.TempDir()
	writeWorkflowJavaSource(t, root, "parity/chain/model/Priority.java", `
package parity.chain.model;
public class Priority {
    public int getWeight() { return 30; }
}
`)
	writeWorkflowJavaSource(t, root, "parity/chain/model/Task.java", `
package parity.chain.model;
public class Task {
    private Priority priority;
    public Priority getPriority() { return priority; }
}
`)
	writeWorkflowJavaSource(t, root, "parity/chain/engine/Planner.java", `
package parity.chain.engine;
import parity.chain.model.Task;
public class Planner {
    public static int weight(Task task) {
        return task.getPriority().getWeight();
    }
}
`)

	outputs := convertJavaProjectDir(t, root)
	out := outputs["parity/chain/engine/Planner.go"]
	if !strings.Contains(out, "task.GetPriorityJava2goExecution(__java2goExecution).GetWeightJava2goExecution(__java2goExecution)") {
		t.Fatalf("expected a chained return type to resolve in the declaring method's package, got:\n%s", out)
	}
}

func TestWorkflowResolution_ListGetPreservesElementTypeForChainedCall(t *testing.T) {
	src := `
import java.util.List;
class Item {
    public String getId() { return "item"; }
}
public class Plan {
    public static String first(List<Item> items) {
        return items.get(0).getId();
    }
}
`

	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "items.Get(0).GetIdJava2goExecution(__java2goExecution)") {
		t.Fatalf("expected List.get to retain its element type for the chained call, got:\n%s", out)
	}
}

func TestWorkflowResolution_LambdaBodyUsesTypedSAMParameters(t *testing.T) {
	src := `
interface Action<T> {
    String apply(Task<T> task);
}
class Task<T> {
    public T getPayload() { return null; }
    public int getAttempts() { return 1; }
}
public class Lambdas {
    public static void run() {
        String prefix = "attempts=";
        Action<String> action = task -> {
            String payload = task.getPayload();
            return prefix + payload + task.getAttempts();
        };
    }
}
`

	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "task.GetPayloadJava2goExecution(__java2goExecution)") ||
		!strings.Contains(out, "task.GetAttemptsJava2goExecution(__java2goExecution)") {
		t.Fatalf("expected inferred SAM parameters to resolve method calls while parsing the lambda body, got:\n%s", out)
	}
}

func TestWorkflowResolution_ImportAliasAvoidsLocalUsedBeforeSAMAdapter(t *testing.T) {
	root := t.TempDir()
	writeWorkflowJavaSource(t, root, "parity/alias/engine/TaskAction.java", `
package parity.alias.engine;
public interface TaskAction<T> {
    T execute(T value);
}
`)
	writeWorkflowJavaSource(t, root, "parity/alias/engine/WorkflowEngine.java", `
package parity.alias.engine;
public class WorkflowEngine<T> {
    public WorkflowEngine() {}
    public void run(TaskAction<T> action) {}
}
`)
	writeWorkflowJavaSource(t, root, "parity/alias/app/Application.java", `
package parity.alias.app;
import parity.alias.engine.TaskAction;
import parity.alias.engine.WorkflowEngine;
public class Application {
    public static void run() {
        WorkflowEngine<String> engine = new WorkflowEngine<String>();
        TaskAction<String> action = value -> value;
        engine.run(action);
    }
}
`)

	outputs := convertJavaProjectDir(t, root)
	out := outputs["parity/alias/app/Application.go"]
	flat := normalizeSpaces(out)
	for _, want := range []string{
		`enginepkg "parity/alias/engine"`,
		"enginepkg.NewWorkflowEngineJava2goExecution[string](__java2goExecution)",
		"enginepkg.NewTaskActionFuncAdapterJava2goExecution[string]",
	} {
		if !strings.Contains(flat, normalizeSpaces(want)) {
			t.Fatalf("expected collision-safe package qualification %q, got:\n%s", want, out)
		}
	}
}

func writeWorkflowJavaSource(t *testing.T, root, relativePath, source string) {
	t.Helper()
	writeJavaSource(t, root, relativePath, source)
}
