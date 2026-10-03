import java.io.Writer;

class ObservationWriter extends Writer {
    final byte[] payload;
    int written;
    ObservationWriter(int bytes) { payload = new byte[bytes]; }
    public void write(char[] chars, int offset, int length) {
        if (!Thread.holdsLock(this)) { throw new AssertionError("missing writer ownership"); }
        written += chars[offset];
    }
    public void flush() {}
    public void close() {}
}

public final class MonitorObservation {
    public static long cycle(int round, int count, int bytes, int mode) throws Exception {
        long checksum = 0;
        for (int i=0; i<count; i++) {
            if (mode == 0) {
                byte[] value = new byte[bytes];
                value[0] = (byte)((round + i) & 127);
                if (Thread.holdsLock(value)) { throw new AssertionError("unheld array owned"); }
                checksum += value[0];
            } else {
                ObservationWriter value = new ObservationWriter(bytes);
                Writer writer = value;
                writer.write(65 + (i & 31));
                checksum += value.written;
                writer.close();
            }
        }
        return checksum;
    }
    public static String ownership() throws Exception {
        Object lock = new Object();
        boolean outside = Thread.holdsLock(lock);
        boolean own;
        boolean nested;
        boolean[] other = new boolean[1];
        synchronized(lock) {
            own = Thread.holdsLock(lock);
            synchronized(lock) { nested = Thread.holdsLock(lock); }
            Thread thread = new Thread(() -> { other[0] = Thread.holdsLock(lock); });
            thread.start();
            thread.join();
        }
        boolean after = Thread.holdsLock(lock);
        String failure = "missing";
        try { Thread.holdsLock(null); } catch (NullPointerException e) { failure="NPE"; }
        return outside+":"+own+":"+nested+":"+other[0]+":"+after+":"+failure;
    }
}
