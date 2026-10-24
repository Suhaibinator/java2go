package probe.nometa;

public final class Lookalike {
    public int messageCalls;
    public int nameCalls;
    public int errorCalls;
    public int hashCalls;

    public String throwableTypeName() {
        nameCalls++;
        return "Pretend";
    }

    public String message() {
        messageCalls++;
        return "ordinary-message";
    }

    public String error() {
        errorCalls++;
        return "ordinary-error";
    }

    @Override
    public int hashCode() {
        hashCalls++;
        return 42;
    }
}
