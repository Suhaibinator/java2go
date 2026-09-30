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
	assertContains(t, out, "stdjava.NewList[*stdjava.JavaString]()")
	assertContains(t, out, "xs.Add(stdjava.JavaStringLiteralUTF16([]uint16{97}))")
	assertContains(t, out, "xs.Get(0)")
	assertContains(t, out, "xs.Size()")
	assertContains(t, out, "xs.Contains(stdjava.JavaStringLiteralUTF16([]uint16{97}), __java2goExecution)")
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
	assertContains(t, out, "a *stdjava.List[*stdjava.JavaString]")
	assertContains(t, out, "b *stdjava.Map[*stdjava.JavaString, *stdjava.Integer]")
	// A Java Set declaration stays a nominal iterable interface, so a source
	// implementation or native set can occupy this field without changing it.
	assertContains(t, out, "c stdjava.JavaIterable")
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
	assertContains(t, out, "range stdjava.CollectionIterationElements(xs)")
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
	assertContains(t, out, "stdjava.NewMap[*stdjava.JavaString, *stdjava.Integer]()")
	assertContains(t, out, "stdjava.MapPutExecution(__java2goExecution, m, stdjava.JavaStringLiteralUTF16([]uint16{107}), stdjava.BoxInteger(int32(1)))")
	assertContains(t, out, "stdjava.UnboxInteger(stdjava.ObjectView[*stdjava.Integer](m.GetObject(stdjava.JavaStringLiteralUTF16([]uint16{107}), __java2goExecution), stdjava.IntegerTypeID))")
	assertContains(t, out, "m.ContainsKey(stdjava.JavaStringLiteralUTF16([]uint16{107}), __java2goExecution)")
	flat := normalizeSpaces(out)
	newMap := strings.Index(flat, "m := stdjava.NewMap[*stdjava.JavaString, *stdjava.Integer]()")
	put := strings.Index(flat, "stdjava.MapPutExecution(__java2goExecution, m,")
	get := strings.Index(flat, "m.GetObject(stdjava.JavaStringLiteralUTF16([]uint16{107}), __java2goExecution)")
	contains := strings.Index(flat, "m.ContainsKey(stdjava.JavaStringLiteralUTF16([]uint16{107}), __java2goExecution)")
	if newMap < 0 || put <= newMap || get <= put || contains <= get {
		t.Fatalf("map construction, put, get, and containsKey changed Java execution order:\n%s", out)
	}
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
	assertContains(t, out, "stdjava.SortOrdered(xs, __java2goExecution)")
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
	assertContains(t, out, "stdjava.MapPutExecution(__java2goExecution, map_, stdjava.JavaStringLiteralUTF16([]uint16{97}), stdjava.BoxInteger(int32(1)))")
	assertContains(t, out, "stdjava.UnboxInteger(stdjava.ObjectView[*stdjava.Integer](map_.GetObject(stdjava.JavaStringLiteralUTF16([]uint16{97}), __java2goExecution), stdjava.IntegerTypeID))")
	flat := normalizeSpaces(out)
	declaration := strings.Index(flat, "map_ := stdjava.NewMap[*stdjava.JavaString, *stdjava.Integer]()")
	put := strings.Index(flat, "stdjava.MapPutExecution(__java2goExecution, map_,")
	get := strings.Index(flat, "map_.GetObject(stdjava.JavaStringLiteralUTF16([]uint16{97}), __java2goExecution)")
	if declaration < 0 || put <= declaration || get <= put {
		t.Fatalf("sanitized map receiver changed identity or Java execution order:\n%s", out)
	}
}

func TestOptional_LambdaAndTypeInference(t *testing.T) {
	src := `
import java.util.Optional;
public class OptProgram {
    static Optional<String> find(int id) {
        if (id == 1) {
            return Optional.of("a");
        }
        return Optional.empty();
    }
    public static int run() {
        Optional<Integer> num = Optional.of(10);
        return num.map(n -> n * 2).get();
    }
}
`
	out := renderGoFileFromJava(t, src)
	// empty() in return position gets its element type from the method return type.
	assertContains(t, out, "stdjava.OptionalEmpty[*stdjava.JavaString]()")
	// of(10) stores a boxed Java Integer inferred from Optional<Integer>.
	assertContains(t, out, "stdjava.OptionalOf[*stdjava.Integer](stdjava.BoxInteger(int32(10)))")
	// map's lambda is re-typed from the element type and the chained .get() resolves.
	assertContains(t, out, "stdjava.FunctionCallbackExecution[*stdjava.Integer, *stdjava.Integer](__java2goExecution, stdjava.NewFunctionFuncAdapter[*stdjava.Integer, *stdjava.Integer](func(__java2goExecution *stdjava.Execution, n *stdjava.Integer) *stdjava.Integer")
	assertContains(t, out, "return stdjava.BoxInteger(int32(stdjava.UnboxInteger(n) * 2))")
	assertContains(t, out, ").Get()")
	assertContains(t, out, "stdjava.UnboxInteger(stdjava.OptionalMap")
}

func TestStringConcat_InfersStringType(t *testing.T) {
	// A var/local initialized from a string concatenation is a String, so String
	// intrinsics dispatch on it.
	src := `
public class ConcatProgram {
    public static int run() {
        var g = "ab" + "cd";
        String h = "x" + 5;
        return g.length() + h.length();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.RequireJavaString(g).Length()")
	assertContains(t, out, "stdjava.RequireJavaString(h).Length()")
}
