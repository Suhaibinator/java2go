package parity.boxedobjects;

import parity.boxedobjects.library.LibraryProbe;
import parity.boxedobjects.model.WrapperState;

@SuppressWarnings("removal")
public class BoxedObjectSemanticsApplication {
    static String trace = "";

    static class Choice {
        String kind;

        Choice(long value) {
            kind = "long";
        }

        Choice(Integer value) {
            kind = "Integer";
        }

        int accept(int value) {
            return value;
        }
    }

    static class DelegatedChoice extends Choice {
        DelegatedChoice() {
            this(1);
        }

        DelegatedChoice(int value) {
            super(value);
        }
    }

    static String select(long value) {
        return "long";
    }

    static String select(Integer value) {
        return "Integer";
    }

    static String phase(Number value) {
        return "Number";
    }

    static String phase(long value) {
        return "long";
    }

    static String spread(Object value) {
        return "Object";
    }

    static String spread(int... values) {
        return "varargs";
    }

    static Integer[] same(Integer... values) {
        return values;
    }

    static void replace(Integer... values) {
        values[0] = 9;
    }

    static void wrongStore(Object... values) {
        values[0] = Long.valueOf(9);
    }

    static Integer missing() {
        trace += "u";
        return null;
    }

    static int later() {
        trace += "a";
        return 1;
    }

    static Integer laterBox() {
        trace += "b";
        return 1;
    }

    static Choice receiver() {
        trace += "r";
        return null;
    }

    static int consume(int first, int second) {
        trace += "c";
        return first + second;
    }

    static String nullableState() {
        WrapperState state = new WrapperState();
        boolean initial = state.allNull();
        String nullValues = state.values();
        state.set(true, (byte) 7, (short) 8, 'A', 1000, 1001L, 1.5f, 2.5);
        String values = state.values();
        boolean aliases = state.flag == WrapperState.identity(state.flag)
                && state.small == WrapperState.identity(state.small)
                && state.narrow == WrapperState.identity(state.narrow)
                && state.character == WrapperState.identity(state.character)
                && state.integer == WrapperState.identity(state.integer)
                && state.wide == WrapperState.identity(state.wide)
                && state.single == WrapperState.identity(state.single)
                && state.decimal == WrapperState.identity(state.decimal);
        Object[] objects = {state.flag, state.small, state.narrow, state.character,
                state.integer, state.wide, state.single, state.decimal};
        boolean kinds = objects[0] instanceof Boolean && objects[1] instanceof Byte
                && objects[2] instanceof Short && objects[3] instanceof Character
                && objects[4] instanceof Integer && objects[5] instanceof Long
                && objects[6] instanceof Float && objects[7] instanceof Double;
        Number[] numbers = {state.small, state.narrow, state.integer,
                state.wide, state.single, state.decimal};
        int sum = 0;
        for (Number number : numbers) {
            sum += WrapperState.numberValue(number);
        }
        Integer inferred = WrapperState.identity(17);
        Number mixed = WrapperState.choose(false, Integer.valueOf(2), Long.valueOf(3));
        boolean casts = (Integer) objects[4] == state.integer
                && (Long) mixed == Long.valueOf(3)
                && ((Integer) null) == null;
        state.set(null, null, null, null, null, null, null, null);
        return initial + "/" + nullValues + "/" + values + "/" + aliases + ":"
                + kinds + ":" + sum + ":" + inferred + ":" + mixed + ":" + casts
                + ":" + state.allNull();
    }

    static String identityAndUpdates() {
        boolean cached = Boolean.valueOf(true) == Boolean.valueOf(true)
                && Byte.valueOf((byte) -128) == Byte.valueOf((byte) -128)
                && Byte.valueOf((byte) 127) == Byte.valueOf((byte) 127)
                && Short.valueOf((short) -128) == Short.valueOf((short) -128)
                && Short.valueOf((short) 127) == Short.valueOf((short) 127)
                && Character.valueOf((char) 0) == Character.valueOf((char) 0)
                && Character.valueOf((char) 127) == Character.valueOf((char) 127)
                && Integer.valueOf(-128) == Integer.valueOf(-128)
                && Integer.valueOf(127) == Integer.valueOf(127)
                && Long.valueOf(-128) == Long.valueOf(-128)
                && Long.valueOf(127) == Long.valueOf(127);
        boolean fresh = new Boolean(true) != new Boolean(true)
                && new Byte((byte) 7) != new Byte((byte) 7)
                && new Short((short) 7) != new Short((short) 7)
                && new Character('A') != new Character('A')
                && new Integer(7) != new Integer(7)
                && new Long(7) != new Long(7)
                && new Float(7) != new Float(7)
                && new Double(7) != new Double(7);
        Integer value = 1000;
        Integer alias = value;
        Integer postfix = value++;
        Integer prefix = ++value;
        value += 3;
        Integer equalValue = new Integer(1000);
        return cached + ":" + fresh + ":" + (alias == postfix) + ":" + (alias == value)
                + ":" + alias + ":" + postfix + ":" + prefix + ":" + value
                + ":" + (alias == equalValue) + ":" + alias.equals(equalValue)
                + ":" + (alias == 1000) + ":" + (1000 == alias);
    }

    static String comparisons() {
        Float floatNaN = Float.valueOf(Float.NaN);
        Float anotherFloatNaN = new Float(Float.NaN);
        Double doubleNaN = Double.valueOf(Double.NaN);
        Double anotherDoubleNaN = new Double(Double.NaN);
        Float floatNegativeZero = Float.valueOf(-0.0f);
        Float floatPositiveZero = Float.valueOf(0.0f);
        Double doubleNegativeZero = Double.valueOf(-0.0);
        Double doublePositiveZero = Double.valueOf(0.0);
        Comparable<Integer> comparable = Integer.valueOf(4);
        Comparable<String> textComparable = "b";
        boolean numbers = Byte.valueOf((byte) 4).intValue() == 4
                && Short.valueOf((short) 257).byteValue() == 1
                && Character.valueOf('\uffff').charValue() == 65535
                && Boolean.valueOf(true).booleanValue()
                && Long.valueOf(4294967297L).intValue() == 1
                && Double.valueOf(Double.POSITIVE_INFINITY).intValue() == Integer.MAX_VALUE
                && Double.valueOf(Double.NEGATIVE_INFINITY).longValue() == Long.MIN_VALUE
                && doubleNaN.intValue() == 0;
        return numbers + ":" + floatNaN.equals(anotherFloatNaN)
                + ":" + (floatNaN.hashCode() == anotherFloatNaN.hashCode())
                + ":" + doubleNaN.equals(anotherDoubleNaN)
                + ":" + (doubleNaN.hashCode() == anotherDoubleNaN.hashCode())
                + ":" + floatNegativeZero.equals(floatPositiveZero)
                + ":" + (floatNegativeZero.hashCode() == floatPositiveZero.hashCode())
                + ":" + doubleNegativeZero.equals(doublePositiveZero)
                + ":" + (doubleNegativeZero.hashCode() == doublePositiveZero.hashCode())
                + ":" + (floatNegativeZero.compareTo(floatPositiveZero) < 0)
                + ":" + (doubleNegativeZero.compareTo(doublePositiveZero) < 0)
                + ":" + (doubleNaN.compareTo(Double.valueOf(Double.POSITIVE_INFINITY)) > 0)
                + ":" + Character.valueOf('A').equals(Integer.valueOf(65))
                + ":" + (comparable.compareTo(5) < 0)
                + ":" + (textComparable.compareTo("a") > 0);
    }

    static String arraysAndConversions() {
        boolean defaults = (new Boolean[1])[0] == null && (new Byte[1])[0] == null
                && (new Short[1])[0] == null && (new Character[1])[0] == null
                && (new Integer[1])[0] == null && (new Long[1])[0] == null
                && (new Float[1])[0] == null && (new Double[1])[0] == null;
        Integer[] values = {1, null};
        boolean sameArray = same(values) == values;
        replace(values);
        boolean nullArray = same((Integer[]) null) == null;
        Integer[] expanded = same(2, 3);
        String store = "missing";
        try {
            wrongStore(values);
        } catch (ArrayStoreException expected) {
            store = "array-store";
        }
        Integer length = 2;
        Integer index = 1;
        int[] primitive = new int[length];
        primitive[index] = 7;
        int selected = 0;
        Boolean condition = true;
        if (condition) {
            switch (index) {
                case 1: selected = primitive[index]; break;
                default: selected = -1;
            }
        }
        return defaults + ":" + sameArray + ":" + values[0] + ":" + values[1]
                + ":" + nullArray + ":" + expanded.length + ":" + expanded[0]
                + ":" + expanded[1] + ":" + store + ":" + selected;
    }

    static String evaluationOrder() {
        trace = "";
        try {
            consume(missing(), later());
        } catch (NullPointerException expected) {
            trace += "n";
        }
        String unboxing = trace;
        trace = "";
        try {
            receiver().accept(later());
        } catch (NullPointerException expected) {
            trace += "n";
        }
        String receiverOrder = trace;
        trace = "";
        Integer absent = null;
        try {
            absent.compareTo(laterBox());
        } catch (NullPointerException expected) {
            trace += "n";
        }
        return unboxing + ":" + receiverOrder + ":" + trace;
    }

    static boolean classLiterals() {
        return Boolean.class != Boolean.TYPE && boolean.class == Boolean.TYPE
                && Byte.class != Byte.TYPE && byte.class == Byte.TYPE
                && Short.class != Short.TYPE && short.class == Short.TYPE
                && Character.class != Character.TYPE && char.class == Character.TYPE
                && Integer.class != Integer.TYPE && int.class == Integer.TYPE
                && Long.class != Long.TYPE && long.class == Long.TYPE
                && Float.class != Float.TYPE && float.class == Float.TYPE
                && Double.class != Double.TYPE && double.class == Double.TYPE;
    }

    public static void main(String[] args) {
        System.out.println("STATE=" + nullableState());
        System.out.println("IDENTITY=" + identityAndUpdates());
        System.out.println("COMPARE=" + comparisons());
        System.out.println("ARRAYS=" + arraysAndConversions());
        System.out.println("OVERLOAD=" + select(1) + ":" + select(Integer.valueOf(1))
                + ":" + phase(Integer.valueOf(1)) + ":" + spread(1)
                + ":" + spread(new int[] {1, 2}) + ":" + new Choice(1).kind
                + ":" + new Choice(Integer.valueOf(1)).kind + ":" + new DelegatedChoice().kind);
        System.out.println("EVALUATION=" + evaluationOrder());
        System.out.println("CLASSES=" + classLiterals());
        System.out.println("LIBRARY=" + LibraryProbe.run());
    }
}
