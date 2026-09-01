package parity.sideeffects.core;

public final class EvaluationOrderSuite {
    private EvaluationOrderSuite() {
    }

    public static String shortCircuit() {
        TraceLog first = new TraceLog();
        int firstScore = 0;
        if (first.decision("A", false) && first.decision("X", true)) {
            firstScore = 1;
        }

        TraceLog second = new TraceLog();
        int secondScore = 0;
        if (second.decision("B", true) || second.decision("X", false)) {
            secondScore = 10;
        }

        TraceLog third = new TraceLog();
        int thirdScore = 0;
        if (third.decision("C", true) && third.decision("D", true)) {
            thirdScore = 20;
        }

        return firstScore + ":" + first.value()
                + "|" + secondScore + ":" + second.value()
                + "|" + thirdScore + ":" + third.value();
    }

    public static String ternary() {
        TraceLog first = new TraceLog();
        int firstValue = first.decision("T", true)
                ? first.number("L", 4)
                : first.number("R", 9);

        TraceLog second = new TraceLog();
        int secondValue = second.decision("F", false)
                ? second.number("L", 4)
                : second.number("R", 9);

        TraceLog nested = new TraceLog();
        int nestedValue = nested.decision("X", false)
                ? nested.number("N", 1)
                : (nested.decision("Y", true)
                        ? nested.number("Z", 7)
                        : nested.number("N", 2));

        return firstValue + ":" + first.value()
                + "|" + secondValue + ":" + second.value()
                + "|" + nestedValue + ":" + nested.value();
    }

    public static String ordinaryInvocation() {
        TraceLog trace = new TraceLog();
        int result = select(trace, false).combine(
                argument(trace, "a", 4),
                argument(trace, "b", 7));
        return result + ":" + trace.value();
    }

    public static String nullInvocation() {
        TraceLog trace = new TraceLog();
        try {
            select(trace, true).combine(
                    argument(trace, "a", 4),
                    argument(trace, "b", 7));
        } catch (NullPointerException expected) {
            trace.add("c");
        }
        return trace.value();
    }

    @SuppressWarnings("static-access")
    public static String staticInvocationThroughExpression() {
        TraceLog trace = new TraceLog();
        int result = select(trace, true).staticCombine(
                trace,
                argument(trace, "a", 4),
                argument(trace, "b", 7));
        return result + ":" + trace.value();
    }

    public static String compoundAssignment() {
        int value = 1;
        int assigned = (value += (value = 5));
        return assigned + ":" + value;
    }

    public static String finallyAfterReturnExpression() {
        TraceLog trace = new TraceLog();
        int result = returnWithFinally(trace);
        return result + ":" + trace.value();
    }

    public static String recursiveArguments() {
        TraceLog trace = new TraceLog();
        int result = recurse(trace, 2, 1);
        return result + ":" + trace.value();
    }

    private static InvocationTarget select(TraceLog trace, boolean returnNull) {
        trace.add("r");
        if (returnNull) {
            return null;
        }
        return new InvocationTarget(trace);
    }

    private static int argument(TraceLog trace, String token, int value) {
        trace.add(token);
        return value;
    }

    private static int returnWithFinally(TraceLog trace) {
        try {
            trace.add("t");
            return 7;
        } finally {
            trace.add("f");
        }
    }

    private static int recurse(TraceLog trace, int depth, int value) {
        trace.add("e" + depth);
        if (depth == 0) {
            return value;
        }
        int result = recurse(trace, depth - 1, argument(trace, "a", value + 1));
        trace.add("x" + depth);
        return result;
    }
}
