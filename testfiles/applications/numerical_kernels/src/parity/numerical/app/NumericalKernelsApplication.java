package parity.numerical.app;

import parity.numerical.data.DeterministicData;
import parity.numerical.kernel.BlockedMatrixKernel;
import parity.numerical.kernel.FloatingPointKernel;
import parity.numerical.kernel.IterativeStencilKernel;
import parity.numerical.model.DenseMatrix;
import parity.numerical.report.NumericChecksum;

public class NumericalKernelsApplication {
    private static final int MATRIX_SIZE = 224;
    private static final int BLOCK_SIZE = 28;
    private static final int MULTIPLY_ROUNDS = 16;
    private static final int STENCIL_ITERATIONS = 300;
    private static final int VECTOR_LENGTH = 262144;
    private static final int VECTOR_PASSES = 96;

    private static long checksumMatrix(DenseMatrix matrix) {
        return NumericChecksum.matrix(matrix, 1000000000000.0);
    }

    public static void main(String[] args) {
        DenseMatrix left = DeterministicData.matrix(MATRIX_SIZE, 17L);
        DenseMatrix right = DeterministicData.matrix(MATRIX_SIZE, 7919L);
        DenseMatrix multiplied = BlockedMatrixKernel.cascade(
                left, right, MULTIPLY_ROUNDS, BLOCK_SIZE);

        DenseMatrix stencilSeed = DeterministicData.matrix(MATRIX_SIZE, 104729L);
        DenseMatrix evolved = IterativeStencilKernel.evolve(stencilSeed, STENCIL_ITERATIONS);

        double[] vectorSeed = DeterministicData.vector(VECTOR_LENGTH, 65537L);
        double[] vector = FloatingPointKernel.iterate(vectorSeed, VECTOR_PASSES);

        long matrixChecksum = checksumMatrix(multiplied);
        long stencilChecksum = checksumMatrix(evolved);
        long floatingChecksum = NumericChecksum.values(vector, 1000000.0);
        long combinedChecksum = matrixChecksum * 31L
                + stencilChecksum * 17L
                + floatingChecksum * 13L;

        System.out.println("application=numerical_kernels");
        System.out.println("matrix_size=" + MATRIX_SIZE);
        System.out.println("block_size=" + BLOCK_SIZE);
        System.out.println("multiply_rounds=" + MULTIPLY_ROUNDS);
        System.out.println("stencil_iterations=" + STENCIL_ITERATIONS);
        System.out.println("vector_length=" + VECTOR_LENGTH);
        System.out.println("vector_passes=" + VECTOR_PASSES);
        System.out.println("matrix_checksum=" + matrixChecksum);
        System.out.println("stencil_checksum=" + stencilChecksum);
        System.out.println("floating_checksum=" + floatingChecksum);
        System.out.println("combined_checksum=" + combinedChecksum);
    }
}
