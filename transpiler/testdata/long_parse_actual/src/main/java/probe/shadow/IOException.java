package probe.shadow;
public final class IOException {
    private static int calls;
    private final String message;
    private final Throwable cause;
    public IOException() { this((String) null, (Throwable) null); }
    public IOException(String message) { this(message, (Throwable) null); }
    public IOException(Throwable cause) { this((String) null, cause); }
    public IOException(String message, Throwable cause) { calls++; this.message = message; this.cause = cause; }
    public String message() { return message; }
    public Throwable cause() { return cause; }
    public static int calls() { return calls; }
}
