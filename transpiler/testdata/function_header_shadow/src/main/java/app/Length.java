package app;
import java.util.function.Function;
public final class Length implements Function<String,Integer> {
    private final int bias;
    private int calls;
    private int total;
    public Length(int bias) { this.bias = bias; }
    public Integer apply(String value) {
        calls++;
        total += value.length();
        return bias + value.length() + calls;
    }
    public int calls() { return calls; }
    public int total() { return total; }
    public String bodyMarker() {
        Function<String,Integer> marker = new Function<>();
        return marker.label();
    }
    public static final class Function<T,R> {
        public String label() { return "body"; }
    }
}
