package parity.numerical.kernel;

import parity.numerical.model.DenseMatrix;

public class IterativeStencilKernel {
    public static DenseMatrix evolve(DenseMatrix seed, int iterations) {
        DenseMatrix current = seed;
        int size = seed.size();

        for (int iteration = 0; iteration < iterations; iteration++) {
            DenseMatrix next = new DenseMatrix(size);
            double forcing = (double) ((iteration % 11) - 5) * 0.0000001;

            for (int row = 0; row < size; row++) {
                int north = row == 0 ? size - 1 : row - 1;
                int south = row + 1 == size ? 0 : row + 1;
                for (int column = 0; column < size; column++) {
                    int west = column == 0 ? size - 1 : column - 1;
                    int east = column + 1 == size ? 0 : column + 1;

                    double center = current.get(row, column);
                    double neighbors = current.get(north, column)
                            + current.get(south, column)
                            + current.get(row, west)
                            + current.get(row, east);
                    double mixed = center * 0.52 + neighbors * 0.12;
                    double squared = mixed * mixed;
                    double transformed = mixed - mixed * squared * 0.00003 + forcing;
                    next.set(row, column, transformed);
                }
            }
            current = next;
        }
        return current;
    }
}
