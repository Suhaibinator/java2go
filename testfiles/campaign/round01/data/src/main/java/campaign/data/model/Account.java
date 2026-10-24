package campaign.data.model;

public final class Account {
    public final String id;
    public String label;
    public int balance;
    public int revision;

    public Account(String id, String label, int balance) {
        this.id = id;
        this.label = label;
        this.balance = balance;
        this.revision = 1;
    }
}
