package campaign.concurrent.model;

public final class Result {
    private final String id;
    private final String encoded;

    public Result(String id, String encoded) {
        this.id = id;
        this.encoded = encoded;
    }

    public String id() {
        return id;
    }

    public String encoded() {
        return encoded;
    }
}
