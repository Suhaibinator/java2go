package parity.numerical.kernel;

public class FloatingPointKernel {
    public static double[] iterate(double[] seed, int passes) {
        double[] current = seed;
        for (int pass = 0; pass < passes; pass++) {
            double[] next = new double[current.length];
            double drift = (double) ((pass % 13) - 6) * 0.00000001;
            for (int i = 0; i < current.length; i++) {
                int previous = i == 0 ? current.length - 1 : i - 1;
                int following = i + 1 == current.length ? 0 : i + 1;
                double value = current[i];
                double coupled = current[previous] * 0.03125 + current[following] * 0.015625;
                double squared = value * value;
                double curved = value - value * squared * 0.00002;
                next[i] = curved * 0.953125 + coupled + drift;
            }
            current = next;
        }
        return current;
    }
}
