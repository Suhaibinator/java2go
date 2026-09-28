package p;
public final class Outer {
    public static final class Inner {
        public <T> T choose(T value) { return value; }
        public <T> T choose(T value, int ignored) { return value; }
    }
    public static String suffix() { return q.Bridge.suffix(); }
}
