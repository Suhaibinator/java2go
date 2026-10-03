package campaign.data2.domain;

import com.google.gson.annotations.SerializedName;
import java.util.ArrayList;
import java.util.List;
import java.util.TreeMap;

public final class Order {
    @SerializedName("order_id")
    public String orderId;
    @SerializedName("customer")
    public String customer;
    @SerializedName("lines")
    public List<LineItem> lines = new ArrayList<>();
    @SerializedName("metadata")
    public TreeMap<String, String> metadata = new TreeMap<>();
    @SerializedName("revision")
    public int revision;
    @SerializedName("subtotal_cents")
    public int subtotalCents;

    public Order(String orderId, String customer, List<LineItem> source, java.util.Map<String, String> metadata) {
        this.orderId = orderId;
        this.customer = customer;
        for (LineItem item : source) this.lines.add(item.copy());
        if (metadata != null) this.metadata.putAll(metadata);
        this.revision = 1;
        recompute();
    }

    public void recompute() {
        int sum = 0;
        for (LineItem item : lines) {
            sum = Math.addExact(sum, Math.multiplyExact(item.quantity, item.unitCents));
        }
        subtotalCents = sum;
    }
}
