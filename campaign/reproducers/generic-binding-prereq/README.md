# Generic binding and overload prerequisite

Queued supplemental Java 21 application. It does not replace the frozen full
campaign applications or their expected outputs. It uses the real, pinned
Apache Commons Lang 3.20.0 `MutableInt` and `MutableLong` implementations;
`dependency-contract.json` lists the exact source-only closure for later
transpiler testing. No dependency source or binary is vendored here.

The workflow creates two ledgers with the same runtime `Integer` opening but
different constructor routes: one direct `Integer` call, and one through the
agent's generic `<Seed extends Number>` helper into the ledger's differently
named generic `<Opening extends Number>` constructor. `Ledger<Balance extends
Number>` and `Agent<Balance extends CharSequence>` deliberately reuse the class
parameter spelling while owning different bounds. The agent's generic
`<Dispatch>` method calls the ledger's `<Adjustment>` method. Calls with
primitive `int`, boxed `Integer`, `Long`, and a `Number` static view select
different overloads, then mutate Commons-backed totals and counters. The
rejected negative entry touches its input and records a rejected route while
leaving the committed total unchanged.

Compile with the explicit JDK 21 and pinned binary jar, then run seeds 17, 41,
and 97 in three fresh JVM processes each. The adjacent private runner prepares
these commands but must not run until the campaign coordinator assigns a heavy
slot. No oracle or Go result has been accepted yet.
