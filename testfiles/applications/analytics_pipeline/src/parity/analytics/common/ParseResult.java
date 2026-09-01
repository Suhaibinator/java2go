package parity.analytics.common;

public class ParseResult<T> {
    private boolean accepted;
    private T value;
    private String code;
    private String detail;

    public ParseResult(boolean accepted, T value, String code, String detail) {
        this.accepted = accepted;
        this.value = value;
        this.code = code;
        this.detail = detail;
    }

    public boolean isAccepted() {
        return this.accepted;
    }

    public T getValue() {
        return this.value;
    }

    public String getCode() {
        return this.code;
    }

    public String getDetail() {
        return this.detail;
    }
}
