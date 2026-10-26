package probe.shadow;
public final class Paths {
    private static int calls;
    public static String get(String first, String... more) { calls++; return first; }
    public static int calls() { return calls; }
}
