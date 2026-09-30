package parity.finallyloop;

public final class FinallyLoopApplication {
    private static String trace = "";

    private FinallyLoopApplication() {
    }

    public static void main(String[] args) {
        for (int index = 0; index < 3; index++) {
            try {
                trace = trace + "t" + index;
                if (index == 0) {
                    continue;
                }
                if (index == 1) {
                    break;
                }
            } finally {
                trace = trace + "f" + index;
            }
        }
        trace = trace + "e";
        System.out.println(trace);
    }
}
