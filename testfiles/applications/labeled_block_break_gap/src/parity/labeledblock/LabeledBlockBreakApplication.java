package parity.labeledblock;

public final class LabeledBlockBreakApplication {
    private LabeledBlockBreakApplication() {}

    public static void main(String[] args) {
        int trace = 0;
        outer: {
            try {
                trace = 1;
                break outer;
            } finally {
                trace = trace * 10 + 2;
            }
        }
        System.out.println(trace);
    }
}
