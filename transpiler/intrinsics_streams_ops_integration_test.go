package transpiler

import "testing"

// Stage 2 stream operations: the element type flows from the receiver, and the
// operations that change or wrap it are emitted as stdjava free functions.
func TestStreamOps_IntermediateOperations(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.List;
public class OpsProgram {
    public static void run() {
        List<Integer> xs = new ArrayList<Integer>();
        xs.stream().distinct().toList();
        xs.stream().skip(2).toList();
        xs.stream().peek(n -> System.out.println(n)).toList();
        xs.stream().sorted().toList();
        xs.stream().sorted((a, b) -> b - a).toList();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.StreamDistinct(")
	assertContains(t, out, ".Skip(")
	assertContains(t, out, ".Peek(func(n int32) {")
	assertContains(t, out, "stdjava.StreamSorted(")
	assertContains(t, out, "stdjava.StreamSortedWith(")
	// The comparator's closure returns Java int, not the element type.
	assertContains(t, out, "func(a int32, b int32) int32")
}

func TestStreamOps_TerminalOperationsReturningOptional(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.List;
public class TerminalProgram {
    public static void run() {
        List<Integer> xs = new ArrayList<Integer>();
        int first = xs.stream().findFirst().get();
        int any = xs.stream().findAny().get();
        int smallest = xs.stream().min((a, b) -> a - b).get();
        int largest = xs.stream().max((a, b) -> a - b).get();
    }
}
`
	out := renderGoFileFromJava(t, src)
	// Each must resolve .get() on the returned Optional, which only works when
	// the terminal's result type is known to be an Optional.
	assertContains(t, out, ".FindFirst().Get()")
	assertContains(t, out, ".FindAny().Get()")
	assertContains(t, out, "stdjava.StreamMinWith(")
	assertContains(t, out, "stdjava.StreamMaxWith(")
	assertNotContains(t, out, ".get()")
}

func TestStreamOps_ReduceArities(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.List;
public class ReduceArityProgram {
    public static void run() {
        List<Integer> xs = new ArrayList<Integer>();
        int withIdentity = xs.stream().reduce(0, (a, b) -> a + b);
        int withoutIdentity = xs.stream().reduce((a, b) -> a + b).get();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.StreamReduce(")
	assertContains(t, out, "stdjava.StreamReduceOptional(")
}

// flatMap's mapper returns a Stream, so its result type has to come from the
// mapper's body rather than the receiver's element type.
func TestStreamOps_FlatMapInfersTheMappedStreamType(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.List;
import java.util.stream.Stream;
public class FlatMapProgram {
    public static void run() {
        List<String> words = new ArrayList<String>();
        words.stream().flatMap(w -> Stream.of(w, w)).toList();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.StreamFlatMap(")
	assertContains(t, out, "func(w string) stdjava.Stream[string]")
}

// parallelStream and parallel() run sequentially, and must still chain.
func TestStreamOps_ParallelMapsToTheSequentialPath(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.List;
public class ParallelProgram {
    public static void run() {
        List<Integer> xs = new ArrayList<Integer>();
        long a = xs.parallelStream().sorted().count();
        long b = xs.stream().parallel().sequential().count();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.StreamSorted(stdjava.StreamOfSlice(xs.Slice())).Count()")
	assertContains(t, out, ".Parallel().Sequential().Count()")
}

func TestOptionalOps_NewOperations(t *testing.T) {
	src := `
import java.util.Optional;
public class OptionalOpsProgram {
    public static void run() {
        Optional<String> name = Optional.of("java");
        boolean kept = name.filter(s -> s.length() == 4).isPresent();
        String viaFlatMap = name.flatMap(s -> Optional.of(s + "2go")).get();
        String fallback = name.orElseGet(() -> "none");
        String required = name.orElseThrow();
        String custom = name.orElseThrow(() -> new IllegalStateException("missing"));
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, ".Filter(func(s string) bool")
	assertContains(t, out, "stdjava.OptionalFlatMap(name, func(s string) stdjava.Optional[string]")
	// A zero-parameter Supplier still needs its result type applied.
	assertContains(t, out, ".OrElseGet(func() string")
	assertContains(t, out, ".OrElseThrow(nil)")
	assertContains(t, out, ".OrElseThrow(func() any")
}
