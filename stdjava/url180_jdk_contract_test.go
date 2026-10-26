package stdjava

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func url180Hex(text *JavaString) string {
	if text == nil {
		return "null"
	}
	parts := make([]string, len(text.units))
	for i, c := range text.units {
		parts[i] = fmt.Sprintf("%04x", c)
	}
	return strings.Join(parts, ",")
}
func url180Cause(execution *Execution, failure any) string {
	if failure == nil {
		return "null"
	}
	id, ok := ObjectDynamicType(failure)
	if !ok {
		panic(fmt.Sprintf("non-Java parser failure %T: %v", failure, failure))
	}
	carrier, ok := failure.(interface{ CauseValue() any })
	if !ok {
		panic(fmt.Sprintf("missing cause state %T", failure))
	}
	return string(id) + ":" + url180Hex(JavaThrowableMessageExecution(execution, failure)) + ":" + url180Cause(execution, carrier.CauseValue())
}

func TestURL180PinnedJDKConstructorMatrix(t *testing.T) {
	url180ProviderRegistryScope(t)
	root := filepath.Join("..", "transpiler", "testdata", "url180_contracts")
	raw, err := os.ReadFile(filepath.Join(root, "constructor-inputs.json"))
	if err != nil {
		t.Fatal(err)
	}
	var inputs []*[]uint16
	if err = json.Unmarshal(raw, &inputs); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(root, "constructor-jdk21.txt"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(want), "\n"), "\n")
	if len(inputs) != len(lines) || len(inputs) < 80 {
		t.Fatalf("incomplete pinned matrix %d/%d", len(inputs), len(lines))
	}
	execution := NewExecution()
	for i, input := range inputs {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			var text *JavaString
			if input != nil {
				text = NewJavaStringUTF16(*input)
			}
			var value *JavaURL
			failure := url180Recovered(func() { value = NewJavaURLStringExecution(execution, text) })
			var actual string
			if failure == nil {
				actual = fmt.Sprintf("%d OK %s", i, url180Hex(value.ToExternalFormJava2goExecution(execution)))
			} else {
				actual = fmt.Sprintf("%d ERR %s", i, url180Cause(execution, failure))
			}
			if actual != lines[i] {
				t.Fatalf("actual %s\nJDK21 %s", actual, lines[i])
			}
		})
	}
}
