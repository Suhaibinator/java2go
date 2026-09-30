package transpiler

import "testing"

func TestBoxedObjectSemanticsNullUnboxingAndNominalViews(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class BoxedNullAndViewsProgram {
    static String trace = "";
    static void booleanSink(boolean value) { trace += "!"; }
    static void byteSink(byte value) { trace += "!"; }
    static void shortSink(short value) { trace += "!"; }
    static void characterSink(char value) { trace += "!"; }
    static void integerSink(int value) { trace += "!"; }
    static void longSink(long value) { trace += "!"; }
    static void floatSink(float value) { trace += "!"; }
    static void doubleSink(double value) { trace += "!"; }

    public static String run() {
        trace = "";
        Boolean flag = null;
        Byte small = null;
        Short narrow = null;
        Character character = null;
        Integer integer = null;
        Long wide = null;
        Float single = null;
        Double decimal = null;
        try { booleanSink(flag); } catch (NullPointerException expected) { trace += "Z"; }
        try { byteSink(small); } catch (NullPointerException expected) { trace += "B"; }
        try { shortSink(narrow); } catch (NullPointerException expected) { trace += "S"; }
        try { characterSink(character); } catch (NullPointerException expected) { trace += "C"; }
        try { integerSink(integer); } catch (NullPointerException expected) { trace += "I"; }
        try { longSink(wide); } catch (NullPointerException expected) { trace += "J"; }
        try { floatSink(single); } catch (NullPointerException expected) { trace += "F"; }
        try { doubleSink(decimal); } catch (NullPointerException expected) { trace += "D"; }

        Object notNumber = Character.valueOf('A');
        try {
            Number wrong = (Number) notNumber;
            trace += "!";
        } catch (ClassCastException expected) {
            trace += ":cast";
        }
        Comparable raw = Integer.valueOf(4);
        try {
            raw.compareTo(Long.valueOf(4));
            trace += "!";
        } catch (ClassCastException expected) {
            trace += ":compare";
        }
        try {
            raw.compareTo(null);
            trace += "!";
        } catch (NullPointerException expected) {
            trace += ":null";
        }
        Integer original = new Integer(1000);
        Number number = original;
        Object object = number;
        Number missing = (Number) integer;
        return trace + ":" + ((Integer) object == original) + ":" + (missing == null)
                + ":" + (object instanceof Integer) + ":" + (object instanceof Number)
                + ":" + (object instanceof Comparable) + ":" + !(notNumber instanceof Number);
    }
}
`)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestBoxedNullAndViewsRuntime(t *testing.T) {
    const want = "ZBSCIJFD:cast:compare:null:true:true:true:true:true:true"
    if got := Run(); got != want {
        t.Fatalf("Run() = %q, want %q", got, want)
    }
}
`)
}
