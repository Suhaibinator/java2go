package parity.enummath;

public class EnumMathApplication {
    public static void main(String[] args) {
        MatrixProcessor<Double> processor = new MatrixProcessor<>(Double.class);
        Double[][] data = {
            {1.0, 2.0, 3.0},
            {4.0, 5.0, 6.0},
            {7.0, 8.0, 9.0}
        };

        Double trace = processor.process(Operation.TRACE, data);
        Double sum = processor.process(Operation.SUM, data);

        System.out.println("Trace: " + trace);
        System.out.println("Sum: " + sum);
        System.out.println("Operation TRACE index: " + Operation.TRACE.getIndex());
    }
}
