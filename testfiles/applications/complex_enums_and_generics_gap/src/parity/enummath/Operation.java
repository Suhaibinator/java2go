package parity.enummath;

public enum Operation {
    TRACE(1) {
        @Override
        public <T extends Number> double apply(T[][] matrix) {
            double sum = 0;
            for (int i = 0; i < matrix.length; i++) {
                sum += matrix[i][i].doubleValue();
            }
            return sum;
        }
    },
    SUM(2) {
        @Override
        public <T extends Number> double apply(T[][] matrix) {
            double sum = 0;
            for (int i = 0; i < matrix.length; i++) {
                for (int j = 0; j < matrix[i].length; j++) {
                    sum += matrix[i][j].doubleValue();
                }
            }
            return sum;
        }
    };

    private final int index;

    Operation(int index) {
        this.index = index;
    }

    public int getIndex() {
        return index;
    }

    public abstract <T extends Number> double apply(T[][] matrix);
}
