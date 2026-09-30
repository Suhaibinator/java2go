package campaign.business.domain;

import campaign.business.pricing.PremiumPriceRule;
import campaign.business.pricing.PriceRule;

public final class PremiumCustomer extends Customer {
    public PremiumCustomer(String id) {
        super(id);
    }

    @Override
    public PriceRule priceRule() {
        return new PremiumPriceRule();
    }
}
