package transpiler

import "testing"

func TestBoxedIntrinsicFactoriesRejectInvocationNarrowing(t *testing.T) {
	for _, expression := range []string{"Byte.valueOf(1)", "Short.valueOf(1)", "Integer.valueOf(1L)", "Float.valueOf(1.0)", "new Byte(1)", "new Integer(1L)"} {
		t.Run(expression, func(t *testing.T) {
			helper := setupParseHelper(t, "class InvalidWrapperCall { Object run() { return "+expression+"; } }")
			resetDiagnostics()
			ParseExpr(findNode(helper.File.Ast, "return_statement").NamedChild(0), helper.File.Source, helper.Ctx)
			if len(Diagnostics()) == 0 {
				t.Fatalf("invalid Java invocation accepted: %s", expression)
			}
			resetDiagnostics()
		})
	}
}

func TestBoxedIntrinsicFactoriesAndCollectionBoundaries(t *testing.T) {
	out := renderGoFileFromJava(t, `
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
public class BoxedIntrinsicBoundaries {
    public static String run() {
        Integer cached = Integer.valueOf(12);
        Integer parsed = Integer.valueOf("c", 16);
        Integer fresh = new Integer("12");
        Byte small = Byte.valueOf((byte) 3);
        Short narrow = new Short("4");
        Character character = Character.valueOf('A');
        Long wide = new Long(5);
        Float single = new Float(6.5);
        Double decimal = Double.valueOf("7.5");
        Boolean flag = Boolean.valueOf("true");
        Number number = fresh;
        Object erased = fresh;
        List<Integer> list = new ArrayList<Integer>();
        list.add(1);
        list.add(2);
        boolean removed = list.remove(Integer.valueOf(1));
        boolean heterogeneous = list.remove(Long.valueOf(2));
        Integer last = list.remove(0);
        Map<Integer, Integer> map = new HashMap<Integer, Integer>();
        map.put(20, 30);
        Integer missing = map.get(Long.valueOf(20));
        Optional<Integer> absent = Optional.ofNullable(null);
        Optional<Integer> present = Optional.of(11);
        Integer fallback = present.orElse(12);
        Optional<Integer> nullMapped = present.map(n -> null);
        Integer magnitude = Math.abs(Integer.valueOf(-3));
        Long maximum = Math.max(cached, Long.valueOf(15));
        Integer rounded = Math.round(Float.valueOf(1.5f));
        return (cached == parsed) + ":" + (cached == fresh) + ":" + fresh.equals(12)
            + ":" + number.intValue() + ":" + erased.equals(cached)
            + ":" + (small.intValue() + narrow.intValue() + character.charValue() + wide.intValue())
            + ":" + single + ":" + decimal + ":" + (flag == Boolean.TRUE)
            + ":" + removed + ":" + heterogeneous + ":" + last
            + ":" + (missing == null) + ":" + absent.isEmpty() + ":" + fallback
            + ":" + nullMapped.isEmpty() + ":" + Optional.ofNullable(null).isEmpty()
            + ":" + magnitude + ":" + maximum + ":" + rounded;
    }
}
`)
	runGeneratedWithStdjava(t, out, `
package main
import "testing"
func TestBoundaries(t *testing.T) {
    const want = "true:false:true:12:true:77:6.5:7.5:true:true:false:2:true:true:11:true:true:3:15:2"
    if got := Run(); got != want { t.Fatalf("Run() = %q, want %q", got, want) }
}
`)
}

func TestBoxedIntrinsicQualifiedNamesAndPrimitiveClassLiterals(t *testing.T) {
	out := renderGoFileFromJava(t, `
class Integer {}
public class QualifiedBoxedIntrinsic {
    public static boolean run() {
        java.lang.Integer value = java.lang.Integer.valueOf(7);
        java.lang.Integer fresh = new java.lang.Integer(7);
        return value.equals(fresh) && value != fresh
            && java.lang.Integer.TYPE == int.class
            && value.getClass() == java.lang.Integer.class
            && value.getClass() != java.lang.Integer.TYPE
            && Float.MIN_VALUE > 0.0f && Double.MIN_NORMAL > Double.MIN_VALUE
            && Float.valueOf(Float.POSITIVE_INFINITY).isInfinite()
            && Double.valueOf(Double.NaN).isNaN();
    }
}
`)
	runGeneratedWithStdjava(t, out, `
package main
import "testing"
func TestQualified(t *testing.T) { if !Run() { t.Fatal("qualified wrapper identity") } }
`)
}
