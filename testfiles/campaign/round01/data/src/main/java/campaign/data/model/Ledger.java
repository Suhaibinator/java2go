package campaign.data.model;

import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Random;
import org.apache.commons.codec.digest.DigestUtils;

public final class Ledger {
    private final Map<String, Account> accounts = new LinkedHashMap<>();
    private final List<Reject> rejects;
    private int applied;

    public Ledger(List<Reject> initialRejects) {
        rejects = new ArrayList<>(initialRejects);
    }

    public void replay(List<Event> source, long seed) {
        List<Event> events = new ArrayList<>(source);
        Collections.shuffle(events, new Random(seed));
        for (Event event : events) {
            try {
                apply(event);
                applied++;
            } catch (IllegalStateException ex) {
                reject(event, ex.getMessage());
            } catch (ArithmeticException ex) {
                reject(event, "OVERFLOW");
            }
        }
    }

    private void apply(Event event) {
        Account current = accounts.get(event.account);
        switch (event.operation) {
            case "PUT":
                accounts.put(event.account, new Account(event.account, event.label, event.amount));
                break;
            case "ADD":
                if (current == null) throw new IllegalStateException("MISSING_ADD");
                if (!current.label.equals(event.label)) throw new IllegalStateException("LABEL");
                current.balance = Math.addExact(current.balance, event.amount);
                current.revision++;
                break;
            case "DEL":
                if (current == null) throw new IllegalStateException("MISSING_DEL");
                if (!current.label.equals(event.label)) throw new IllegalStateException("LABEL");
                accounts.remove(event.account);
                break;
            default:
                throw new IllegalStateException("UNKNOWN");
        }
    }

    private void reject(Event event, String code) {
        String key = event.account + "|" + event.operation + "|" + event.label + "|" + event.amount;
        rejects.add(new Reject(event.location(), code, DigestUtils.sha256Hex(key).substring(0, 12)));
    }

    public Map<String, Account> accounts() { return accounts; }
    public List<Reject> rejects() { return rejects; }
    public int applied() { return applied; }
}
