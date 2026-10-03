package probe.state;
public final class Journal {
    private final Object lock = new Object();
    private final StringBuilder trace = new StringBuilder();
    private int count;
    public Object lock() { return lock; }
    public void mark(String label) { count++; trace.append(label).append(':'); }
    public int count() { return count; }
    public String trace() { return trace.toString(); }
    public static String units(String text) {
        if (text == null) return "null";
        StringBuilder result = new StringBuilder();
        for (int i = 0; i < text.length(); i++) result.append((int) text.charAt(i)).append(',');
        return result.toString();
    }
}
