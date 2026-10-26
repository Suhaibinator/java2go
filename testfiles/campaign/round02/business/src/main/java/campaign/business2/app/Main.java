package campaign.business2.app;

import campaign.business2.dispatch.Coordinator;
import campaign.business2.dispatch.Depot;
import campaign.business2.ledger.Ledger;
import campaign.business2.model.Customer;
import campaign.business2.model.PriorityCustomer;
import campaign.business2.model.Request;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.Map;
import java.util.Random;
import java.util.TreeMap;

public final class Main {
    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        Random random = new Random(seed);
        Depot depot = new Depot();
        depot.add("BOLT", 12, 180);
        depot.add("NUT", 10, 240);
        depot.add("WASHER", 9, 150);
        Ledger ledger = new Ledger();
        ledger.open("HOUSE", 0);
        ledger.open("C0", 2600);
        ledger.open("C1", 3500);
        ledger.open("C2", 450);
        Customer[] customers = {new Customer("C0"), new PriorityCustomer("C1"), new Customer("C2")};
        String[] sku = {"BOLT", "NUT", "WASHER"};
        String[] route = {"E", "W", "X"};
        List<Request> requests = new ArrayList<>();
        requests.add(new Request("F0", customers[0], "BOLT", 2, "E", 2));
        requests.add(new Request("F1", customers[1], "NUT", 3, "W", 3));
        requests.add(new Request("F2", customers[2], "WASHER", 4, "E", 1));
        requests.add(new Request("F3", customers[0], "BOLT", 99, "W", 0));
        requests.add(new Request("F4", customers[0], "TAPE", 1, "E", 1));
        for (int i = 0; i < 12; i++) {
            String id = "S" + (i < 10 ? "0" : "") + i;
            requests.add(new Request(id, customers[random.nextInt(3)], sku[random.nextInt(3)],
                    1 + random.nextInt(4), route[random.nextInt(3)], random.nextInt(4)));
        }

        Map<String, List<Request>> groups = new TreeMap<>();
        for (Request request : requests) {
            groups.computeIfAbsent(request.route(), key -> new ArrayList<>()).add(request);
        }
        Comparator<Request> priorityOrder = Comparator.comparingInt(Request::priority).reversed()
                .thenComparing(Request::id);
        Coordinator coordinator = new Coordinator(depot, ledger);
        System.out.println("seed=" + seed);
        for (Map.Entry<String, List<Request>> group : groups.entrySet()) {
            List<Request> ordered = group.getValue();
            ordered.sort(priorityOrder);
            System.out.println("route " + group.getKey() + " count=" + ordered.size());
            for (Request request : ordered) {
                System.out.println(request.id() + ":" + request.customer().id() + ":" + request.sku()
                        + "x" + request.quantity() + "=" + coordinator.dispatch(request));
            }
        }
        System.out.println("duplicate F0=" + coordinator.dispatch(requests.get(0)));
        System.out.println("return F1=" + coordinator.returnShipment("F1"));
        System.out.println("return F1=" + coordinator.returnShipment("F1"));
        System.out.println("return missing=" + coordinator.returnShipment("MISSING"));
        System.out.println("stock=" + depot.available("BOLT") + "," + depot.available("NUT")
                + "," + depot.available("WASHER"));
        System.out.println("cash=" + ledger.cash("C0") + "," + ledger.cash("C1") + ","
                + ledger.cash("C2") + "," + ledger.cash("HOUSE"));
        System.out.println("holds=" + depot.closed() + "," + depot.committed() + "," + depot.released()
                + ",shipments=" + coordinator.size() + ",entries=" + ledger.entries());
        System.out.println("audit=" + ledger.auditHex() + ",balanced=" + coordinator.balanced());
    }
}
