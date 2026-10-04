# Source-object construction: read-only audit

**The next bounded codegen candidate is to omit redundant interface fields when the emitted concrete method set already satisfies the interface.** For allocation count, prioritize a closure-free per-object receiver, then an in-place identity record. These are unmeasured proposals. No build, test, benchmark, heap process or implementation change was made for this audit.

## Existing evidence

The passing hierarchy artifact `.campaign/performance/object-identity-20260927T235324/generated/ObjectIdentityProbe.go` allocates separate root/middle/leaf structs at lines 82/147/209, then shared ObjectInfo and its bound receiver closure at line 91. The existing escape log confirms all three struct allocations escape. Its payload adds an array wrapper and backing storage. The runtime study measured **7 allocations per graph construction**, **2 for NewGeneratedObjectInfo**, and **1 for bare NewObjectInfo**. The queue candidate leaves construction at 7. These are previously recorded Go counts, not new measurements or JVM comparisons.

Class descriptors are already registered once outside constructors. Generated view switches return existing subobject pointers. Hoisting class metadata or making view wrappers lazy therefore does not remove an existing repeated allocation here. A receiver-bound closure cannot be shared across instances. Go's interface method table already provides shared dispatch metadata.

The passing round01 data allocation profile mostly identifies Strings, I/O and streams; it does not establish source-object construction as an application bottleneck. Its Codec classes supply structural examples only. Existing evidence paths/hashes are preserved in `evidence.json`.

## Bounded compiler candidate

Frozen `transpiler/declaration.go:137–146` embeds implemented interfaces. `implementedInterfaceTypeExpr` at 819 already skips structural runtime interfaces Runnable/Callable/Comparable, but retains ordinary source interfaces. Default-method interfaces use a separate implementation carrier.

| Passing generated struct | Ordinary embedded interfaces | Evidence |
| --- | --- | --- |
| GraphLeaf | graphTag | Value and its execution-aware method are defined directly. The field is never assigned/read; views return the leaf itself. |
| URLCodec | BinaryEncoder, BinaryDecoder, StringEncoder, StringDecoder | Concrete overloads exist; complete inherited/bridge method coverage must be proven before omission. |
| Hex | BinaryEncoder, BinaryDecoder | Same method-set proof required. No evidence of frequent construction in this workload. |

General rule, without source-name specialization:

1. Resolve the source interface and exclude any inherited default-method carrier.
2. Prove that final concrete/inherited/bridge methods satisfy every required Go signature without the field. Preserve overload renaming, generic substitution, covariance and execution-aware methods.
3. Omit only that anonymous field. Keep nominal registration edges, view results, class descriptors, dispatch receivers and Java field/constructor initialization unchanged.
4. Require a nonzero object layout or a distinct nonzero canonical identity allocation. **Do not turn a marker-only class into a zero-sized Go allocation:** its address may be shared, violating Java identity. Keep the field or existing identity representation until that case is handled.

The examples remain nonzero through a parent pointer or charset field. Interface slots generally occupy two machine words on this Go ABI; omission may reduce object bytes and GC scanning after padding/allocator rounding. This is not a measured byte saving and usually does not remove an allocation event. URLCodec is constructed in loader/writer initialization, so end-to-end impact may be negligible.

## Allocation-count candidates

| Priority | Proposal | Hypothesis and obligations |
| --- | --- | --- |
| 1 | Store the generated receiver interface in ObjectInfo instead of a bound method value | Aim for metadata 2→1 and graph 7→6 allocations. Preserve custom function-provider support, immutable dynamic type and early identity installation. A larger record may offset bytes; measure both counts and bytes. No global receiver table. |
| 2 | Store ObjectInfo by value in the hierarchy root and initialize in place | With closure-free receiver storage, potentially graph 6→5. All views must return the same stable record address. Never copy after publication; preserve nil/uninitialized behavior and constructor publication order. |
| 3 | One allocation containing all superclass subobjects, with existing pointers wired to its fields | Potentially replaces three struct allocations with one. Highest risk: separate allocation from initialization while preserving super/this delegation, zero state, overridden calls during superclass construction, early this escape and exceptions. Retained interior/base pointers must keep the most-derived object alive. |

None is implemented or newly measured. Canonical identity must remain per object, not per class. Lazy identity introduces atomic-publication and constructor-callback hazards; eager in-place identity is preferable to investigate. Lazy views offer no gain where no wrappers exist. Do not pool/reuse identities while references, arrays, monitors or reflection can observe them.

## Validation after coordinator release

Reuse the existing hierarchy Java source/drivers and passing application sources unchanged; regenerate through an isolated generic compiler candidate. First test structural field omission separately from metadata/layout changes. Never count hand-edited generated Go as acceptance evidence.

Check method sets/defaults/generic bridges, constructor callbacks/delegation, zero-field object uniqueness, base/derived/erased aliases, reference arrays/casts, monitor ownership through aliases, dynamic getClass/reflection and GC with no roots or only a base/array view retained. Keep class-level descriptors free of receiver captures. Java reflection contracts must survive physical Go field changes. The frozen evidence predates recent monitor repairs, so reconcile publication invariants with the current accepted implementation first.

For measurement, reuse explicit Execution tokens and escaping sinks. Report construction bytes and allocation count separately from payload allocation. The existing 8-byte payload case establishes comparability; the large-payload retention modes check lifetime. Verify all Java observations and descriptor/monitor registry counts. No speed or faster-than-Java claim follows from this audit.
