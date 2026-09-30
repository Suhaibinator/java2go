package parity.arraycovariance;

public class CovariantArrayStoreApplication {
    static String trace;

    static void record(String token) {
        trace = trace + token;
    }

    static class Base {
        int value;

        Base(int value) {
            record("b");
            this.value = value;
        }
    }

    static class Child extends Base {
        Child(int value) {
            super(value);
            record("c");
        }
    }

    static Base[] select(Base[] values) {
        record("a");
        return values;
    }

    static int index() {
        record("i");
        return 0;
    }

    static Base replacement() {
        record("r");
        return new Base(9);
    }

    static int recursiveSum(Base[] values, int index) {
        record("s");
        return index == values.length
                ? 0
                : values[index].value + recursiveSum(values, index + 1);
    }

    public static void main(String[] args) {
        trace = "";
        Base[] view = new Child[] {new Child(4), new Child(5)};
        int before = recursiveSum(view, 0);

        try {
            select(view)[index()] = replacement();
            record("x");
        } catch (ArrayStoreException expected) {
            record("e");
        }

        int after = recursiveSum(view, 0);
        System.out.println("TRACE=" + trace);
        System.out.println("SUMS=" + before + ":" + after);
        System.out.println("VALUES=" + view[0].value + ":" + view[1].value);
    }
}
