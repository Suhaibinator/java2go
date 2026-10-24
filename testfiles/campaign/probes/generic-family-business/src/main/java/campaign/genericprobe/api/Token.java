package campaign.genericprobe.api;

public final class Token<T> {
    private final String key;

    public Token(String key) {
        this.key = key;
    }

    public String key() {
        return key;
    }
}
