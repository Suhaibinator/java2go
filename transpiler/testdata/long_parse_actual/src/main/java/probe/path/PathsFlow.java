package probe.path;
import probe.state.Journal;
public final class PathsFlow {
    private static int one(String label, String first, String[] more, int state) {
        try {
            java.nio.file.Path result = java.nio.file.Paths.get(first, more);
            state += result.getNameCount();
            System.out.println("path:" + label + ":ok:" + Journal.units(result.toString()) + ":" + result.getNameCount() + ":" + state);
        } catch (RuntimeException problem) {
            state++;
            System.out.println("path:" + label + ":error:" + problem.getClass().getName() + ":" + Journal.units(problem.getMessage()) + ":" + state);
            if (problem instanceof java.nio.file.InvalidPathException) {
                java.nio.file.InvalidPathException invalid = (java.nio.file.InvalidPathException) problem;
                System.out.println("invalid:" + label + ":" + Journal.units(invalid.getInput()) + ":" + Journal.units(invalid.getReason()) + ":" + invalid.getIndex());
            }
        }
        return state;
    }
    private static String evaluated(Journal journal, String label, String text) { journal.mark(label); return text; }
    public static int run(int seed) {
        int state = seed;
        String leaf = "leaf" + seed;
        state = one("array", "root//", new String[]{"", "part/", leaf}, state);
        java.nio.file.Path packed = java.nio.file.Paths.get("root//", "", "part/", leaf);
        System.out.println("path:varargs:" + Journal.units(packed.toString()) + ":" + packed.equals(java.nio.file.Paths.get("root//", new String[]{"", "part/", leaf})));
        Journal evaluation = new Journal();
        java.nio.file.Path ordered = java.nio.file.Paths.get(evaluated(evaluation, "first", "root"), evaluated(evaluation, "one", "a"), evaluated(evaluation, "two", leaf));
        System.out.println("path:evaluation:" + evaluation.count() + ":" + evaluation.trace() + ":" + Journal.units(ordered.toString()));
        state = one("absolute-later", "", new String[]{"/", leaf}, state);
        state = one("root", "/", new String[]{"", ""}, state);
        state = one("empty", "", new String[]{"", ""}, state);
        state = one("null-first", null, new String[]{"ok"}, state);
        state = one("null-first-and-array", null, null, state);
        state = one("null-array", "ok", null, state);
        state = one("null-segment", "ok", new String[]{"part", null}, state);
        state = one("nul-before-null", "a\u0000b", new String[]{null}, state);
        state = one("surrogate-before-null", "a\ud800b", new String[]{null}, state);
        state = one("nul-first", "a\u0000b", new String[]{}, state);
        state = one("nul-segment", "root", new String[]{"a\u0000b"}, state);
        state = one("high", "a\ud800b", new String[]{}, state);
        state = one("low", "a\udc00b", new String[]{}, state);
        state = one("pair", "a\ud800\udc00b", new String[]{}, state);
        state = one("bmp", "\u03b1", new String[]{leaf}, state);
        System.out.println("path:state:" + state);
        return state;
    }
}
