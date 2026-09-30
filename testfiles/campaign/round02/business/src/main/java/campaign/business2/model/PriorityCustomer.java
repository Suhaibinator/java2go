package campaign.business2.model;

public final class PriorityCustomer extends Customer {
    public PriorityCustomer(String id) {
        super(id);
    }

    @Override
    public long charge(long quotedCents) {
        return quotedCents * 9 / 10 + 25;
    }
}
