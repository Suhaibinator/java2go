package transpiler

import "testing"

func TestBoxedMethodReferenceOverloadPhases(t *testing.T) {
	out := assertGeneratedCompiles(t, `
interface ReferenceBoxedChoice { String apply(Integer value); }
interface ReferencePrimitiveChoice { String apply(int value); }
interface ReferenceUnboundChoice { String apply(ReferenceOverloads receiver, Integer value); }
interface ReferenceConstructorChoice { ReferenceOverloads make(Integer value); }
class ReferenceOverloads {
    String selected;
    ReferenceOverloads(int value) { selected = "constructor primitive"; }
    ReferenceOverloads(Number value) { selected = "constructor reference"; }
    static String widening(Integer value) { return "boxed"; }
    static String widening(long value) { return "long"; }
    static String strict(int value) { return "primitive"; }
    static String strict(Object value) { return "object"; }
    static String specific(Object value) { return "object"; }
    static String specific(Number value) { return "number"; }
    static String loose(Object... values) { return "varargs"; }
    static String loose(Integer value) { return "boxed"; }
    String instance(int value) { return "primitive"; }
    String instance(Number value) { return "number"; }
}
public class BoxedOverloadReferences {
    public static String run() {
        ReferencePrimitiveChoice widening = ReferenceOverloads::widening;
        ReferenceBoxedChoice strict = ReferenceOverloads::strict;
        ReferenceBoxedChoice specific = ReferenceOverloads::specific;
        ReferencePrimitiveChoice loose = ReferenceOverloads::loose;
        ReferenceOverloads receiver = new ReferenceOverloads(1);
        ReferenceBoxedChoice bound = receiver::instance;
        ReferenceUnboundChoice unbound = ReferenceOverloads::instance;
        ReferenceConstructorChoice constructor = ReferenceOverloads::new;
        return widening.apply(1) + ":" + strict.apply(1) + ":" + specific.apply(1)
                + ":" + loose.apply(1) + ":" + bound.apply(1) + ":"
                + unbound.apply(receiver, 1) + ":" + constructor.make(1).selected;
    }
}
`)
	runGeneratedWithStdjava(t, out, `
package main
import "testing"
func TestOverloadedReferences(t *testing.T) {
    if got := Run(); got != "long:object:number:boxed:number:number:constructor reference" {
        t.Fatalf("Run() = %q", got)
    }
}
`)
}

func TestBoxedMethodReferencesInferGenericTargetsAndKeepVarargsArrays(t *testing.T) {
	out := assertGeneratedCompiles(t, `
import java.util.function.Supplier;
import java.util.stream.Stream;
interface GenericReferencePrimitive { int apply(int value); }
interface GenericReferencePair { Integer apply(Integer first, Integer second); }
interface GenericReferenceNumberPair { Number apply(Integer first, Long second); }
interface GenericReferenceArray { Integer[] apply(Integer[] values); }

public class GenericBoxedReferences {
    static <T> T identity(T value) { return value; }
    static <T> T first(T... values) { return values[0]; }
    static <T> T[] array(T... values) { return values; }
    static <T extends Number> T second(T first, T second) { return second; }
    static <T> T missing() { return null; }
    public static String run() {
        GenericReferencePrimitive primitive = GenericBoxedReferences::identity;
        GenericReferencePrimitive explicit = GenericBoxedReferences::<Integer>identity;
        GenericReferencePair pair = GenericBoxedReferences::first;
        GenericReferenceArray array = GenericBoxedReferences::array;
        GenericReferenceNumberPair number = GenericBoxedReferences::second;
        Supplier<Integer> missing = GenericBoxedReferences::missing;
        Integer first = new Integer(300);
        Integer second = new Integer(400);
        Integer[] values = new Integer[]{first, second};
        Integer fromStream = Stream.of(first).map(GenericBoxedReferences::identity).findFirst().get();
        return primitive.apply(9) + ":" + explicit.apply(8) + ":"
                + (pair.apply(first, second) == first) + ":"
                + (array.apply(values) == values) + ":"
                + (array.apply(null) == null) + ":" + number.apply(first, 7L).longValue()
                + ":" + (missing.get() == null) + ":" + (fromStream == first);
    }
}
`)
	runGeneratedWithStdjava(t, out, `
package main
import "testing"
func TestGenericReferences(t *testing.T) {
    if got := Run(); got != "9:8:true:true:true:7:true:true" {
        t.Fatalf("Run() = %q", got)
    }
}
`)
}

func TestBoxedMethodReferencesAdaptSourceMethodsAndPreserveBoundTiming(t *testing.T) {
	out := assertGeneratedCompiles(t, `
import java.util.function.Function;
import java.util.function.Supplier;
import java.util.function.ToIntFunction;

interface BoxedReferenceMapper { Integer apply(Integer value); }
interface PrimitiveReferenceMapper { int apply(int value); }
interface BoxedReferenceSupplier { Integer get(); }
interface UnboundReferenceMapper { Integer apply(ReferenceTarget receiver, Integer value); }

class ReferenceTarget {
    static int captures = 0;
    int stored;
    ReferenceTarget(int value) { stored = value; }
    int primitive(int value) { return value + stored; }
    Integer boxed(Integer value) { return value + stored; }
    int read() { return stored; }
    static int staticPrimitive(int value) { return value + 1; }
    static Integer staticBoxed(Integer value) { return value + 2; }
    static ReferenceTarget capture() { captures++; return new ReferenceTarget(10); }
}

public class BoxedSourceReferences {
    public static String run() {
        BoxedReferenceMapper staticBoxed = ReferenceTarget::staticPrimitive;
        PrimitiveReferenceMapper staticPrimitive = ReferenceTarget::staticBoxed;
        ReferenceTarget receiver = new ReferenceTarget(3);
        BoxedReferenceMapper bound = receiver::primitive;
        UnboundReferenceMapper unbound = ReferenceTarget::primitive;
        PrimitiveReferenceMapper boundPrimitive = receiver::boxed;
        BoxedReferenceSupplier captured = ReferenceTarget.capture()::read;
        int captures = ReferenceTarget.captures;
        Integer first = captured.get();
        Integer second = captured.get();
        Function<Integer, Integer> builtinStatic = ReferenceTarget::staticPrimitive;
        Function<Integer, Integer> builtinBound = receiver::primitive;
        Supplier<Integer> builtinSupplier = receiver::read;
        ToIntFunction<Integer> builtinUnboxResult = receiver::boxed;
        boolean caught = false;
        try {
            ReferenceTarget missing = null;
            BoxedReferenceSupplier neverCalled = missing::read;
        } catch (NullPointerException expected) {
            caught = true;
        }
        return staticBoxed.apply(4) + ":" + staticPrimitive.apply(4) + ":"
                + bound.apply(4) + ":" + unbound.apply(receiver, 4) + ":"
                + boundPrimitive.apply(4) + ":" + captures + ":"
                + first + ":" + second + ":" + ReferenceTarget.captures + ":"
                + builtinStatic.apply(4) + ":" + builtinBound.apply(4) + ":"
                + builtinSupplier.get() + ":" + builtinUnboxResult.applyAsInt(4) + ":" + caught;
    }
}
`)
	runGeneratedWithStdjava(t, out, `
package main
import "testing"
func TestAdaptedSourceReferences(t *testing.T) {
    if got := Run(); got != "5:6:7:7:7:1:10:10:1:5:7:3:7:true" {
        t.Fatalf("Run() = %q", got)
    }
}
`)
}

func TestBoxedMethodReferencesRuntimeFactoriesAndComparable(t *testing.T) {
	out := assertGeneratedCompiles(t, `
interface ReferenceIntBoxer { Integer apply(int value); }
interface ReferenceStringBoxer { Integer apply(String value); }
interface ReferenceUnboxer { int apply(Integer value); }
interface ReferenceLength { Integer apply(String value); }
interface ReferenceBoundInt { int get(); }
interface ReferenceRawCompare { Integer apply(Comparable receiver, Object other); }
interface ReferenceTypedCompare { Integer apply(Comparable<Integer> receiver, Integer other); }

public class BoxedRuntimeReferences {
    static int captures = 0;
    static Integer capture() { captures++; return new Integer(300); }
    public static String run() {
        ReferenceIntBoxer cached = Integer::valueOf;
        ReferenceIntBoxer fresh = Integer::new;
        ReferenceStringBoxer parsed = Integer::valueOf;
        ReferenceStringBoxer constructed = Integer::new;
        ReferenceUnboxer unbox = Integer::intValue;
        ReferenceLength length = String::length;
        ReferenceBoundInt bound = capture()::intValue;
        ReferenceRawCompare raw = Comparable::compareTo;
        ReferenceTypedCompare typed = Comparable<Integer>::compareTo;
        boolean caught = false;
        try {
            Integer missing = null;
            ReferenceBoundInt neverCalled = missing::intValue;
        } catch (NullPointerException expected) { caught = true; }
        boolean castCaught = false;
        try { raw.apply(1, 1L); }
        catch (ClassCastException expected) { castCaught = true; }
        return (cached.apply(7) == cached.apply(7)) + ":"
                + (fresh.apply(7) == fresh.apply(7)) + ":"
                + (parsed.apply("7") == cached.apply(7)) + ":"
                + (constructed.apply("7") == cached.apply(7)) + ":"
                + unbox.apply(9) + ":" + length.apply("abc") + ":"
                + bound.get() + ":" + bound.get() + ":" + captures + ":"
                + raw.apply(1, 2) + ":" + typed.apply(3, 2) + ":" + caught + ":" + castCaught;
    }
}
`)
	runGeneratedWithStdjava(t, out, `
package main
import "testing"
func TestRuntimeReferences(t *testing.T) {
    if got := Run(); got != "true:false:true:false:9:3:300:300:1:-1:1:true:true" {
        t.Fatalf("Run() = %q", got)
    }
}
`)
}
