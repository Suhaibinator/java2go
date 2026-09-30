package parity.synchronizedarray;

public final class SynchronizedArrayMonitorApplication {
    private static int trace;

    private SynchronizedArrayMonitorApplication() {}

    private static int[] evaluatedLock(int[] lock, int marker) {
        trace = trace * 10 + marker;
        return lock;
    }

    public static void main(String[] args) {
        int[] lock = new int[] {7, 11, 13};
        int[] alias = lock;
        trace = 1;

        synchronized (evaluatedLock(alias, 2)) {
            trace = trace * 10 + 3;
            alias[1] += trace;
        }

        trace = trace * 10 + 4;
        System.out.println("TRACE=" + trace);
        System.out.println("VALUES=" + lock[0] + ":" + lock[1] + ":" + lock[2]);
    }
}
