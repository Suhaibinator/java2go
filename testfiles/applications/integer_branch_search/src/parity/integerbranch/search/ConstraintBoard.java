package parity.integerbranch.search;

public class ConstraintBoard {
    public static long[] allowedRows(int size, int variant) {
        long fullMask = (1L << size) - 1L;
        long[] allowed = new long[size];

        for (int row = 0; row < size; row++) {
            long rowMask = fullMask;
            if (variant > 0) {
                int firstColumn = (row * (variant * 2 + 3) + variant * 5 + 1) % size;
                rowMask = rowMask & ~(1L << firstColumn);
            }
            if (variant == 2 && row % 3 == 0) {
                int secondColumn = (row * 7 + 11) % size;
                rowMask = rowMask & ~(1L << secondColumn);
            }
            if (variant == 3 && row % 2 == 1) {
                int secondColumn = (row * 9 + 4) % size;
                rowMask = rowMask & ~(1L << secondColumn);
            }
            allowed[row] = rowMask;
        }
        return allowed;
    }
}
