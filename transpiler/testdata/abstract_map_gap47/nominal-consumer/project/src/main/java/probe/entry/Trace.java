package probe.entry;
public final class Trace {
    private static String events = "";
    public static void add(String event) { events += event + ","; }
    public static String take() { String found = events; events = ""; return found; }
}
