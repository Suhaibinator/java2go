import java.util.Objects;
public final class ObjectsHashProbe {
    static String trace = "";
    static Object[] live;
    static int mark(int value) { trace += value; return value; }
    static class Key {
        int value;
        Key(int value) { this.value = value; }
        @Override public int hashCode() { trace += "K"; return value; }
    }
    static final class ChangedKey extends Key {
        ChangedKey(int value) { super(value); }
        @Override public int hashCode() { trace += "D"; return value + 1; }
    }
    static final class MutatingKey extends Key {
        MutatingKey(int value) { super(value); }
        @Override public int hashCode() { trace += "M"; live[1] = mark(9); return value; }
    }
    static final class BadKey extends Key {
        BadKey() { super(0); }
        @Override public int hashCode() { trace += "B"; throw new ArithmeticException(); }
    }
    public static String run() {
        String answer = "";
        trace = "";
        int expanded = Objects.hash(mark(1), new Key(mark(2)), mark(3));
        answer += "expanded:" + expanded + ":" + trace;
        trace = "";
        Key key = new ChangedKey(7);
        int dynamic = Objects.hash(key, (Object) null, "Ω😀");
        answer += "|dynamic:" + dynamic + ":" + trace;
        trace = "";
        live = new Object[] {new MutatingKey(4), Integer.valueOf(2)};
        int changed = Objects.hash(live);
        answer += "|live:" + changed + ":" + trace + ":" + live[1];
        trace = "";
        try { Objects.hash(new BadKey(), new Key(5)); }
        catch (ArithmeticException failure) { trace += "C"; }
        finally { trace += "F"; }
        answer += "|abrupt:" + trace;
        int[] primitive = new int[] {1, 2};
        Object[] references = new Object[] {Integer.valueOf(1), null, "Ω😀"};
        String[] words = new String[] {"one", "two"};
        boolean primitiveIdentity = Objects.hash((Object) primitive) == 31 + primitive.hashCode();
        boolean referenceIdentity = Objects.hash((Object) references) == 31 + references.hashCode();
        boolean fixedAlias = Objects.hash(references) == Objects.hash(1, (Object) null, "Ω😀");
        boolean covariance = Objects.hash(words) == Objects.hash("one", "two");
        boolean singleDifferent = Objects.hash("one") == 31 + "one".hashCode();
        answer += "|arrays:" + primitiveIdentity + ":" + referenceIdentity + ":" + fixedAlias + ":" + covariance + ":" + singleDifferent;
        answer += "|null:" + Objects.hash((Object[]) null) + ":" + Objects.hash((Object) null) + ":" + Objects.hash() + ":" + Objects.hash(null);
        answer += "|boxed:" + Objects.hash((byte) -1, (short) 2, 'Ω', 3, 4L, true, Double.parseDouble("NaN"), Double.parseDouble("-0.0"), Float.parseFloat("NaN"), Float.parseFloat("-0.0"));
        answer += "|overflow:" + Objects.hash(Integer.MAX_VALUE, Integer.MIN_VALUE, Long.MIN_VALUE);
        return answer;
    }
}
