package parity.dispatch.shadowing;

public class ShadowedBoundsApp {
    interface ClassBound {}
    interface MethodBound { int value(); }
    static class ClassValue implements ClassBound {}
    static class MethodValue implements MethodBound {
        public int value() { return 7; }
    }
    static class MethodLeaf extends MethodValue {
        public int value() { return 9; }
    }
    static class Holder<T extends ClassBound> {
        <T extends MethodBound> T echo(T value) { return value; }
    }

    public static String run() {
        Holder<ClassValue> holder = new Holder<ClassValue>();
        MethodLeaf value = new MethodLeaf();
        MethodBound result = holder.echo(value);
        MethodBound absent = holder.echo((MethodLeaf) null);
        return result.value() + ":" + (result == value) + ":" + (absent == null);
    }

    public static void main(String[] args) { System.out.println(run()); }
}
