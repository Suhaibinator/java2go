package transpiler

import (
	"strings"
	"testing"
)

func TestCollections_ListConstructionAndMethods(t *testing.T) {
	src := `
import java.util.List;
import java.util.ArrayList;
public class ListProgram {
    public static void run() {
        List<String> xs = new ArrayList<String>();
        xs.add("a");
        String first = xs.get(0);
        int n = xs.size();
        boolean has = xs.contains("a");
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.NewList[string]()")
	assertContains(t, out, "xs.Add(\"a\")")
	assertContains(t, out, "xs.Get(0)")
	assertContains(t, out, "xs.Size()")
	assertContains(t, out, "xs.Contains(\"a\")")
}

func TestCollections_DeclaredTypeMapsToStdjava(t *testing.T) {
	src := `
import java.util.List;
import java.util.Map;
import java.util.Set;
public class DeclProgram {
    List<String> a;
    Map<String, Integer> b;
    Set<Long> c;
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "a *stdjava.List[string]")
	assertContains(t, out, "b *stdjava.Map[string, int32]")
	assertContains(t, out, "c *stdjava.Set[int64]")
}

func TestCollections_EnhancedForRangesOverSlice(t *testing.T) {
	src := `
import java.util.List;
import java.util.ArrayList;
public class ForProgram {
    public static void run() {
        List<String> xs = new ArrayList<String>();
        for (String s : xs) {
            System.out.println(s);
        }
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "range xs.Slice()")
}

func TestCollections_MapConstructionAndMethods(t *testing.T) {
	src := `
import java.util.Map;
import java.util.HashMap;
public class MapProgram {
    public static void run() {
        Map<String, Integer> m = new HashMap<String, Integer>();
        m.put("k", 1);
        int v = m.get("k");
        boolean has = m.containsKey("k");
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.NewMap[string, int32]()")
	assertContains(t, out, "m.Put(\"k\", 1)")
	assertContains(t, out, "m.Get(\"k\")")
	assertContains(t, out, "m.ContainsKey(\"k\")")
}

func TestCollections_StaticsAndArrays(t *testing.T) {
	src := `
import java.util.List;
import java.util.ArrayList;
import java.util.Collections;
public class StaticsProgram {
    public static void run() {
        List<Integer> xs = new ArrayList<Integer>();
        Collections.sort(xs);
        Collections.reverse(xs);
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.SortOrdered(xs)")
	assertContains(t, out, "stdjava.ReverseList(xs)")
}

func TestCollections_KeywordVariableSanitized(t *testing.T) {
	// A Java variable named `map` collides with a Go keyword and must be renamed
	// consistently at its declaration and every reference.
	src := `
import java.util.Map;
import java.util.HashMap;
public class KeywordProgram {
    public static void run() {
        Map<String, Integer> map = new HashMap<String, Integer>();
        map.put("a", 1);
        int x = map.get("a");
    }
}
`
	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "map :=") || strings.Contains(out, "map.Put") {
		t.Fatalf("Go keyword `map` was not sanitized:\n%s", out)
	}
	assertContains(t, out, "map_ := stdjava.NewMap")
	assertContains(t, out, "map_.Put(\"a\", 1)")
	assertContains(t, out, "map_.Get(\"a\")")
}
