package campaign.business.app;

import campaign.business.domain.Customer;
import campaign.business.domain.PremiumCustomer;
import campaign.business.domain.Product;
import campaign.business.inventory.Inventory;
import campaign.business.ledger.Ledger;
import campaign.business.order.Order;
import campaign.business.order.OrderEngine;
import campaign.business.order.OrderLine;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.Random;

public final class BusinessApplication {
    private static Order order(String id, Customer customer, OrderLine... lines) {
        return new Order(id, customer, Arrays.asList(lines));
    }

    private static void place(OrderEngine engine, Order order) {
        System.out.println(order.id() + "=" + engine.place(order));
    }

    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        Random random = new Random(seed);
        Inventory inventory = new Inventory();
        inventory.add(new Product("INK", 250), 13);
        inventory.add(new Product("PAPER", 400), 11);
        inventory.add(new Product("CLIP", 125), 17);
        Ledger ledger = new Ledger();
        ledger.open("HOUSE", 0);
        ledger.open("ALICE", 10000);
        ledger.open("BOB", 7000);
        ledger.open("CARA", 500);
        Customer alice = new Customer("ALICE");
        Customer bob = new PremiumCustomer("BOB");
        Customer cara = new Customer("CARA");
        OrderEngine engine = new OrderEngine(inventory, ledger);

        System.out.println("seed=" + seed);
        place(engine, order("O01", alice, new OrderLine(" ink ", 2), new OrderLine("paper", 1)));
        place(engine, order("O02", bob, new OrderLine("  clip  ", 3), new OrderLine("INK", 1)));
        place(engine, order("O03", cara, new OrderLine("PAPER", 2)));
        place(engine, order("O04", alice, new OrderLine("ink", 99)));
        place(engine, order("O05", alice, new OrderLine(" ", 1)));
        place(engine, order("O06", alice, new OrderLine("glue", 1)));
        place(engine, order("O01", bob, new OrderLine("clip", 1)));
        System.out.println("cancel O02=" + engine.cancel("O02"));
        System.out.println("cancel O02=" + engine.cancel("O02"));
        System.out.println("cancel O99=" + engine.cancel("O99"));
        inventory.restock("INK", 2);
        System.out.println("restock INK=2");

        Customer[] customers = {alice, bob, cara};
        String[] sku = {"ink", "paper", "clip"};
        List<String> placed = new ArrayList<>();
        for (int i = 0; i < 12; i++) {
            String id = "R" + (i < 10 ? "0" : "") + i;
            Customer customer = customers[random.nextInt(customers.length)];
            String raw = random.nextBoolean() ? " " + sku[random.nextInt(sku.length)] + " " : sku[random.nextInt(sku.length)];
            int count = 1 + random.nextInt(4);
            OrderLine first = new OrderLine(raw, count);
            Order candidate;
            if (i % 4 == 0) {
                OrderLine second = new OrderLine(sku[random.nextInt(sku.length)], 1 + random.nextInt(2));
                candidate = order(id, customer, first, second);
            } else {
                candidate = order(id, customer, first);
            }
            String result = engine.place(candidate);
            System.out.println(id + ":" + customer.id() + ":" + first.sku() + "x" + first.quantity() + "=" + result);
            if (result.startsWith("PLACED")) {
                placed.add(id);
            }
            if (i == 5 && !placed.isEmpty()) {
                String previous = placed.get(0);
                System.out.println("cancel " + previous + "=" + engine.cancel(previous));
            }
        }
        place(engine, order("O07", alice, new OrderLine("clip", 0)));
        System.out.println("stock=" + inventory.available("INK") + ","
                + inventory.available("PAPER") + "," + inventory.available("CLIP"));
        System.out.println("balances=" + ledger.balance("ALICE") + "," + ledger.balance("BOB")
                + "," + ledger.balance("CARA") + "," + ledger.balance("HOUSE"));
        System.out.println("orders=" + engine.size() + ",entries=" + ledger.entries()
                + ",checksum=" + ledger.checksum());
        System.out.println("invariants=" + inventory.balanced(engine.orders()) + "," + ledger.balanced());
    }
}
