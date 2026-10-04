package campaign.data.model;

public final class Event {
    public final String source;
    public final int line;
    public final String account;
    public final String operation;
    public final String label;
    public final int amount;

    public Event(String source, int line, String account, String operation, String label, int amount) {
        this.source = source;
        this.line = line;
        this.account = account;
        this.operation = operation;
        this.label = label;
        this.amount = amount;
    }

    public String location() {
        return source + ":" + line;
    }
}
