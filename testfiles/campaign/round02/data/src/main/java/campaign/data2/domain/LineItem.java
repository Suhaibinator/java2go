package campaign.data2.domain;

import com.google.gson.annotations.SerializedName;

public final class LineItem {
    @SerializedName("sku")
    public String sku;
    @SerializedName("title")
    public String title;
    @SerializedName("quantity")
    public int quantity;
    @SerializedName("unit_cents")
    public int unitCents;

    public LineItem() {}

    public LineItem(String sku, String title, int quantity, int unitCents) {
        this.sku = sku;
        this.title = title;
        this.quantity = quantity;
        this.unitCents = unitCents;
    }

    public LineItem copy() {
        return new LineItem(sku, title, quantity, unitCents);
    }
}
