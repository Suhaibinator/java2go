package parity.resourcesuppression;

final class TraceHolder {
    private String value;

    TraceHolder() {
        value = "";
    }

    void append(String piece) {
        value = value + piece;
    }

    String get() {
        return value;
    }
}

final class FailingResource implements AutoCloseable {
    private final TraceHolder trace;
    private final String name;
    private final boolean fail;

    FailingResource(TraceHolder trace, String name, boolean fail) {
        this.trace = trace;
        this.name = name;
        this.fail = fail;
    }

    @Override
    public void close() {
        trace.append(name);
        if (fail) {
            throw new IllegalStateException("close-" + name);
        }
    }
}

final class SharedException extends RuntimeException {
    SharedException(String message) {
        super(message);
    }
}

final class ExceptionResource implements AutoCloseable {
    private final SharedException exception;

    ExceptionResource(SharedException exception) {
        this.exception = exception;
    }

    @Override
    public void close() {
        throw exception;
    }
}

public final class ResourceSuppressionSemanticsApplication {
    private ResourceSuppressionSemanticsApplication() {}

    private static void bodyPrimary() {
        TraceHolder trace = new TraceHolder();
        try (FailingResource first = new FailingResource(trace, "1", true);
             FailingResource second = new FailingResource(trace, "2", true)) {
            trace.append("B");
            throw new IllegalArgumentException("body");
        } catch (IllegalArgumentException ex) {
            System.out.println("BODY_TRACE=" + trace.get());
            System.out.println("BODY_PRIMARY=" + ex.getMessage());
            System.out.println("BODY_SUPPRESSED=" + ex.getSuppressed().length + ":" +
                    ex.getSuppressed()[0].getMessage() + ":" +
                    ex.getSuppressed()[1].getMessage());
        }
    }

    private static void closePrimary() {
        TraceHolder trace = new TraceHolder();
        try (FailingResource first = new FailingResource(trace, "1", true);
             FailingResource second = new FailingResource(trace, "2", true)) {
            trace.append("B");
        } catch (IllegalStateException ex) {
            System.out.println("CLOSE_TRACE=" + trace.get());
            System.out.println("CLOSE_PRIMARY=" + ex.getMessage());
            System.out.println("CLOSE_SUPPRESSED=" + ex.getSuppressed().length + ":" +
                    ex.getSuppressed()[0].getMessage());
        }
    }

    private static String returnVsClose() {
        TraceHolder trace = new TraceHolder();
        try (FailingResource resource = new FailingResource(trace, "C", true)) {
            trace.append("B");
            return "wrong-return";
        } catch (IllegalStateException ex) {
            return trace.get() + ":" + ex.getMessage();
        }
    }

    private static void selfSuppression() {
        SharedException shared = new SharedException("shared");
        try (ExceptionResource resource = new ExceptionResource(shared)) {
            throw shared;
        } catch (IllegalArgumentException ex) {
            System.out.println("SELF=" + ex.getMessage() + ":" +
                    ex.getSuppressed().length + ":" +
                    (ex.getCause() == shared) + ":" +
                    ex.getCause().getMessage());
        }
    }

    private static void distinctEqualValues() {
        SharedException body = new SharedException("same");
        SharedException close = new SharedException("same");
        try (ExceptionResource resource = new ExceptionResource(close)) {
            throw body;
        } catch (SharedException ex) {
            System.out.println("DISTINCT=" + (ex == body) + ":" +
                    ex.getSuppressed().length + ":" +
                    (ex.getSuppressed()[0] == close));
        }
    }

    public static void main(String[] args) {
        bodyPrimary();
        closePrimary();
        System.out.println("RETURN=" + returnVsClose());
        selfSuppression();
        distinctEqualValues();
    }
}
