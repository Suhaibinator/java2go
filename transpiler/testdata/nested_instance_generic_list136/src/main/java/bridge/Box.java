package bridge;
import java.util.List;
import model.Foo;

@SuppressWarnings("unchecked")
public class Box<T> {
    public int calls;
    public String trace = "";
    public final int marker;
    public Box(int marker) { this.marker = marker; }

    public <R> R identity(R value) {
        calls++;
        trace += "I";
        return value;
    }

    public static List<Foo> call(Box<List<Foo>> box, List<Foo> value) {
        return box.identity(value);
    }

    public <R> R polluted(R value) {
        calls++;
        trace += "P";
        return (R)(Object)Integer.valueOf(marker);
    }

    public <R extends Number> R numeric(Object value) {
        calls++;
        trace += "N";
        R result = (R)value;
        trace += "n";
        return result;
    }

    public <R extends Foo> R declared(Object value) {
        calls++;
        trace += "D";
        R result = (R)value;
        trace += "d";
        return result;
    }
}
