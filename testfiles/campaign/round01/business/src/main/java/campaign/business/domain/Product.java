package campaign.business.domain;

public final class Product extends Entity<String> {
    private final long cents;

    public Product(String sku, long cents) {
        super(sku);
        this.cents = cents;
    }

    public long cents() {
        return cents;
    }
}
