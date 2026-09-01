package transpiler

import (
	"strings"
	"testing"
)

func assertNotContains(t *testing.T, out, unwanted string) {
	t.Helper()
	if strings.Contains(out, unwanted) {
		t.Fatalf("generated Go unexpectedly contains %q:\n%s", unwanted, out)
	}
}

func TestComparator_CollectionsSortTypesTheLambdaFromTheListArgument(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
public class SortProgram {
    public static void run() {
        List<Integer> xs = new ArrayList<Integer>();
        Collections.sort(xs, (a, b) -> a - b);
    }
}
`
	out := renderGoFileFromJava(t, src)
	// The element type comes from argument 0, since a static call has no receiver.
	assertContains(t, out, "stdjava.SortWith(xs, func(a int32, b int32) int32")
}

func TestComparator_CollectionsSortNaturalOrderingIsUnchanged(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
public class NaturalSortProgram {
    public static void run() {
        List<Integer> xs = new ArrayList<Integer>();
        Collections.sort(xs);
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.SortOrdered(xs)")
}

func TestComparator_CollectionsMaxMinOverloads(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
public class MaxMinProgram {
    public static void run() {
        List<Integer> xs = new ArrayList<Integer>();
        int naturalMax = Collections.max(xs);
        int comparatorMax = Collections.max(xs, (a, b) -> a - b);
        int naturalMin = Collections.min(xs);
        int comparatorMin = Collections.min(xs, (a, b) -> a - b);
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.MaxOrdered(xs)")
	assertContains(t, out, "stdjava.MaxWith(xs, func(a int32, b int32) int32")
	assertContains(t, out, "stdjava.MinOrdered(xs)")
	assertContains(t, out, "stdjava.MinWith(xs, func(a int32, b int32) int32")
}

func TestComparator_ListSortTypesTheLambdaFromTheReceiver(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.List;
public class ListSortProgram {
    public static void run() {
        List<String> names = new ArrayList<String>();
        names.sort((a, b) -> a.compareTo(b));
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.SortWith(names, func(a string, b string) int32")
}

func TestComparator_ArraysSortOverloads(t *testing.T) {
	src := `
import java.util.Arrays;
public class ArraySortProgram {
    public static void run() {
        String[] words = new String[2];
        Arrays.sort(words);
        Arrays.sort(words, (a, b) -> a.length() - b.length());
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.SortArray(words)")
	assertContains(t, out, "stdjava.SortArrayWith(words, func(a string, b string) int32")
}

// A Comparator local must carry the named runtime type, not the unnamed func
// type Go would infer from the literal, or its methods are unreachable.
func TestComparator_LocalCarriesTheNamedRuntimeType(t *testing.T) {
	src := `
import java.util.Comparator;
public class ComparatorLocalProgram {
    public static void run() {
        Comparator<Integer> ascending = (a, b) -> a - b;
        Comparator<Integer> descending = ascending.reversed();
        int result = ascending.compare(1, 2);
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.Comparator[int32](func(a int32, b int32) int32")
	assertContains(t, out, "ascending.Reversed()")
	assertContains(t, out, "ascending.Compare(")
}

// thenComparing is overloaded on Comparator vs key extractor; the two are told
// apart by the lambda's parameter count.
func TestComparator_ThenComparingSelectsTheRightOverload(t *testing.T) {
	src := `
import java.util.Comparator;
public class ThenComparingProgram {
    public static void run() {
        Comparator<String> byLength = (a, b) -> a.length() - b.length();
        Comparator<String> byLengthThenText = byLength.thenComparing((a, b) -> a.compareTo(b));
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "byLength.ThenComparing(")
}

func TestComparator_NaturalAndReverseOrderFactories(t *testing.T) {
	src := `
import java.util.Comparator;
public class OrderFactoryProgram {
    public static void run() {
        Comparator<Integer> ascending = Comparator.naturalOrder();
        Comparator<Integer> descending = Comparator.reverseOrder();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.NaturalOrder()")
	assertContains(t, out, "stdjava.ReverseOrder()")
}

// A class implementing Comparable must not embed a nonexistent Comparable type;
// its generated CompareTo satisfies the contract structurally.
func TestComparable_IsNotEmbeddedInTheGeneratedStruct(t *testing.T) {
	src := `
public class Version implements Comparable<Version> {
    int major;
    public int compareTo(Version other) { return this.major - other.major; }
}
`
	out := renderGoFileFromJava(t, src)
	assertNotContains(t, out, "Comparable[")
	assertContains(t, out, "CompareTo(other *Version) int32")
}
