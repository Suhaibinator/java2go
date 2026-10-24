package campaign.concurrent2.model;

public final class Job {
    private final String id;
    private final String payload;

    public Job(String id, String payload) {
        this.id = id;
        this.payload = payload;
    }

    public String id() { return id; }
    public String payload() { return payload; }
}
