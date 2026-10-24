package campaign.business.order;

import campaign.business.domain.Product;
import campaign.business.inventory.Inventory;
import campaign.business.ledger.Ledger;
import campaign.business.pricing.PriceRule;
import java.util.LinkedHashMap;
import java.util.Map;

public final class OrderEngine {
    private final Inventory inventory;
    private final Ledger ledger;
    private final Map<String, Order> orders = new LinkedHashMap<>();

    public OrderEngine(Inventory inventory, Ledger ledger) {
        this.inventory = inventory;
        this.ledger = ledger;
    }

    public String place(Order order) {
        if (order.id() == null || order.id().isBlank()) {
            return "BAD_ID";
        }
        if (orders.containsKey(order.id())) {
            return "DUPLICATE";
        }
        String reserved = inventory.reserve(order);
        if (!reserved.equals("OK")) {
            return reserved;
        }
        PriceRule rule = order.customer().priceRule();
        long cents = 0;
        for (OrderLine line : order.lines()) {
            Product product = inventory.product(line.sku());
            cents += rule.charge(product, line.quantity());
        }
        if (!ledger.charge(order, cents)) {
            inventory.release(order);
            return "NO_CREDIT";
        }
        order.markPlaced(cents);
        orders.put(order.id(), order);
        return "PLACED:" + cents;
    }

    public String cancel(String id) {
        Order order = orders.get(id);
        if (order == null) {
            return "MISSING";
        }
        if (order.cancelled()) {
            return "ALREADY_CANCELLED";
        }
        ledger.refund(order);
        inventory.release(order);
        order.markCancelled();
        return "REFUNDED:" + order.charged();
    }

    public Iterable<Order> orders() {
        return orders.values();
    }

    public int size() {
        return orders.size();
    }
}
