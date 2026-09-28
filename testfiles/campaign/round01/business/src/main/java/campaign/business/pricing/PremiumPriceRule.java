package campaign.business.pricing;

import campaign.business.domain.Product;

public final class PremiumPriceRule extends StandardPriceRule {
    @Override
    public long charge(Product product, int quantity) {
        return super.charge(product, quantity) * 9 / 10;
    }
}
