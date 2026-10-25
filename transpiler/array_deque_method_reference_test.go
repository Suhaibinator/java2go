package transpiler

import (
	"strings"
	"testing"
)

func TestArrayDequeMethodReferences224(t *testing.T) {
	for _, entry := range []struct {
		name, source string
		want, refuse []string
	}{
		{"deque_void_adaptation", `import java.util.ArrayDeque;class Probe {Runnable run(ArrayDeque<String> values){return values::remove;}Runnable last(ArrayDeque<String> values){return values::getLast;}}`, []string{"NewRunnableFuncAdapter", "QueueRemoveExecution", "DequeGetLastExecution", "ReferenceRequireNonNull"}, []string{"values.remove", "values.getLast", "NewPlainRunnableFuncAdapter", "ObjectView["}},
		{"iterator_void_adaptation", `import java.util.Iterator;class Probe {Runnable run(Iterator<String> cursor){return cursor::remove;}}`, []string{"NewRunnableFuncAdapter", "IteratorRemoveExecution"}, []string{"NewPlainRunnableFuncAdapter"}},
		{"typed_read", `import java.util.Deque;interface Read {String value();}class Probe {Read run(Deque<String> values){return values::getFirst;}}`, []string{"DequeGetFirstExecution", "ObjectView[*stdjava.JavaString]"}, nil},
		{"unbound_typed_read", `import java.util.Deque;interface Read {String value(Deque<String> values);}class Probe {Read run(){return Deque<String>::getFirst;}}`, []string{"DequeGetFirstExecution", "ObjectView[*stdjava.JavaString]"}, nil},
		{"raw_unbound_unboxed_read", `import java.util.Deque;interface Read {int value(Deque<Integer> receiver);}class Probe {Read run(){return Deque::getFirst;}}`, []string{"DequeGetFirstExecution", "ObjectView[*stdjava.Integer]", "UnboxInteger"}, nil},
		{"unboxed_read", `import java.util.ArrayDeque;interface Read {int value();}class Probe {Read run(ArrayDeque<Integer> values){return values::remove;}}`, []string{"QueueRemoveExecution", "ObjectView[*stdjava.Integer]", "UnboxInteger"}, nil},
		{"boxed_input", `import java.util.Deque;interface Put {boolean value(int element);}class Probe {Put run(Deque<Integer> values){return values::add;}}`, []string{"QueueAddExecution", "BoxInteger"}, nil},
		{"array_result", `import java.util.ArrayDeque;interface Copy {String[] value(String[] target);}class Probe {Copy run(ArrayDeque<String> values){return values::toArray;}}`, []string{"ArrayDequeCollectionToArrayIntoExecution"}, nil},
		{"source_owner", `class ArrayDeque {String getFirst(){return "source";}}interface Read {String value();}class Probe {Read run(ArrayDeque values){return values::getFirst;}}`, nil, []string{"DequeGetFirstExecution"}},
		{"foreign_owner", `import foreign.ArrayDeque;interface Read {String value();}class Probe {Read run(ArrayDeque values){return values::getFirst;}}`, nil, []string{"DequeGetFirstExecution"}},
	} {
		t.Run(entry.name, func(t *testing.T) {
			generated := renderGoFileFromJava(t, entry.source)
			for _, text := range entry.want {
				if !strings.Contains(generated, text) {
					t.Fatalf("missing method-reference contract %q: %s", text, generated)
				}
			}
			for _, text := range entry.refuse {
				if strings.Contains(generated, text) {
					t.Fatalf("unexpected method-reference lowering %q: %s", text, generated)
				}
			}
		})
	}
}
