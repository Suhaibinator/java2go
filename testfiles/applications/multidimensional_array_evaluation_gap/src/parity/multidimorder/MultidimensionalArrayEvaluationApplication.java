package parity.multidimorder;

public final class MultidimensionalArrayEvaluationApplication {
    private static int trace;

    private MultidimensionalArrayEvaluationApplication() {
    }

    private static int dimension(int marker, int value, int depth) {
        if (depth == 0) {
            trace = trace * 10 + marker;
            return value;
        }
        return dimension(marker, value, depth - 1);
    }

    public static void main(String[] args) {
        try {
            int[][][] ignored = new int[
                    dimension(1, 2, 2)
            ][
                    dimension(2, -1, 1)
            ][
                    dimension(3, 4, 3)
            ];
            trace = trace * 100 + ignored.length + 40;
        } catch (NegativeArraySizeException expected) {
            trace = trace * 10 + 8;
        }

        System.out.println(trace);
    }
}
