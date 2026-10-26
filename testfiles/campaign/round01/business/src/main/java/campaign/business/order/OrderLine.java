package campaign.business.order;

import java.util.Locale;

public final class OrderLine {
    private final String sku;
    private final int quantity;

    public OrderLine(String rawSku, int quantity) {
        String clean = rawSku == null ? "" : rawSku.trim();
        this.sku = clean.toUpperCase(Locale.ROOT);
        this.quantity = quantity;
    }

    public String sku() {
        return sku;
    }

    public int quantity() {
        return quantity;
    }
}
