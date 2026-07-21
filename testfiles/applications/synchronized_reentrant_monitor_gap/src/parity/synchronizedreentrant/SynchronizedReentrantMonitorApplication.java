package parity.synchronizedreentrant;

public final class SynchronizedReentrantMonitorApplication {
    private static int trace;

    private SynchronizedReentrantMonitorApplication() {}

    private static Object evaluatedLock(Object lock, int marker) {
        trace = trace * 10 + marker;
        return lock;
    }

    public static void main(String[] args) {
        Object lock = new Object();
        trace = 1;

        synchronized (evaluatedLock(lock, 2)) {
            System.out.println("OUTER=" + trace);
            synchronized (evaluatedLock(lock, 3)) {
                trace = trace * 10 + 4;
            }
            trace = trace * 10 + 5;
        }

        trace = trace * 10 + 6;
        System.out.println("RESULT=" + trace);
    }
}
