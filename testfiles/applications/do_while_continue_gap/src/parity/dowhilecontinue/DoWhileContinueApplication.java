package parity.dowhilecontinue;

public final class DoWhileContinueApplication {
    private DoWhileContinueApplication() {}

    public static void main(String[] args) {
        int iterations = 0;
        int trace = 0;
        do {
            iterations++;
            try {
                if (iterations == 1) {
                    continue;
                }
            } finally {
                trace = trace * 10 + iterations;
            }
        } while (false);
        System.out.println(iterations + ":" + trace);
    }
}
