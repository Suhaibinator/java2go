package campaign.business2.dispatch;

import java.util.LinkedHashMap;
import java.util.Map;
import org.apache.commons.lang3.mutable.MutableInt;

public final class Depot {
    private final Map<String, MutableInt> stock = new LinkedHashMap<>();
    private final Map<String, Integer> original = new LinkedHashMap<>();
    private final Map<String, Integer> unitPrice = new LinkedHashMap<>();
    private int closed;
    private int released;
    private int committed;

    public void add(String sku, int units, int cents) {
        stock.put(sku, new MutableInt(units));
        original.put(sku, units);
        unitPrice.put(sku, cents);
    }

    public int price(String sku) throws DispatchFailure {
        Integer value = unitPrice.get(sku);
        if (value == null) {
            throw new DispatchFailure("UNKNOWN_SKU");
        }
        return value;
    }

    public Hold hold(String sku, int units) throws DispatchFailure {
        if (units <= 0) {
            throw new DispatchFailure("BAD_QUANTITY");
        }
        MutableInt available = stock.get(sku);
        if (available == null) {
            throw new DispatchFailure("UNKNOWN_SKU");
        }
        if (available.intValue() < units) {
            throw new DispatchFailure("OUT_OF_STOCK");
        }
        available.subtract(units);
        return new Hold(sku, units);
    }

    public void returnUnits(String sku, int units) {
        stock.get(sku).add(units);
    }

    public int available(String sku) {
        return stock.get(sku).intValue();
    }

    public int closed() {
        return closed;
    }

    public int released() {
        return released;
    }

    public int committed() {
        return committed;
    }

    public boolean balanced(Iterable<Shipment> shipments, Map<String, Integer> unitsById, Map<String, String> skuById) {
        Map<String, Integer> shipped = new LinkedHashMap<>();
        for (String sku : stock.keySet()) {
            shipped.put(sku, 0);
        }
        for (Shipment shipment : shipments) {
            if (!shipment.returned()) {
                String sku = skuById.get(shipment.id());
                shipped.put(sku, shipped.get(sku) + unitsById.get(shipment.id()));
            }
        }
        for (String sku : stock.keySet()) {
            if (available(sku) + shipped.get(sku) != original.get(sku)) {
                return false;
            }
        }
        return true;
    }

    public final class Hold implements AutoCloseable {
        private final String sku;
        private final int units;
        private boolean accepted;
        private boolean ended;

        private Hold(String sku, int units) {
            this.sku = sku;
            this.units = units;
        }

        public void accept() {
            accepted = true;
        }

        @Override
        public void close() {
            if (ended) {
                return;
            }
            ended = true;
            closed++;
            if (accepted) {
                committed++;
            } else {
                stock.get(sku).add(units);
                released++;
            }
        }
    }
}
