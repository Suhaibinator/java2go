package parity.numerical.kernel;

import parity.numerical.model.DenseMatrix;

public class BlockedMatrixKernel {
    public static DenseMatrix multiply(DenseMatrix left, DenseMatrix right, int blockSize) {
        int size = left.size();
        DenseMatrix result = new DenseMatrix(size);
        double scale = 1.0 / (double) size;

        for (int rowBlock = 0; rowBlock < size; rowBlock += blockSize) {
            int rowLimit = Math.min(rowBlock + blockSize, size);
            for (int depthBlock = 0; depthBlock < size; depthBlock += blockSize) {
                int depthLimit = Math.min(depthBlock + blockSize, size);
                for (int columnBlock = 0; columnBlock < size; columnBlock += blockSize) {
                    int columnLimit = Math.min(columnBlock + blockSize, size);
                    for (int row = rowBlock; row < rowLimit; row++) {
                        for (int depth = depthBlock; depth < depthLimit; depth++) {
                            double leftValue = left.get(row, depth);
                            for (int column = columnBlock; column < columnLimit; column++) {
                                double product = leftValue * right.get(depth, column);
                                result.add(row, column, product * scale);
                            }
                        }
                    }
                }
            }
        }
        return result;
    }

    public static DenseMatrix cascade(DenseMatrix initial, DenseMatrix right, int rounds, int blockSize) {
        DenseMatrix current = initial;
        for (int round = 0; round < rounds; round++) {
            current = multiply(current, right, blockSize);
            double diagonalBias = (double) (round + 1) * 0.00001;
            for (int index = 0; index < current.size(); index++) {
                current.add(index, index, diagonalBias);
            }
        }
        return current;
    }
}
