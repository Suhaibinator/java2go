# Stateful order, inventory, and ledger campaign

Run `campaign.business.app.BusinessApplication` with one integer seed. The fixed prefix covers successful standard and premium sales, insufficient credit, stock shortage, invalid SKU, duplicate ID, refund, repeated refund, missing cancellation, and restocking. Twelve generated orders vary customer, SKU, quantity, and multi-line combinations by seed. Final inventory and ledger invariants are calculated from mutable business state.

The real Apache Commons Lang 3 dependency supplies `MutableInt` inventory counters and `MutableLong` ledger balances; its methods perform every stock and cash transfer. The cross-package import graph includes `order` ↔ `inventory` and `order` ↔ `ledger`, which are legal Java package cycles. The entry point accepts seeds 17, 41, and 97; the oracle files are generated only after three equal JVM runs per seed.
