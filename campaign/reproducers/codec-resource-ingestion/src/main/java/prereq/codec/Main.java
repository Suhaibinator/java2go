package prereq.codec;

import java.io.InputStream;
import org.apache.commons.codec.Resources;

/** Reads the real locked rule payload through the real dependency implementation. */
public final class Main {
    public static void main(String[] args) throws Exception {
        byte[] bytes;
        try (InputStream stream = Resources.getInputStream("/org/apache/commons/codec/language/dmrules.txt")) {
            bytes = stream.readAllBytes();
        }
        long sum = 0;
        long weighted = 0;
        for (int index = 0; index < bytes.length; index++) {
            int value = bytes[index] & 255;
            sum += value;
            weighted += (index + 1L) * value;
        }
        boolean same = true;
        try (InputStream relative = Resources.class.getResourceAsStream("language/dmrules.txt")) {
            byte[] other = relative.readAllBytes();
            same = bytes.length == other.length;
            for (int index = 0; index < bytes.length && same; index++) {
                same = bytes[index] == other[index];
            }
        }
        System.out.println("seed=" + args[0]);
        System.out.println("bytes=" + bytes.length + ":" + sum + ":" + weighted);
        System.out.println("relative=" + same);
        System.out.println("missing=" + (Resources.class.getResourceAsStream("/codec-prerequisite-missing.txt") == null));
    }
}
