package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestBoxingUnboxingBoundsRetainDeclarationIdentity(t *testing.T) {
	outer := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "java.lang.Long"}})
	inner := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "java.lang.Integer"}})
	dependent := symbol.NewTypeParam("U", []symbol.JavaType{{Original: "T", TypeParameterBindings: map[string]*symbol.TypeParamDeclaration{"T": outer.Declaration}}})
	number := symbol.NewTypeParam("N", []symbol.JavaType{{Original: "java.lang.Number"}})
	ctx := Ctx{syntheticTypeParameters: []symbol.TypeParam{outer, inner, dependent, number}}
	for _, test := range []struct{ source, want string }{{"T", "int"}, {"U", "long"}, {"N", ""}} {
		got, _ := javaUnboxingPrimitive(test.source, ctx)
		if got != test.want {
			t.Errorf("unboxing %s = %q, want %q", test.source, got, test.want)
		}
	}
	shadow := symbol.NewTypeParam("Integer", nil)
	ctx.syntheticTypeParameters = []symbol.TypeParam{shadow}
	if _, builtin := builtinJavaWrapperPrimitive("Integer", ctx); builtin {
		t.Fatal("a type parameter named Integer must shadow the builtin wrapper")
	}
}

func TestBoxingLoweringWrapperBoundReturnAndInvocation(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class WrapperBoundConversions {
    static <T extends Integer> long widen(T value) { return value; }
    static <U extends Integer, T extends U> int dependent(T value) { return value; }
    static String take(long value) { return "long:" + value; }
    static <T extends Integer> String call(T value) { return take(value); }
    static <T extends Boolean> boolean condition(T value) { if (value) return true; return false; }
    public static String run() {
        return widen(3) + ":" + dependent(4) + ":" + call(5) + ":" + condition(true);
    }
}
`)
	runGoTestInTempModule(t, out, `package main
import "testing"
func TestWrapperBoundConversions(t *testing.T) {
    if got := Run(); got != "3:4:long:5:true" { t.Fatalf("Run() = %q", got) }
}
`)
}

func TestBoxingLoweringArraysSnapshotEarlierReads(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class BoxedArraySequencing {
    public static String run() {
        Integer value = 1000;
        Integer[] references = new Integer[] { value, value++ };
        boolean same = references[0] == references[1];
        int index = 0;
        int[] primitive = new int[] { 7, 8 };
        primitive[index] = index++;
        Integer[] array = new Integer[] { 1 };
        Integer[] original = array;
        array[0] = (array = new Integer[] { 2 })[0];
        int dimension = 1;
        int[][] matrix = new int[dimension][dimension++];
        return same + ":" + primitive[0] + ":" + primitive[1] + ":" + original[0] + ":" + matrix.length + ":" + matrix[0].length;
    }
}
`)
	runGoTestInTempModule(t, out, `package main
import "testing"
func TestBoxedArraySequencing(t *testing.T) {
    if got := Run(); got != "true:0:8:2:1:1" { t.Fatalf("Run() = %q", got) }
}
`)
}

func TestBoxingInvocationRejectsNarrowingAndWideningThenBoxing(t *testing.T) {
	for _, test := range []struct{ expression, target string }{
		{"1", "Byte"},
		{"1", "Short"},
		{"1", "Long"},
		{"1", "Double"},
		{"1L", "Integer"},
		{"Integer.valueOf(1)", "short"},
		{"Integer.valueOf(1)", "Long"},
	} {
		t.Run(test.expression+"_to_"+test.target, func(t *testing.T) {
			helper := setupParseHelper(t, "class Conversion { Object run() { return "+test.expression+"; } }")
			expression := findNode(helper.File.Ast, "return_statement").NamedChild(0)
			if _, _, applicable := javaInvocationConversionCost(expression, test.target, nil, helper.Ctx, helper.File.Source); applicable {
				t.Fatalf("strict invocation unexpectedly permits %s -> %s", test.expression, test.target)
			}
			if _, _, applicable := javaLooseInvocationConversionCost(expression, test.target, nil, helper.Ctx, helper.File.Source); applicable {
				t.Fatalf("loose invocation unexpectedly permits %s -> %s", test.expression, test.target)
			}
		})
	}
}

func TestBoxingLoweringAssignmentsReturnsAndLoops(t *testing.T) {
	out := renderGoFileFromJava(t, `
import java.util.Arrays;
import java.util.List;
public class BoxedConversionsProgram {
    static Integer field;
    static Integer pass(Integer value) { return value; }
    static Integer number() { return 7; }
    static int primitive(Integer value) { return value; }
    public static String run() {
        Boolean yes = true;
        Byte b = 2;
        Short s = 3;
        Character c = 'A';
        Integer i = number();
        Long l = 9L;
        Float f = 1.5F;
        Double d = 2.5;
        Object erased = i;
        boolean identity = ((Integer) erased) == i;
        erased = "different";
        Number numeric = i;
        numeric = l;
        int total = primitive(pass(4));
        List<Integer> values = Arrays.asList(1, 2, 3);
        for (int value : values) total += value;
        Integer[] array = new Integer[] { i, null };
        Integer index = 0;
        total += array[index];
        int[] sized = new int[index + 1];
        if (yes) total += b + s;
        boolean absent = field == null;
        Character missing = null;
        return identity + ":" + absent + ":" + total + ":" + c + ":" + missing + ":" + l + ":" + f + ":" + d + ":" + sized.length;
    }
}
`)
	runGoTestInTempModule(t, out, `package main
import "testing"
func TestBoxedConversions(t *testing.T) {
    if got := Run(); got != "true:true:22:A:null:9:1.5:2.5:1" { t.Fatalf("Run() = %q", got) }
}
`)
}

func TestBoxingLoweringOverloadPhasesAndConstructorSelection(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class BoxedOverloadProgram {
    String selected;
    BoxedOverloadProgram(long value) { selected = "long"; }
    BoxedOverloadProgram(Integer value) { selected = "integer"; }
    static String wide(long value) { return "wide"; }
    static String wide(Integer value) { return "boxed"; }
    static String reference(Object value) { return "object"; }
    static String reference(long value) { return "unboxed"; }
    static String loose(Integer value) { return "integer"; }
    static String loose(int... values) { return "varargs"; }
    static String numeric(Number value) { return "number"; }
    static String numeric(Object value) { return "object"; }
    public static String run() {
        Integer value = 3;
        return wide(1) + ":" + reference(value) + ":" + loose(2) + ":" + numeric(3) + ":" + new BoxedOverloadProgram(1).selected + ":" + new BoxedOverloadProgram(value).selected;
    }
}
`)
	runGoTestInTempModule(t, out, `package main
import "testing"
func TestBoxedOverloads(t *testing.T) {
    if got := Run(); got != "wide:object:integer:number:long:integer" { t.Fatalf("Run() = %q", got) }
}
`)
}

func TestBoxingLoweringNullViewsAndUnboxingOrder(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class BoxedNullProgram {
    static String log = "";
    static int later() { log += "later"; return 1; }
    static void take(int first, int second) { log += "take"; }
    static Integer missing() { return null; }
    public static String run() {
        Integer value = missing();
        Object erased = value;
        boolean instance = erased instanceof Integer;
        if (erased instanceof Integer bound) log += "bound";
        Integer roundTrip = (Integer) erased;
        try { take(value, later()); } catch (NullPointerException ex) { log += "null"; }
        return instance + ":" + (roundTrip == null) + ":" + log;
    }
}
`)
	runGoTestInTempModule(t, out, `package main
import "testing"
func TestBoxedNulls(t *testing.T) {
    if got := Run(); got != "false:true:null" { t.Fatalf("Run() = %q", got) }
}
`)
}
