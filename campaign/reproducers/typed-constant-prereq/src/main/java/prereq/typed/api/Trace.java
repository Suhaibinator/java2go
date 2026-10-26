package prereq.typed.api;

public final class Trace {
    private static final StringBuilder EVENTS = new StringBuilder();

    private Trace() {}

    public static void add(java.lang.String value) {
        EVENTS.append(value).append('|');
    }

    public static java.lang.String snapshot() {
        return EVENTS.toString();
    }
}
