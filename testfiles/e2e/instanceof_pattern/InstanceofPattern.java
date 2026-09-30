public class InstanceofPattern {
    static String describe(Object o) {
        if (o instanceof String s) {
            return "string of length " + s.length();
        } else if (o instanceof Integer i) {
            return "int doubled " + (i * 2);
        } else {
            return "unknown";
        }
    }

    public static void main(String[] args) {
        Object[] items = new Object[] { "hello", 21, 3.5 };
        for (Object o : items) {
            System.out.println(describe(o));
        }
    }
}
