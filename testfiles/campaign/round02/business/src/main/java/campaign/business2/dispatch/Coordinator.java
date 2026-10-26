package campaign.business2.dispatch;

import campaign.business2.ledger.Ledger;
import campaign.business2.model.Document;
import campaign.business2.model.Request;
import java.util.LinkedHashMap;
import java.util.Map;

public final class Coordinator {
    private final Depot depot;
    private final Ledger ledger;
    private final Map<String, Shipment> shipments = new LinkedHashMap<>();
    private final Map<String, Integer> unitsById = new LinkedHashMap<>();
    private final Map<String, String> skuById = new LinkedHashMap<>();
    private final Map<String, String> customerById = new LinkedHashMap<>();

    public Coordinator(Depot depot, Ledger ledger) {
        this.depot = depot;
        this.ledger = ledger;
    }

    public String dispatch(Request request) {
        if (shipments.containsKey(request.id())) {
            return "DUPLICATE";
        }
        try {
            long quote = (long) depot.price(request.sku()) * request.quantity();
            Document<String> document = Document.shipment(request.id(), quote);
            Shipment shipment = (Shipment) document;
            try (Depot.Hold hold = depot.hold(request.sku(), request.quantity())) {
                if (!request.route().equals("E") && !request.route().equals("W")) {
                    throw new DispatchFailure("BAD_ROUTE");
                }
                shipment.deliver(request.customer().charge(document.quotedAmount()));
                if (!ledger.pay(request.customer().id(), shipment)) {
                    throw new DispatchFailure("NO_FUNDS");
                }
                hold.accept();
                shipments.put(request.id(), shipment);
                unitsById.put(request.id(), request.quantity());
                skuById.put(request.id(), request.sku());
                customerById.put(request.id(), request.customer().id());
                return "SENT:" + document.quotedAmount() + ":" + shipment.settledAmount();
            }
        } catch (DispatchFailure failure) {
            return failure.getMessage();
        }
    }

    public String returnShipment(String id) {
        Shipment shipment = shipments.get(id);
        if (shipment == null) {
            return "MISSING";
        }
        if (shipment.returned()) {
            return "ALREADY_RETURNED";
        }
        ledger.refund(customerById.get(id), shipment);
        depot.returnUnits(skuById.get(id), unitsById.get(id));
        shipment.markReturned();
        return "REFUNDED:" + shipment.settledAmount() + ":" + shipment.revision();
    }

    public int size() {
        return shipments.size();
    }

    public boolean balanced() {
        return depot.balanced(shipments.values(), unitsById, skuById) && ledger.balanced();
    }
}
