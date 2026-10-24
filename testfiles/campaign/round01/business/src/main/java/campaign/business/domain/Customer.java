package campaign.business.domain;

import campaign.business.pricing.PriceRule;
import campaign.business.pricing.StandardPriceRule;

public class Customer extends Entity<String> {
    public Customer(String id) {
        super(id);
    }

    public PriceRule priceRule() {
        return new StandardPriceRule();
    }
}
