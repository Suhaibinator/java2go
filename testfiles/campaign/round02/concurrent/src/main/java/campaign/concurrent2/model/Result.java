package campaign.concurrent2.model;

public final class Result {
    private final String id;
    private final String hex;
    private final int byteCount;

    public Result(String id, String hex, int byteCount) {
        this.id = id;
        this.hex = hex;
        this.byteCount = byteCount;
    }

    public String id() { return id; }
    public String hex() { return hex; }
    public int byteCount() { return byteCount; }

    public String line() {
        return id + "|" + byteCount + "|" + hex;
    }
}
