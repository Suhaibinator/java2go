package probe.parse;
import probe.state.Journal;
import static java.lang.Long.parseLong;
public final class LongFlow {
    public static long run(int seed) {
        String[] inputs = {"9223372036854775807", "-9223372036854775808", "+0", "-0",
            "\u0661\u0662", "\uff11\uff12", "\u0967\u0968", "\uff26\uff26", "7fffffffffffffff", "-8000000000000000",
            "9223372036854775808", "-9223372036854775809", "8000000000000000", null, null, null,
            "", "+", "-", " 17", "17 ", "1_7", "\ud800", "\u0000", "12", "12"};
        int[] radix = {10,10,10,10,10,10,10,16,16,16,10,10,16,10,1,37,10,10,10,10,10,10,10,10,1,37};
        long state = seed;
        for (int i = 0; i < inputs.length; i++) {
            try {
                long value = java.lang.Long.parseLong(inputs[i], radix[i]);
                state = state * 31 + value;
                System.out.println("long:" + i + ":ok:" + value + ":" + state);
            } catch (NumberFormatException problem) {
                state = state * 31 + i;
                System.out.println("long:" + i + ":error:" + problem.getClass().getName() + ":" + Journal.units(problem.getMessage()) + ":" + state);
            }
        }
        String dynamic = new String(new char[]{(char) ('0' + seed % 10), (char) ('0' + seed / 10 % 10)});
        long decimal = parseLong(dynamic);
        state += decimal;
        System.out.println("long:default:" + Journal.units(dynamic) + ":" + decimal + ":" + state);
        return state;
    }
}
