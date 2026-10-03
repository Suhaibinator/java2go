package builderprobe;

// Supplemental oracle, independent of the unchanged stateful workload.
public class ArgumentOracle {
    static int trace = 0;
    static StringBuilder receiver(StringBuilder text) { trace = trace * 10 + 1; return text; }
    static int offset(int at) { trace = trace * 10 + 2; return at; }
    static String value(String value) { trace = trace * 10 + 3; return value; }
    public static void main(String[] args) {
        String missing = null;
        StringBuilder text = new StringBuilder("A😀B");
        text.append(missing).insert(0, missing).append("");
        text.append((String)null).insert(text.length(), (String)null);
        text.append("雪").insert(0, "🧪");
        System.out.println(text.toString());
        trace = 0;
        receiver(text).insert(offset(1), value("Q"));
        // Remove the insertion before a String snapshot: offset 1 splits 🧪.
        text.deleteCharAt(1);
        System.out.println(trace + ":" + text.toString());
        trace = 0;
        try { receiver(text).insert(offset(-1), value(missing)); }
        catch (StringIndexOutOfBoundsException e) { System.out.println(trace + ":" + e.getClass().getSimpleName()); }
        StringBuilder other = new StringBuilder();
        Object erased = "erased";
        Object erasedNull = null;
        other.append(erased).append(erasedNull).append(123).append('!');
        java.lang.String qualified = "qualified";
        other.append(qualified).insert(0, qualified);
        System.out.println(other.toString());
        StringBuilder snapshot = new StringBuilder("x😀");
        String before = snapshot.toString();
        snapshot.append(value("suffix")).insert(0, value("prefix"));
        System.out.println(before + ":" + snapshot.toString());
    }
}
