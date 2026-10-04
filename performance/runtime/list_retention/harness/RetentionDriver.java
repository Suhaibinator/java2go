public final class RetentionDriver {
    private static long usedHeap() {
        System.gc();
        System.gc();
        Runtime runtime = Runtime.getRuntime();
        return runtime.totalMemory() - runtime.freeMemory();
    }
    private static void metric(String phase, long baseline) {
        long used = usedHeap();
        System.err.println("{\"runtime\":\"java\",\"phase\":\"" + phase
                + "\",\"used_heap\":" + used + ",\"delta\":" + (used-baseline) + "}");
    }
    public static void main(String[] args) {
        int count = Integer.parseInt(args[0]);
        int bytes = Integer.parseInt(args[1]);
        int rounds = Integer.parseInt(args[2]);
        for (int i=0; i<4; i++) { ListRetention.cycle(i, count, 1024); }
        ListRetention.reset();
        // Warm output formatting and metric machinery before baseline.
        usedHeap();
        String.format("round=%d checksum=%d size=%d", 0, 0L, 0);
        long baseline = usedHeap();
        metric("baseline", baseline);
        for (int round=0; round<rounds; round++) {
            long checksum = ListRetention.cycle(round, count, bytes);
            System.out.printf("round=%d checksum=%d size=%d%n", round, checksum, ListRetention.size());
            metric("removed-"+round, baseline);
        }
        ListRetention.clearList();
        System.out.println("cleared size=" + ListRetention.size());
        metric("cleared", baseline);
    }
}
