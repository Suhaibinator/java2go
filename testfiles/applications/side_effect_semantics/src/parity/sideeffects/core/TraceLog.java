package parity.sideeffects.core;

public final class TraceLog {
    private String value;

    public TraceLog() {
        value = "";
    }

    public void add(String token) {
        value = value + token;
    }

    public boolean decision(String token, boolean result) {
        add(token);
        return result;
    }

    public int number(String token, int result) {
        add(token);
        return result;
    }

    public String value() {
        return value;
    }
}
