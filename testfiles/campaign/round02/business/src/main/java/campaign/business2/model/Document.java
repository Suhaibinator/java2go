package campaign.business2.model;

import campaign.business2.dispatch.Shipment;

public abstract class Document<K> {
    private final K id;
    protected long amount;
    private int revision;

    protected Document(K id, long quotedAmount) {
        this.id = id;
        this.amount = quotedAmount;
    }

    public final K id() {
        return id;
    }

    public final long quotedAmount() {
        return amount;
    }

    protected final void advance() {
        revision++;
    }

    public final int revision() {
        return revision;
    }

    public abstract String state();

    public static Document<String> shipment(String id, long quote) {
        return new Shipment(id, quote);
    }
}
