package campaign.business.inventory;

import campaign.business.domain.Product;
import campaign.business.order.Order;
import campaign.business.order.OrderLine;
import java.util.LinkedHashMap;
import java.util.Map;
import org.apache.commons.lang3.mutable.MutableInt;

public final class Inventory {
    private final Map<String, Product> products = new LinkedHashMap<>();
    private final Map<String, MutableInt> available = new LinkedHashMap<>();
    private final Map<String, Integer> original = new LinkedHashMap<>();
    private final Map<String, Integer> added = new LinkedHashMap<>();

    public void add(Product product, int count) {
        products.put(product.id(), product);
        available.put(product.id(), new MutableInt(count));
        original.put(product.id(), count);
        added.put(product.id(), 0);
    }

    public Product product(String sku) {
        return products.get(sku);
    }

    public String reserve(Order order) {
        Map<String, Integer> needed = new LinkedHashMap<>();
        for (OrderLine line : order.lines()) {
            if (line.quantity() <= 0 || line.sku().isEmpty()) {
                return "INVALID_LINE";
            }
            if (!products.containsKey(line.sku())) {
                return "UNKNOWN_SKU";
            }
            needed.put(line.sku(), needed.getOrDefault(line.sku(), 0) + line.quantity());
        }
        if (needed.isEmpty()) {
            return "EMPTY_ORDER";
        }
        for (Map.Entry<String, Integer> item : needed.entrySet()) {
            if (item.getValue() > available.get(item.getKey()).intValue()) {
                return "OUT_OF_STOCK";
            }
        }
        for (Map.Entry<String, Integer> item : needed.entrySet()) {
            available.get(item.getKey()).subtract(item.getValue());
        }
        return "OK";
    }

    public void release(Order order) {
        for (OrderLine line : order.lines()) {
            available.get(line.sku()).add(line.quantity());
        }
    }

    public void restock(String sku, int count) {
        available.get(sku).add(count);
        added.put(sku, added.get(sku) + count);
    }

    public int available(String sku) {
        return available.get(sku).intValue();
    }

    public boolean balanced(Iterable<Order> orders) {
        Map<String, Integer> sold = new LinkedHashMap<>();
        for (String sku : products.keySet()) {
            sold.put(sku, 0);
        }
        for (Order order : orders) {
            if (!order.cancelled()) {
                for (OrderLine line : order.lines()) {
                    sold.put(line.sku(), sold.get(line.sku()) + line.quantity());
                }
            }
        }
        for (String sku : products.keySet()) {
            if (available.get(sku).intValue() + sold.get(sku) != original.get(sku) + added.get(sku)) {
                return false;
            }
        }
        return true;
    }
}
