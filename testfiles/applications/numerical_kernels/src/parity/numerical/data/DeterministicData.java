package parity.numerical.data;

import parity.numerical.model.DenseMatrix;

public class DeterministicData {
    private static long advance(long state) {
        return (state * 48271L + 17L) % 2147483647L;
    }

    public static DenseMatrix matrix(int size, long seed) {
        DenseMatrix matrix = new DenseMatrix(size);
        long state = seed;
        for (int row = 0; row < size; row++) {
            for (int column = 0; column < size; column++) {
                state = advance(state);
                int bucket = (int) (state % 2001L) - 1000;
                double value = (double) bucket / 997.0
                        + (double) row * 0.0005
                        - (double) column * 0.0003;
                matrix.set(row, column, value);
            }
        }
        return matrix;
    }

    public static double[] vector(int length, long seed) {
        double[] values = new double[length];
        long state = seed;
        for (int i = 0; i < length; i++) {
            state = advance(state);
            int bucket = (int) (state % 4001L) - 2000;
            values[i] = (double) bucket / 1999.0 + (double) (i % 17) * 0.00001;
        }
        return values;
    }
}
