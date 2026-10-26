package campaign.business2.dispatch;

import campaign.business2.model.Document;

public final class Shipment extends Document<String> {
    // A settled amount may differ from the quote inherited from Document.
    private long amount;
    private boolean delivered;
    private boolean returned;

    public Shipment(String id, long quote) {
        super(id, quote);
    }

    public void deliver(long settledAmount) {
        amount = settledAmount;
        delivered = true;
        advance();
    }

    public void markReturned() {
        returned = true;
        advance();
    }

    public long settledAmount() {
        return amount;
    }

    public boolean returned() {
        return returned;
    }

    @Override
    public String state() {
        if (returned) {
            return "RETURNED";
        }
        return delivered ? "DELIVERED" : "CREATED";
    }
}
