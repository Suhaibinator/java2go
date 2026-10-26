# Shipment settlement and returns

Seventeen requests are grouped by route and dispatched in typed priority order. Depot holds reserve stock and use `AutoCloseable` cleanup to restore it on route and funding failures. Successful shipments move cash through a ledger; returns refund and restock. The final report checks inventory and cash conservation, hold cleanup, shipment revision, and a hexadecimal journal fingerprint. Seeds 17, 41, and 97 vary customer, SKU, quantity, route, and priority.

The application uses the real pinned Apache Commons Lang `MutableInt`/`MutableLong` implementations for stock and balances, and Apache Commons Codec `Hex` for the dynamic journal fingerprint. `model.Document` constructs `dispatch.Shipment`, while `Shipment` extends `Document<String>`; these packages legally depend on one another. `Shipment` also hides the base quote amount with its settled amount so both values remain observable.
