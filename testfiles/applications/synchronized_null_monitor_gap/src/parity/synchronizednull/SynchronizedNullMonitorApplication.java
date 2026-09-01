package parity.synchronizednull;

public final class SynchronizedNullMonitorApplication {
    private static int trace;

    private SynchronizedNullMonitorApplication() {}

    private static Object evaluatedNullLock() {
        trace = trace * 10 + 1;
        return null;
    }

    public static void main(String[] args) {
        try {
            synchronized (evaluatedNullLock()) {
                trace = trace * 10 + 2;
            }
            trace = trace * 10 + 3;
        } catch (NullPointerException expected) {
            trace = trace * 10 + 4;
        }

        trace = trace * 10 + 5;
        System.out.println("TRACE=" + trace);
    }
}
