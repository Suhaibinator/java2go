package parity.synchronizedcontrol;

public final class SynchronizedLoopControlApplication {
    private static final Object LOCK = new Object();

    private SynchronizedLoopControlApplication() {}

    public static void main(String[] args) {
        int trace = 0;
        for (int index = 0; index < 3; index++) {
            synchronized (LOCK) {
                trace = trace * 10 + index + 1;
                try {
                    trace = trace * 10 + 7;
                } finally {
                    trace = trace * 10 + 8;
                }
                if (index < 2) {
                    continue;
                }
            }
            trace = trace * 10 + 9;
        }
        System.out.println(trace);
    }
}
