package snapshot.model;

public final class Entry {
    public final String orderId;
    public final int delta;
    public final String parsedBy;
    public Entry(String orderId, int delta, String parsedBy) {
        this.orderId = orderId; this.delta = delta; this.parsedBy = parsedBy;
    }
}
