package app;
public class StringHolder<T extends java.util.List<String>> {
    public T value;
    public StringHolder(T value) { this.value = value; }
    public int size() { return value.size(); }
    public static class String {
        public static java.lang.String label() { return "body-string"; }
    }
}
