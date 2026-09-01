package parity.numerical.model;

public final class DenseMatrix {
    private final int size;
    private final double[] values;

    public DenseMatrix(int size) {
        this.size = size;
        this.values = new double[size * size];
    }

    public int size() {
        return this.size;
    }

    public double get(int row, int column) {
        return this.values[row * this.size + column];
    }

    public void set(int row, int column, double value) {
        this.values[row * this.size + column] = value;
    }

    public void add(int row, int column, double value) {
        int index = row * this.size + column;
        this.values[index] = this.values[index] + value;
    }

    public double[] values() {
        return this.values;
    }
}
