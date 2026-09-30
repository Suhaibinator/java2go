package prereq.factory.api;

public final class Token<T> {
    private final String kind;

    public Token(String kind) {
        this.kind = kind;
    }

    public String kind() {
        return kind;
    }
}
