package parity.enummath;

public class MatrixProcessor<T extends Number> {
    private final Class<T> type;

    public MatrixProcessor(Class<T> type) {
        this.type = type;
    }

    public T process(Operation op, T[][] matrix) {
        double result = op.apply(matrix);
        // A simple cast back for Double type only for testing purposes
        if (type == Double.class) {
            return (T) Double.valueOf(result);
        }
        return null;
    }
}
