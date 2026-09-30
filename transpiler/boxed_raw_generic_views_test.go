package transpiler

import "testing"

func TestBoxedRawGenericNumberViews_PreserveIdentityAndCastTiming(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class BoxedRawNumberViews {
    static int reads;
    static class Box<T extends Number> {
        T value;
        Box(T value) { this.value = value; }
        T read() { reads++; return value; }
        void write(T value) { this.value = value; }
        int throughBound() { return value.intValue(); }
    }
    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Box<Integer> typed = new Box<Integer>(1);
        Box raw = typed;
        Long replacement = Long.valueOf(8);
        raw.write(replacement);
        Number fieldView = typed.value;
        Object objectView = typed.value;
        Number resultView = typed.read();
        String fieldCast;
        try { Integer exact = typed.value; fieldCast = "bad"; }
        catch (ClassCastException expected) { fieldCast = "cast"; }
        String resultCast;
        try { Integer exact = typed.read(); resultCast = "bad"; }
        catch (ClassCastException expected) { resultCast = "cast"; }
        String memberCast;
        try { int exact = typed.value.intValue(); memberCast = "bad"; }
        catch (ClassCastException expected) { memberCast = "cast"; }
        return (fieldView == replacement) + ":" + (objectView == replacement)
            + ":" + (resultView == replacement) + ":" + fieldCast + ":" + resultCast + ":" + reads
            + ":" + typed.throughBound() + ":" + memberCast;
    }
}
`, "true:true:true:cast:cast:2:8:cast")
}

func TestBoxedRawGenericComparableViews_PreserveIdentityAndCastTiming(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class BoxedRawComparableViews {
    static class Box<T extends Comparable<T>> {
        T value;
        Box(T value) { this.value = value; }
        T read() { return value; }
        void write(T value) { this.value = value; }
    }
    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Box<Integer> typed = new Box<Integer>(1);
        Box raw = typed;
        String replacement = "next";
        raw.write(replacement);
        Object fieldView = typed.value;
        Object resultView = typed.read();
        String fieldCast;
        try { Integer exact = typed.value; fieldCast = "bad"; }
        catch (ClassCastException expected) { fieldCast = "cast"; }
        String resultCast;
        try { Integer exact = typed.read(); resultCast = "bad"; }
        catch (ClassCastException expected) { resultCast = "cast"; }
        return (fieldView == replacement) + ":" + (resultView == replacement)
            + ":" + fieldCast + ":" + resultCast;
    }
}
`, "true:true:cast:cast")
}
