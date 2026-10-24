package bridge;
import model.Foo;

@SuppressWarnings("unchecked")
public class Box<T> {
    public int calls;
    public String trace = "";
    public final int marker;
    public static int staticCalls;
    public static String staticTrace = "";
    public Box(int marker) { this.marker = marker; }

    public <R> R identity(R value) { calls++; trace += "I"; return value; }
    private <R> R privateIdentity(R value) { calls++; trace += "p"; return value; }
    private static <R> R staticIdentity(R value) { staticCalls++; staticTrace += "s"; return value; }
    public Foo throughPrivate(Foo value) { return privateIdentity(value); }
    public static Foo throughStatic(Foo value) { return staticIdentity(value); }
    public static <X> X relay(Box<X> box, X value) { return box.identity(value); }

    public <R> R polluted(R value) { calls++; trace += "P"; return (R)(Object)new foreign.Foo(marker + 100); }
    public <R> R wrapperPolluted(R value) { calls++; trace += "W"; return (R)(Object)new foreign.Foo(marker + 200); }
    public <R extends Foo> R declared(Object value) { calls++; trace += "D"; R result = (R)value; trace += "d"; return result; }
    public <R> R throwing(R value) { calls++; trace += "E"; throw new IllegalStateException("body"); }
}
