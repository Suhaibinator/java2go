import java.util.Arrays;
import java.util.stream.Collectors;

// The same implementation is compiled by JDK21 and strictly translated to Go.
// Input preparation belongs outside sustained timing in either harness.
public class JoiningWorkload {
    public static String[] retained = new String[16];

    public static String[] prepare(int seed, int mode, int count, int width) {
        String[] values = new String[count];
        int state = seed;
        for (int item = 0; item < count; item++) {
            if (mode == 2 && (item & 3) == 3) {
                values[item] = null;
                continue;
            }
            char[] chars = new char[width];
            for (int unit = 0; unit < width; unit++) {
                state = state * 1664525 + 1013904223;
                int choice = (state >>> 24) & 3;
                if (mode == 0) chars[unit] = (char)('a' + choice);
                else if (mode == 1) chars[unit] = (char)(0xe000 + choice);
                else if (choice == 0) chars[unit] = (char)0;
                else if (choice == 1) chars[unit] = (char)0xd800;
                else if (choice == 2) chars[unit] = (char)0xdc00;
                else chars[unit] = (char)'Z';
            }
            values[item] = new String(chars);
        }
        return values;
    }

    public static long exercise(String[] values, int arity, int repetitions) {
        long digest = 17;
        String previous = null;
        for (int repeat = 0; repeat < repetitions; repeat++) {
            String result;
            if (arity == 0) result = Arrays.stream(values).collect(Collectors.joining());
            else if (arity == 1) result = Arrays.stream(values).collect(Collectors.joining("|"));
            else result = Arrays.stream(values).collect(Collectors.joining("|", "[", "]"));
            digest = digest * 31 + result.length();
            digest = digest * 31 + result.hashCode();
            digest = digest * 31 + (result == previous ? 1 : 0);
            retained[repeat & 15] = result;
            previous = result;
        }
        for (int slot = 0; slot < retained.length; slot++) {
            String value = retained[slot];
            if (value != null) digest = digest * 31 + value.length() + value.hashCode();
        }
        return digest;
    }

    public static String observe(String[] values, int arity) {
        exercise(values, arity, 16);
        String result = retained[0];
        StringBuilder observation = new StringBuilder();
        observation.append(result.length()).append(':').append(result.hashCode()).append(':');
        for (int unit = 0; unit < result.length(); unit++) observation.append((int)result.charAt(unit)).append(',');
        return observation.toString();
    }

    public static void main(String[] args) {
        int[] seeds = new int[]{17, 41, 97};
        for (int seed : seeds) {
            for (int mode = 0; mode < 3; mode++) {
                for (int arity = 0; arity < 3; arity++) {
                    for (int count : new int[]{0, 1, 8, 128}) {
                        String[] values = prepare(seed, mode, count, 16);
                        System.out.println(seed + ":" + mode + ":" + arity + ":" + count + ":" + exercise(values, arity, 16));
                    }
                    System.out.println(seed + ":" + mode + ":" + arity + ":units:" + observe(prepare(seed, mode, 3, 8), arity));
                    System.out.println(seed + ":" + mode + ":" + arity + ":long:" + observe(prepare(seed, mode, 8, 1024), arity));
                }
            }
        }
    }
}
