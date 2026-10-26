public class StatementFormalControlFlow {
    static int trace;

    static int step(/* first formal */ int digit /* after first */, /* second formal */ int scale /* after second */) {
        trace = trace * scale + digit;
        return digit;
    }

    public static String run() {
        trace = 0;
        int remaining = 2;
        int total = 0;
        int visits = 0;
        while (/* condition */ remaining > 0) /* between condition and body */ {
            total += step(remaining, 10);
            remaining--;
            visits++;
        }
        do /* before body */ {
            total += step(3, 10);
            visits++;
        } /* between body and while */ while (/* final condition */ false);
        int[] values = new int[]{4, 5};
        for (/* before type */ int /* before name */ value /* before colon */ : /* before iterable */ values) /* before body */ {
            total += step(value, 10);
            visits++;
        }
        return "Ω😀|trace:" + trace + "|state:" + remaining + ":" + total + ":" + visits
            + "|iterable:" + values[0] + ":" + values[1];
    }
}
