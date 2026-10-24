package prereq.constant.api;

public final class Trace {
    private static final StringBuilder EVENTS = new StringBuilder();

    private Trace() {}

    public static void add(java.lang.String event) {
        EVENTS.append(event).append('|');
    }

    public static java.lang.String snapshot() {
        return EVENTS.toString();
    }
}
