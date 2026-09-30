package prereq.factory.domain;

import org.apache.commons.lang3.mutable.MutableInt;

public final class Order {
    private final String sku;
    private final int units;
    private final MutableInt fulfilled = new MutableInt();
    private final MutableInt allocations = new MutableInt();

    public Order(String sku, int units) {
        this.sku = sku;
        this.units = units;
    }

    public boolean allocate(int count) {
        if (count < 0 || fulfilled.intValue() + count > units) {
            return false;
        }
        fulfilled.add(count);
        allocations.increment();
        return true;
    }

    public String snapshot() {
        return sku + ":" + fulfilled.intValue() + "/" + units + ":" + allocations.intValue();
    }
}
