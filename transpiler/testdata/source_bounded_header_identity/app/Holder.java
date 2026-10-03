package app;
import origin.Limit;
public class Holder<T extends Limit> {
    public T value;
    public Holder(T value) { this.value = value; }
    public int read() { return value.read(); }
    public static class Limit {
        public static java.lang.String label() { return "body-limit"; }
    }
}
