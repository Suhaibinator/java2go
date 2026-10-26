package campaign.business2.model;

public class Customer implements Pricing {
    private final String id;

    public Customer(String id) {
        this.id = id;
    }

    public String id() {
        return id;
    }

    @Override
    public long charge(long quotedCents) {
        return quotedCents;
    }
}
