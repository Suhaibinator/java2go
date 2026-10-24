import builderprobe.BuilderWorkload;
import java.lang.management.ManagementFactory;

// Deliberately not transpiled: JVM-specific allocation instrumentation.
public class AllocationDriver {
    public static void main(String[] args) {
        com.sun.management.ThreadMXBean meter =
            (com.sun.management.ThreadMXBean)ManagementFactory.getThreadMXBean();
        meter.setThreadAllocatedMemoryEnabled(true);
        long thread = Thread.currentThread().threadId();
        int[] seeds = {1, 17, 9031, -104729};
        for (int seed : seeds) for (int mode = 0; mode < 3; mode++) {
            long expected = BuilderWorkload.run(seed, mode, 16, 128);
            for (int i = 0; i < 12; i++)
                if (BuilderWorkload.run(seed, mode, 16, 128) != expected) throw new AssertionError("warmup parity");
            for (int trial = 0; trial < 4; trial++) {
                long before = meter.getThreadAllocatedBytes(thread);
                long actual = BuilderWorkload.run(seed, mode, 16, 128);
                long allocated = meter.getThreadAllocatedBytes(thread) - before;
                if (actual != expected) throw new AssertionError("measurement parity");
                System.out.println(seed + "," + mode + "," + trial + "," + actual + "," + allocated);
            }
        }
    }
}
