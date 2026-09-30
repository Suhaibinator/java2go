interface IntOp {
    int apply(int a, int b);
}

interface IntPred {
    boolean test(int value);
}

public class Lambdas {
    static int reduce(int[] xs, int seed, IntOp op) {
        int acc = seed;
        for (int x : xs) {
            acc = op.apply(acc, x);
        }
        return acc;
    }

    static int countMatching(int[] xs, IntPred pred) {
        int count = 0;
        for (int x : xs) {
            if (pred.test(x)) {
                count++;
            }
        }
        return count;
    }

    public static void main(String[] args) {
        int[] xs = new int[] { 1, 2, 3, 4, 5 };

        IntOp add = (a, b) -> a + b;
        IntOp mul = (a, b) -> a * b;

        System.out.println(reduce(xs, 0, add));
        System.out.println(reduce(xs, 1, mul));

        IntPred even = v -> v % 2 == 0;
        System.out.println(countMatching(xs, even));
    }
}
