package campaign.data2.workflow;

import campaign.data2.domain.Command;
import campaign.data2.domain.LineItem;
import campaign.data2.domain.LocatedCommand;
import campaign.data2.domain.Order;
import campaign.data2.ingest.Issue;
import java.text.Normalizer;
import java.util.ArrayList;
import java.util.Collections;
import java.util.Comparator;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Random;

public final class OrderWorkflow {
    private final Map<String, Order> orders = new LinkedHashMap<>();
    private final List<Issue> issues;
    private int applied;

    public OrderWorkflow(List<Issue> initial) { issues = new ArrayList<>(initial); }

    public void replay(List<LocatedCommand> source, long seed) {
        List<LocatedCommand> shuffled = new ArrayList<>(source);
        Collections.shuffle(shuffled, new Random(seed));
        for (LocatedCommand located : shuffled) {
            try {
                apply(located.command);
                applied++;
            } catch (IllegalArgumentException | IllegalStateException ex) {
                issues.add(new Issue(located.location(), ex.getMessage(), fingerprint(located)));
            } catch (ArithmeticException ex) {
                issues.add(new Issue(located.location(), "OVERFLOW", fingerprint(located)));
            }
        }
    }

    private String fingerprint(LocatedCommand located) {
        Command command = located.command;
        String fields = located.location() + "|" + command.kind + "|" + command.orderId
                + "|" + command.sku + "|" + command.delta;
        return Integer.toHexString(fields.hashCode());
    }

    private void apply(Command command) {
        if (command.orderId == null || command.orderId.isBlank()) throw new IllegalArgumentException("ORDER_ID");
        Order order = orders.get(command.orderId);
        switch (command.kind) {
            case "CREATE":
                if (order != null) throw new IllegalStateException("DUPLICATE");
                if (command.customer == null || command.customer.isBlank()) throw new IllegalArgumentException("CUSTOMER");
                if (command.lines == null || command.lines.isEmpty()) throw new IllegalArgumentException("LINES");
                String customer = Normalizer.normalize(command.customer, Normalizer.Form.NFC);
                for (LineItem line : command.lines) {
                    if (line.sku == null || line.sku.isBlank() || line.title == null || line.quantity <= 0 || line.unitCents < 0)
                        throw new IllegalArgumentException("LINE");
                    line.title = Normalizer.normalize(line.title, Normalizer.Form.NFC);
                }
                orders.put(command.orderId, new Order(command.orderId, customer, command.lines, command.metadata));
                break;
            case "INCREMENT":
                if (order == null) throw new IllegalStateException("MISSING_INCREMENT");
                LineItem target = null;
                for (LineItem line : order.lines) if (line.sku.equals(command.sku)) target = line;
                if (target == null) throw new IllegalArgumentException("SKU");
                int quantity = Math.addExact(target.quantity, command.delta);
                if (quantity <= 0) throw new IllegalArgumentException("QUANTITY");
                int subtotal = Math.addExact(order.subtotalCents, Math.multiplyExact(command.delta, target.unitCents));
                target.quantity = quantity;
                order.subtotalCents = subtotal;
                order.revision++;
                break;
            case "RENAME":
                if (order == null) throw new IllegalStateException("MISSING_RENAME");
                if (command.customer == null || command.customer.isBlank()) throw new IllegalArgumentException("CUSTOMER");
                order.customer = Normalizer.normalize(command.customer, Normalizer.Form.NFC);
                order.revision++;
                break;
            case "TAG":
                if (order == null) throw new IllegalStateException("MISSING_TAG");
                if (command.note == null) throw new IllegalArgumentException("NOTE");
                order.metadata.put("note", command.note);
                order.revision++;
                break;
            case "CANCEL":
                if (order == null) throw new IllegalStateException("MISSING_CANCEL");
                orders.remove(command.orderId);
                break;
            default:
                throw new IllegalArgumentException("KIND");
        }
    }

    public List<Order> sortedOrders() {
        return orders.values().stream().sorted(Comparator.comparing(order -> order.orderId)).toList();
    }
    public List<Issue> issues() { return issues; }
    public int applied() { return applied; }
}
