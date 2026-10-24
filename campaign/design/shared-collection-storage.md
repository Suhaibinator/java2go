# Shared collection storage and typed views

Status: proposed prerequisite design, 2026-09-28. No implementation or acceptance
claim. The unchanged frozen Gson application remains the acceptance gate. This
plan records the runtime and compiler contracts required before activating
container specializations in the generic-family lowering.

## Grounding in the locked dependency

The source is Gson 2.14.0 from the project-local locked sources archive
`.campaign/cache/gson-2.14.0-sources.jar`, SHA-256
`a4873f0ef88981cab520c3d7449cd89a68e605f15e9fe46e1aa2c77d5f87eb6f`.
The dependency source and frozen application must remain unchanged.

Observed declarations and operations in the complete archive members:

| Member | Lines | Required behavior |
| --- | --- | --- |
| `com/google/gson/internal/bind/CollectionTypeAdapterFactory.java` | 64–85 | `Adapter<E> extends TypeAdapter<Collection<E>>`; a stored `ObjectConstructor<? extends Collection<E>>` constructs a collection; deserialization adds each element |
| Same member | 92 onward | Serialization iterates the supplied `Collection<E>` and passes each element to its adapter |
| `com/google/gson/internal/bind/MapTypeAdapterFactory.java` | 161–207 | `Adapter<K,V> extends TypeAdapter<Map<K,V>>`; stored constructor produces a map; deserialization inserts keys and values |
| Same member | 215–256 | Serialization iterates `Map.Entry<K,V>` from `entrySet()`, builds temporary `List<JsonElement>` and `List<V>`, and reads their indexed elements |

These are genuine source generic specializations, not permission to implement
Gson adapters by hand. Whole-class translation includes behavior beyond the
particular frozen application's execution path.

## Existing representation and rejection boundary

`transpiler/generic_family_plan.go` inventories source family edges, fields,
method types, body type syntax, anonymous subclasses, and recursively owned
source storage. It correctly declines runtime container specializations that
cannot share storage. Canonicalizing `TypeAdapter<T>` alone cannot make
`TypeAdapter<Collection<E>>` or `TypeAdapter<Map<K,V>>` sound.

The current runtime uses:

- `List[T]` with `[]T`, or a special `ReferenceArray` backing for `Arrays.asList`.
  It has no structural modification counter.
- `Map[K,V]` with typed bucket records, typed ordered records, a typed comparator,
  and a structural modification counter. `Set[T]` embeds `Map[T,bool]`.
- `Iterable[T]` with only `Slice() []T`. Array-backed list iteration already reads
  each slot lazily, but ordinary iteration uses a slice.
- Snapshot map key and entry collections. `MapValuesView` retains the map until
  consumption, then obtains a snapshot. `MapEntry` is currently a copied value.
- Private erased protocols for equality and `putAll`. These establish useful
  operation boundaries but do not provide shared mutable generic storage.

Go pointer reinterpretation, copying on conversion, or aliases to `List[any]`
would not repair Java identity, mutation visibility, or late checkcasts. Merely
changing storage to `[]any` while retaining eager typed getters is also unsound.
For example, a polluted `List<String>` element can be read as `Object`; only a
consumer requiring `String` must fail. Ignoring the result of `set`, `remove`, or
`put` must not add a checkcast that Java does not execute.

## Representation direction

Each Java collection allocation owns a nongeneric storage object containing
raw Java references, its dynamic nominal class, shared `ObjectInfo`, and a
structural version. Generic Go facades reference that same allocation. Converting
a facade must not copy elements or create another Java object identity.

Erased operation entry points perform collection work and return raw references.
The compiler supplies checkcasts or unboxing at the actual consuming boundary,
using the same nominal reference machinery as generated objects and arrays.
Typed facade methods may be convenience APIs only where their result conversion
matches that Java boundary; they must not force eager casts globally.

The following contracts constrain this representation:

- **Identity:** typed, raw, wildcard, Object, and erased-family references to one
  allocation share equality, identity hash, `getClass`, and monitor identity.
  Facade conversion preserves null. Runtime descriptors must distinguish actual
  classes where those classes are supported; a Go storage implementation is not
  itself a Java nominal class.
- **Values:** raw storage preserves object identity, including boxed wrappers.
  It must not rebox references. Null conversion preserves the runtime's String
  null sentinel at typed String boundaries while Object reads remain Java null.
- **Writes:** generic collection element types are erased. Ordinary raw writes
  cannot be rejected simply because another facade is `List<String>`.
  Reified array restrictions remain on an `Arrays.asList` backing array.
- **Mutation:** all facades observe additions, replacements, removals, and clear.
  Structural versions belong to shared storage. Replacements must not be
  classified as structural merely because a different facade performed them.
- **Iteration:** cursors retain shared storage and an expected version, read the
  next value at advancement, and follow the selected JDK implementation's
  mutation checks and iterator-removal behavior. Converting to a typed snapshot
  before iteration would introduce early casts and hide later replacements.
- **Map entries and views:** live key/value/entry collections retain the source
  map. Entries retain actual records so supported `setValue` writes reach the
  map. A generic facade is distinct from a Java wrapper: `subList`, unmodifiable
  wrappers, and map views have their own Java identities and policies while
  sharing underlying content. A sublist additionally tracks its range and root
  structural version.
- **Callbacks and ordering:** hashes, equality, comparators, and mapping callbacks
  retain Java execution context, evaluation order, exception behavior, and
  side effects. Sorted storage retains the original comparator; facade
  conversion cannot replace it or change comparison-based key equivalence.
- **Utility methods:** sort, reverse, shuffle, equality, hashing, and bulk copies
  must operate through the shared representation. A newly snapshot-producing
  `Slice()` cannot silently make these methods mutate detached data.

ObjectView or a compiler-emitted collection projection helper must materialize
the requested Go facade after nominal checking, rather than assert invariant Go
pointer types. The exact interface/factory mechanism is still a design decision.
It should reuse existing ObjectInfo and reference projection infrastructure,
without a process-global registry strongly retaining collection instances.

## Proposed ownership

Runtime owner:

- New `stdjava/collection_storage.go`, `collection_views.go`, and
  `collection_iteration.go`.
- Initial list slice: `stdjava/list.go`, `list_constructors.go`, `iterable.go`,
  `collection_values.go`; narrow list helper regions in `collections_common.go`
  and `comparator.go`.
- Coordinated narrow projection hook in `stdjava/reference_arrays.go`, if the
  agreed conversion mechanism requires it.
- Later map slice: `stdjava/map.go`, `map_put_all.go`, `set.go`, plus dedicated
  live-view and entry files as needed.
- New focused runtime tests and uniquely named
  `transpiler/campaign_runtime_collection_*_test.go` JVM differential tests.

Compiler owner:

- Family capability checks and atomic activation, preserving the existing
  refusal until each necessary runtime storage boundary is supported.
- Collection reference conversions, erased invocation-result emission,
  consumer checkcasts/unboxing, and enhanced-for/iterator lowering.
- Corresponding narrow collection intrinsic and type-signature changes, agreed
  before editing shared files.

Existing source-level generic signature metadata must remain intact. A physical
storage decision must not mutate the declarations later needed by reflection.
This is a generated ABI change: historical outputs require regeneration.

## Red-to-green phases

Each phase starts with a reduced JVM oracle and retains that oracle unchanged.
Direct runtime tests supplement rather than replace source differential tests.

1. **Shared ArrayList vertical slice.** A generic source holder/family stores a
   list, projects typed and raw references, mutates through each, and verifies
   shared contents, reference identity, class identity, and monitor identity.
   Include nullable references and repeated boxed-object references. This slice
   proves a real source-family boundary, not just two Go wrappers in isolation.
2. **Delayed consumer conversion.** Pollute a typed list through a raw reference.
   Object reads and discarded operation results succeed; String-consuming reads
   fail at the correct point. Preserve mutation before a return-value checkcast
   failure, argument side effects, null unboxing, and nested collection values.
3. **Iteration and array views.** Check replacement before iterator advancement,
   structural mutation through another facade, iterator removal, enhanced-for,
   fixed-size policies, and actual covariant array-store checks. Ensure utility
   mutations operate on shared content. Test exception timing and partial state
   where sorting or callbacks fail rather than assuming transactionality.
4. **Shared maps and sets.** Verify put/get/remove across facades, retained key
   identity on replacement, missing versus null-valued mappings, bulk operations,
   callback mutation checks, and live views/entries. Reuse shared map storage for
   sets while preserving each collection's own nominal identity.
5. **Sorted storage.** Exercise original comparator identity and callbacks,
   equal-by-comparison keys, null policy, typed comparator bridge failures, and
   mutations observed through generic views. Keep natural and explicit ordering
   distinct.
6. **Family and application gates.** Activate only supported container boundaries
   for a complete reduced collection/map adapter family. Regenerate and verify
   historical applications, then run the unchanged full frozen Gson challenge.
   A focused green slice does not constitute full Gson acceptance.

## Remaining uncertainties and explicit boundaries

- The final facade projection mechanism must handle typed runtime pointers,
  erased Object references, generic Collection/Iterable interfaces, and generated
  source implementations without manufacturing a separate identity.
- Existing collection mappings collapse some JDK implementation distinctions.
  The nominal descriptor and constructor policy must be specified before claiming
  implementation-specific `getClass`, iterator, or modification behavior.
- Existing immutable/unmodifiable helpers are acknowledged approximations.
  Shared storage must not disguise those gaps as newly implemented policies.
- JDK fail-fast iterators are best effort, not a concurrency safety guarantee.
  Tests must use controlled sequential mutation; this plan does not make ordinary
  collections thread-safe. ConcurrentHashMap is a separate representation and
  must not accidentally inherit ordinary-map iteration semantics.
- Raw pollution, wildcard capture, bounded type variables, nested containers,
  bridge methods, method references, and source subclasses need explicit compiler
  boundary tests. A green List slice alone cannot activate the entire TypeAdapter
  family while its Map/Collection dependencies remain unsupported.
- Performance and allocation cost need measurement after semantic correctness.
  Avoid eager snapshots and per-element wrapper allocation, but do not trade away
  Java identity or late failure timing to preserve the old slice fast paths.

