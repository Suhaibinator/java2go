package campaign.business2.ledger;

import campaign.business2.dispatch.Shipment;
import java.util.LinkedHashMap;
import java.util.Map;
import org.apache.commons.codec.binary.Hex;
import org.apache.commons.lang3.mutable.MutableLong;

public final class Ledger {
    private final Map<String, MutableLong> cash = new LinkedHashMap<>();
    private long openingTotal;
    private long journalCode = 1469598103934665603L;
    private int entries;

    public void open(String id, long cents) {
        cash.put(id, new MutableLong(cents));
        openingTotal += cents;
    }

    public boolean pay(String customer, Shipment shipment) {
        long cents = shipment.settledAmount();
        if (cash.get(customer).longValue() < cents) {
            return false;
        }
        transfer(shipment.id(), customer, "HOUSE", cents);
        return true;
    }

    public void refund(String customer, Shipment shipment) {
        transfer(shipment.id(), "HOUSE", customer, shipment.settledAmount());
    }

    private void transfer(String id, String from, String to, long cents) {
        cash.get(from).subtract(cents);
        cash.get(to).add(cents);
        entries++;
        for (int i = 0; i < id.length(); i++) {
            journalCode = (journalCode ^ id.charAt(i)) * 1099511628211L;
        }
        journalCode = (journalCode ^ cents) * 1099511628211L;
        journalCode = (journalCode ^ from.length() ^ (to.length() << 8)) * 1099511628211L;
    }

    public String auditHex() {
        byte[] data = new byte[8];
        for (int i = 0; i < 8; i++) {
            data[i] = (byte) (journalCode >>> (8 * (7 - i)));
        }
        return Hex.encodeHexString(data);
    }

    public long cash(String id) {
        return cash.get(id).longValue();
    }

    public int entries() {
        return entries;
    }

    public boolean balanced() {
        long sum = 0;
        for (MutableLong amount : cash.values()) {
            if (amount.longValue() < 0) {
                return false;
            }
            sum += amount.longValue();
        }
        return sum == openingTotal;
    }
}
