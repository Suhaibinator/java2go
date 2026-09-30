package parity.dispatch.bounded;

public class BoundedDispatchApp {
    interface Measure { <T extends Number> T measure(T value); }
    static abstract class Base implements Measure {
        public abstract <T extends Number> T measure(T value);
    }
    static class Child extends Base {
        public <T extends Number> T measure(T value) { total += value.intValue(); return value; }
    }
    static int total;
    public static String run() {
        Base base = new Child();
        Measure service = base;
        Integer first = base.measure(Integer.valueOf(7));
        Long second = service.measure(Long.valueOf(9));
        return first.intValue() + ":" + second.longValue() + ":" + total;
    }
    public static void main(String[] args) { System.out.println(run()); }
}
