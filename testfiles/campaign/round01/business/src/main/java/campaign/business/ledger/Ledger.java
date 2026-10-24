package campaign.business.ledger;

import campaign.business.order.Order;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import org.apache.commons.lang3.mutable.MutableLong;

public final class Ledger {
    private final Map<String, MutableLong> balances = new LinkedHashMap<>();
    private final List<Transfer> journal = new ArrayList<>();
    private long initialTotal;

    public void open(String account, long cents) {
        balances.put(account, new MutableLong(cents));
        initialTotal += cents;
    }

    public boolean charge(Order order, long cents) {
        if (balances.get(order.customer().id()).longValue() < cents) {
            return false;
        }
        move(order.id(), order.customer().id(), "HOUSE", cents);
        return true;
    }

    public void refund(Order order) {
        move(order.id(), "HOUSE", order.customer().id(), order.charged());
    }

    private void move(String orderId, String from, String to, long cents) {
        balances.get(from).subtract(cents);
        balances.get(to).add(cents);
        journal.add(new Transfer(orderId, from, to, cents));
    }

    public long balance(String account) {
        return balances.get(account).longValue();
    }

    public int entries() {
        return journal.size();
    }

    public long checksum() {
        long result = 0;
        for (Transfer entry : journal) {
            result = result * 31 + entry.cents();
            result = result * 31 + entry.orderId().charAt(entry.orderId().length() - 1);
            result = result * 31 + entry.from().length() - entry.to().length();
        }
        return result;
    }

    public boolean balanced() {
        long total = 0;
        for (MutableLong balance : balances.values()) {
            total += balance.longValue();
            if (balance.longValue() < 0) {
                return false;
            }
        }
        return total == initialTotal;
    }

    private record Transfer(String orderId, String from, String to, long cents) {}
}
