package builderprobe;

// Stateful seeded editing; completed strings never contain isolated surrogates.
public class BuilderWorkload {
    private static long fold(String value, long digest) {
        char[] rendered = value.toCharArray();
        for (char unit : rendered) digest = digest * 31 + unit;
        return digest * 31 + rendered.length;
    }

    public static long run(int seed, int mode, int batches, int steps) {
        String[] words = mode == 0
            ? new String[]{"alpha", "beta", "gamma", "delta"}
            : new String[]{"naïve", "雪山", "😀orbit", "🧪lab"};
        char[] units = mode == 0
            ? new char[]{'[', 'a', 'b', ']'}
            : new char[]{'[', (char)0xd83d, (char)0xde00, '雪', ']'};
        int state = seed;
        long digest = 17;
        for (int batch = 0; batch < batches; batch++) {
            StringBuilder text = new StringBuilder(mode == 0 ? "start" : "A😀B");
            for (int step = 0; step < steps; step++) {
                state = state * 1664525 + 1013904223;
                int choice = (state >>> 16) & 7;
                String word = words[(state >>> 24) & 3];
                switch (choice) {
                    case 0: text.append(word); break;
                    case 1: text.append(units); break;
                    case 2: text.insert(0, word); break;
                    case 3: text.insert(text.length(), units); break;
                    case 4: text.reverse(); text.append('|'); break;
                    case 5: text.append(step); text.append(':'); break;
                    case 6:
                        text.append('!');
                        text.deleteCharAt(text.length() - 1);
                        text.insert(0, '#');
                        break;
                    default:
                        if (mode == 0) text.append('x');
                        else text.append((char)0xd83d).append((char)0xde00);
                        break;
                }
                if (mode == 2 || (step & 15) == 15) {
                    String snapshot = text.toString();
                    digest = fold(snapshot, digest);
                    digest = digest * 31 + snapshot.length();
                }
                digest = digest * 31 + text.charAt((state >>> 8) % text.length());
            }
            text.reverse();
            String finalText = text.toString();
            digest = fold(finalText, digest);
            digest = digest * 31 + finalText.length();
            for (int unit = 0; unit < text.length(); unit++) {
                digest = digest * 31 + text.charAt(unit);
            }
        }
        return digest;
    }

    public static void main(String[] args) {
        int[] seeds = {1, 17, 9031, -104729};
        for (int seed : seeds) {
            for (int mode = 0; mode < 3; mode++) {
                System.out.println(seed + ":" + mode + ":" + run(seed, mode, 16, 128));
            }
        }
    }
}
