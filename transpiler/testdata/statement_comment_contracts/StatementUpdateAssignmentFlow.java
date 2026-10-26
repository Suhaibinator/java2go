public class StatementUpdateAssignmentFlow {
    static int trace;
    static int[] shared;

    static int index(int digit) {
        trace = trace * 10 + digit;
        return 0;
    }

    static int mutate(int digit) {
        trace = trace * 10 + digit;
        shared[0] = 100;
        return digit;
    }

    public static String run() {
        trace = 0;
        shared = new int[]{8, 12, 16};
        int cursor = 1;
        int previous = cursor /* spelling ++ is trivia */ --;
        ++ /* spelling -- is trivia */ cursor;
        cursor /* spelling ++ is trivia */ --;
        int next = ++ /* prefix operand */ cursor;

        shared[index(1)] /* save the old component first */ += /* then RHS */ mutate(2);
        int compound = shared[0];
        shared[index(3)] = /* store after the RHS mutation */ mutate(4);
        int simple = shared[0];
        int selected = shared[index(5)] /* binary left */ + /* binary right */ mutate(6);

        return "Ω😀|trace:" + trace + "|updates:" + previous + ":" + cursor + ":" + next
            + "|saved:" + compound + ":" + simple + ":" + selected
            + "|array:" + shared[0] + ":" + shared[1] + ":" + shared[2];
    }
}
