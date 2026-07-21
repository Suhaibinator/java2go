package transpiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImplementedInterfaceEmbedding_PreservesGenericArgumentsAcrossPackages(t *testing.T) {
	root := t.TempDir()
	writeJavaTestSource(t, root, "parity/generic/api/RecordParser.java", `
package parity.generic.api;
public interface RecordParser<T> {
    T parse();
}
`)
	writeJavaTestSource(t, root, "parity/generic/model/Event.java", `
package parity.generic.model;
public class Event {}
`)
	writeJavaTestSource(t, root, "parity/generic/impl/EventParser.java", `
package parity.generic.impl;
import parity.generic.api.RecordParser;
import parity.generic.model.Event;
public abstract class EventParser implements RecordParser<Event> {}
`)

	outputs := convertJavaProjectDir(t, root)
	out := outputs["parity/generic/impl/EventParser.go"]
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "type EventParser struct { api.RecordParser[*model.Event] }") {
		t.Fatalf("expected implemented interface and concrete type argument to retain package qualification, got:\n%s", out)
	}
	if !strings.Contains(flat, `api "parity/generic/api"`) || !strings.Contains(flat, `model "parity/generic/model"`) {
		t.Fatalf("expected imports for implemented interface and its type argument, got:\n%s", out)
	}
}

func TestImplementedInterfaceEmbedding_PreservesInScopeTypeParameter(t *testing.T) {
	root := t.TempDir()
	writeJavaTestSource(t, root, "parity/generic/api/TaskRule.java", `
package parity.generic.api;
public interface TaskRule<T> {
    boolean evaluate(T value);
}
`)
	writeJavaTestSource(t, root, "parity/generic/impl/EffortLimitRule.java", `
package parity.generic.impl;
import parity.generic.api.TaskRule;
public abstract class EffortLimitRule<T> implements TaskRule<T> {}
`)

	outputs := convertJavaProjectDir(t, root)
	out := outputs["parity/generic/impl/EffortLimitRule.go"]
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "type EffortLimitRule[T any] struct { api.TaskRule[T] }") {
		t.Fatalf("expected implemented interface to retain the implementing class type parameter, got:\n%s", out)
	}
	if strings.Contains(flat, "api.TaskRule[*T]") {
		t.Fatalf("type parameters in implemented interfaces must not be pointer-wrapped, got:\n%s", out)
	}
}

func writeJavaTestSource(t *testing.T, root, relativePath, source string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating Java test source directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("writing Java test source: %v", err)
	}
}
