public class Nested {
    static class Counter {
        private int count;

        void inc() {
            count++;
        }

        int get() {
            return count;
        }
    }

    class Inner {
        String tag() {
            return "inner of " + label;
        }
    }

    private String label = "outer";

    Inner makeInner() {
        return new Inner();
    }

    public static void main(String[] args) {
        Counter c = new Counter();
        c.inc();
        c.inc();
        c.inc();
        System.out.println(c.get());

        Nested outer = new Nested();
        Nested.Inner inner = outer.makeInner();
        System.out.println(inner.tag());
    }
}
