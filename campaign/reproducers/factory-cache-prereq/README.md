# Generic factory method and cache prerequisite

This queued supplemental Java 21 application targets the frozen full Gson
blocker: a nongeneric source `Factory` interface declaring a generic
`<T> Adapter<T> create(Context, Token<T>)` method. `Context` calls it through a
list of interface references and caches returned adapters across `Document`
and `Order` lookups. A generic `DocumentFactory<Prefix>` implementation, a
named `OrderFactory`, and an anonymous null-returning implementation exercise
method-bound versus owner-bound parameters, erased dispatch bridges, null
results, and adapter identity. `Factory` and `Context` form a legal Java
cross-package dependency cycle.

The workflow mutates two document revisions and two order allocations, then
tries an intentionally mismatched typed cache view. That view must fail at the
adapter bridge before changing the order or document call counters. A second
context reuses the factory objects but creates a separate document adapter.
Repeated unknown lookups must probe all factories instead of caching null.
Outputs observe state and order across seeds 17, 41, and 97.

The business counters use real, pinned Apache Commons Lang 3.20.0
`MutableInt`. `dependency-contract.json` lists the whole source-only closure;
no dependency binary or implementation source is vendored. This probe does not
replace full Gson or modify any frozen application. JDK oracle and transpiler
parity are both UNRUN until a bounded execution slot is assigned.
