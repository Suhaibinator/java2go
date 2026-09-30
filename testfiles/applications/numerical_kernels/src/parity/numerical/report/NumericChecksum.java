package parity.numerical.report;

import parity.numerical.model.DenseMatrix;

public class NumericChecksum {
    public static long matrix(DenseMatrix matrix, double scale) {
        return values(matrix.values(), scale);
    }

    public static long values(double[] values, double scale) {
        long checksum = 1469598103934665603L;
        for (int i = 0; i < values.length; i++) {
            long quantized = (long) (values[i] * scale);
            checksum = checksum * 1000003L + quantized + (long) (i + 1) * 97L;
        }
        return checksum;
    }
}
