package probe.shadow;
public final class Long {
    private static int calls;
    public static long parseLong(String text, int radix) { calls++; return text.length() + radix; }
    public static int calls() { return calls; }
}
