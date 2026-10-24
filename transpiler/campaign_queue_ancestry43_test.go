package transpiler

import (
	"fmt"
	"strings"
	"testing"
)

func TestCampaignQueueAncestry43GenericJVM(t *testing.T) {
	source := `import java.util.Collection;import java.util.AbstractCollection;import java.util.ArrayDeque;import java.util.Deque;import java.util.Queue;
class QueueAncestry {static class Filter{} static <E> int select(Collection<E> value){return 1;}static int select(Object value){return 2;}static <E> int base(AbstractCollection<E> value){return 1;}static int base(Object value){return 2;}static <E> int queue(Queue<E> value){return 1;}static int queue(Object value){return 2;}static <E> int deque(Deque<E> value){return 1;}static int deque(Object value){return 2;}
public static String run(){ArrayDeque<Filter> a=new ArrayDeque<>();Deque<Filter> d=a;Queue<Filter> q=d;return select(a)+":"+select(d)+":"+select(q)+":"+base(a)+":"+queue(a)+":"+deque(a);}}`
	want := campaignRuntimeJavaOracle(t, "QueueAncestry", source)
	t.Logf("JVM oracle %q", want)
	h := setupParseHelper(t, source)
	var values []string
	for _, pair := range [][2]string{{"ArrayDeque<Filter>", "Collection<E>"}, {"Deque<Filter>", "Collection<E>"}, {"Queue<Filter>", "Collection<E>"}, {"ArrayDeque<Filter>", "AbstractCollection<E>"}, {"ArrayDeque<Filter>", "Queue<E>"}, {"ArrayDeque<Filter>", "Deque<E>"}} {
		value := 2
		if builtinJavaReferenceAssignableWithTypeParameters(pair[0], pair[1], []string{"E"}, h.Ctx) {
			value = 1
		}
		values = append(values, fmt.Sprint(value))
	}
	if got := strings.Join(values, ":"); got != want {
		t.Fatalf("resolution %q; JVM %q", got, want)
	}
}
func TestCampaignQueueAncestry43InvariantJVM(t *testing.T) {
	source := `class QueueInvariant {static int select(java.util.Collection<Object> value){return 1;}static int select(Object value){return 2;}public static String run(){java.util.ArrayDeque<String> a=new java.util.ArrayDeque<>();java.util.Deque<String>d=a;java.util.Queue<String>q=d;return select(a)+":"+select(d)+":"+select(q);}}`
	want := campaignRuntimeJavaOracle(t, "QueueInvariant", source)
	t.Logf("JVM oracle %q", want)
	h := setupParseHelper(t, source)
	var values []string
	for _, typ := range []string{"java.util.ArrayDeque<String>", "java.util.Deque<String>", "java.util.Queue<String>"} {
		v := 2
		if builtinJavaReferenceAssignable(typ, "java.util.Collection<Object>", h.Ctx) {
			v = 1
		}
		values = append(values, fmt.Sprint(v))
	}
	if got := strings.Join(values, ":"); got != want {
		t.Fatalf("resolution %q; JVM %q", got, want)
	}
}
func TestCampaignQueueAncestry43SourceShadowJVM(t *testing.T) {
	source := `class ArrayDeque<E>{} class Deque<E>{} class Queue<E>{} class QueueShadow {static <E> int select(java.util.Collection<E> value){return 1;}static int select(Object value){return 2;}public static String run(){return select(new ArrayDeque<String>())+":"+select(new Deque<String>())+":"+select(new Queue<String>());}}`
	want := campaignRuntimeJavaOracle(t, "QueueShadow", source)
	t.Logf("JVM oracle %q", want)
	h := setupParseHelper(t, source)
	var values []string
	for _, typ := range []string{"ArrayDeque<String>", "Deque<String>", "Queue<String>"} {
		v := 2
		if builtinJavaReferenceAssignableWithTypeParameters(typ, "java.util.Collection<E>", []string{"E"}, h.Ctx) {
			v = 1
		}
		values = append(values, fmt.Sprint(v))
	}
	if got := strings.Join(values, ":"); got != want {
		t.Fatalf("resolution %q; JVM %q", got, want)
	}
}
