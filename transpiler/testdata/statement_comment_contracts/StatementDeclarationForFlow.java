public class StatementDeclarationForFlow {
    static int trace;

    static int mark(int digit) {
        trace = trace * 10 + digit;
        return digit;
    }

    public static String run() {
        trace = 0;
        int first /* first declaration name */ = mark(1), second /* second declaration name */ = first + mark(2);
        int total = 0;
        int rounds = 0;
        int last = 0;
        for (int i /* first for-local */ = mark(3), j /* later local sees i */ = i + mark(4);
             i /* condition left */ < /* condition right */ 5;
             i /* spelling -- is trivia */ ++) {
            total /* compound target */ += /* RHS observes i and j */ i * 10 + j;
            last = mark(i);
            rounds++;
        }
        var inferred /* initializer determines the Java type */ = mark(5);
        int uninitialized /* no initializer exists */;
        uninitialized = /* assignment RHS */ inferred + second;
        return "Ω😀|trace:" + trace + "|locals:" + first + ":" + second + ":" + inferred
            + "|loop:" + rounds + ":" + last + ":" + total + "|assigned:" + uninitialized;
    }
}
