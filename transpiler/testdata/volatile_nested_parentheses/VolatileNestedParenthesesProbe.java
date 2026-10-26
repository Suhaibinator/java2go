public class VolatileNestedParenthesesProbe {
    volatile int count = 3;
    volatile VolatileNestedParenthesesProbe next;
    static volatile int global = 5;
    static int receivers;
    static int rhsCalls;
    static int cleaned;
    static String trace = "";

    static VolatileNestedParenthesesProbe receiver(VolatileNestedParenthesesProbe value) {
        receivers++;
        trace += "R";
        return value;
    }
    static int rhs(int value) {
        rhsCalls++;
        trace += "H";
        return value;
    }
    static int failingRhs() {
        rhsCalls++;
        trace += "H";
        throw new ArithmeticException("rhs");
    }
    static void reset() {
        receivers = 0;
        rhsCalls = 0;
        cleaned = 0;
        trace = "";
    }
    static String observation(String label, int value) {
        return label + "=" + value + "@" + trace + ":" + receivers + ":" + rhsCalls + ":" + cleaned + ";";
    }
    public static String run() {
        VolatileNestedParenthesesProbe value = new VolatileNestedParenthesesProbe();
        String result = "";
        reset();
        result += observation("read", ((receiver(value).count)));
        reset();
        int assigned = (((receiver(value).count)) = rhs(7));
        result += observation("assign", assigned) + ((value.count)) + ";";
        reset();
        int compound = (((receiver(value).count)) += rhs(4));
        result += observation("compound", compound) + ((value.count)) + ";";
        reset();
        int post = ((receiver(value).count))++;
        result += observation("post", post) + ((value.count)) + ";";
        reset();
        int pre = ++((receiver(value).count));
        result += observation("pre", pre) + ((value.count)) + ";";
        reset();
        value.next = value;
        int nestedReceiver = (((value.next.count)) += rhs(2));
        result += observation("nestedReceiver", nestedReceiver) + ((value.next.count)) + ";";
        reset();
        result += observation("static", ((global)));

        reset();
        try { ((receiver(null).count)) = rhs(17); }
        catch (NullPointerException failure) { trace += "N"; }
        finally { cleaned++; trace += "F"; }
        result += observation("nullAssign", 0);
        reset();
        try { ((receiver(null).count)) += rhs(19); }
        catch (NullPointerException failure) { trace += "N"; }
        finally { cleaned++; trace += "F"; }
        result += observation("nullCompound", 0);
        reset();
        try { ((receiver(null).count))++; }
        catch (NullPointerException failure) { trace += "N"; }
        finally { cleaned++; trace += "F"; }
        result += observation("nullPost", 0);
        reset();
        try { ++((receiver(null).count)); }
        catch (NullPointerException failure) { trace += "N"; }
        finally { cleaned++; trace += "F"; }
        result += observation("nullPre", 0);
        reset();
        try { ((receiver(null).count)) = failingRhs(); }
        catch (NullPointerException failure) { trace += "N"; }
        catch (ArithmeticException failure) { trace += "A"; }
        finally { cleaned++; trace += "F"; }
        result += observation("throwingSimpleRhs", 0);
        reset();
        try { ((receiver(null).count)) += failingRhs(); }
        catch (NullPointerException failure) { trace += "N"; }
        catch (ArithmeticException failure) { trace += "A"; }
        finally { cleaned++; trace += "F"; }
        return result + observation("throwingCompoundRhs", 0);
    }
}
