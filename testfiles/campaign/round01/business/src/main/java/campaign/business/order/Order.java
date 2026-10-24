package campaign.business.order;

import campaign.business.domain.Customer;
import campaign.business.domain.Entity;
import java.util.ArrayList;
import java.util.List;

public final class Order extends Entity<String> {
    private final Customer customer;
    private final List<OrderLine> lines;
    private long charged;
    private boolean cancelled;

    public Order(String id, Customer customer, List<OrderLine> lines) {
        super(id);
        this.customer = customer;
        this.lines = new ArrayList<>(lines);
    }

    public Customer customer() {
        return customer;
    }

    public List<OrderLine> lines() {
        return lines;
    }

    public long charged() {
        return charged;
    }

    public void markPlaced(long cents) {
        charged = cents;
    }

    public boolean cancelled() {
        return cancelled;
    }

    public void markCancelled() {
        cancelled = true;
    }
}
