interface Transformer {
    int transform(int value);
}

abstract class Describable {
    abstract String describe();

    String shout() {
        return describe() + "!";
    }
}

public class AnonLocal {
    static int applyTwice(Transformer t, int x) {
        return t.transform(t.transform(x));
    }

    public static void main(String[] args) {
        // anonymous class implementing a SAM interface, capturing a local
        int bump = 3;
        Transformer addBump = new Transformer() {
            public int transform(int value) {
                return value + bump;
            }
        };
        System.out.println(applyTwice(addBump, 10));

        // anonymous class extending an abstract class with multiple methods
        Describable d = new Describable() {
            String describe() {
                return "anon";
            }
        };
        System.out.println(d.describe());
        System.out.println(d.shout());

        // local class declared inside a method
        class Counter {
            private int n;

            void inc() {
                n++;
            }

            int value() {
                return n;
            }
        }
        Counter c = new Counter();
        c.inc();
        c.inc();
        System.out.println(c.value());
    }
}
