package campaign.business.pricing;

import campaign.business.domain.Product;

public class StandardPriceRule implements PriceRule {
    @Override
    public long charge(Product product, int quantity) {
        return product.cents() * quantity;
    }
}
