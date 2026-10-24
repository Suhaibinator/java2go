import java.util.ArrayList;
import java.util.List;

// The same application logic is compiled by javac and transpiled to Go.
public final class ListRetention {
    private static List<byte[]> values = new ArrayList<>();

    public static void reset() { values = new ArrayList<>(); }
    public static int size() { return values.size(); }
    public static void clearList() { values.clear(); }

    public static long cycle(int round, int count, int bytes) {
        for (int i = 0; i < count; i++) {
            byte[] value = new byte[bytes];
            value[0] = (byte) ((round * 31 + i) & 127);
            value[bytes - 1] = (byte) ((round * 17 + i * 3) & 127);
            values.add(value);
        }
        long checksum = 0;
        while (!values.isEmpty()) {
            byte[] removed = values.remove(values.size() - 1);
            checksum += removed[0] + removed[removed.length - 1];
        }
        return checksum;
    }
}
