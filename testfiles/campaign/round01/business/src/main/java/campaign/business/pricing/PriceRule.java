package campaign.business.pricing;

import campaign.business.domain.Product;

public interface PriceRule {
    long charge(Product product, int quantity);
}
