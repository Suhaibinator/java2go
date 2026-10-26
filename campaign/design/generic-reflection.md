# Generic reflection metadata: Phase 3 design

Status: design only. Phase 1/2 provide nominal `Type` protocols, `Class` compatibility,
virtual `getTypeName`, and dispatch to real source implementations of
`ParameterizedType`, `GenericArrayType`, `WildcardType`, `TypeVariable`, and
`GenericDeclaration`. They do **not** provide class/member generic signatures or
runtime-created type variables. Production remains held pending the coordinator's gate.

## Evidence from the frozen Gson 2.14.0 source

The source below is the unmodified dependency under
`.campaign/sources/gson-2.14.0/`; these are requirements, not candidates for replacing
Gson with runtime intrinsics.

| Caller | Required metadata and behavior |
| --- | --- |
| `reflect/TypeToken.java:96–110` | Anonymous subclass `getClass().getGenericSuperclass()` must return the actual parameterized superclass; its raw type is the canonical `TypeToken.class`. A raw subclass returns the raw `Class`, causing Gson's own error path. |
| `TypeToken.java:124–151` | Traverse variable declarations, generic arrays, owners, arguments, and wildcard bounds. Missing metadata cannot become null or empty collections. |
| `TypeToken.java:264–289,396–429` | Declared type parameters, ordered generic interfaces, generic superclass, and actual bounds used for argument-count/bound validation. |
| `internal/GsonTypes.java:105–120,171–218` | Canonicalization and structural equality consume all five Type shapes, including comparisons with Gson's own source implementations. |
| `GsonTypes.java:235–265,448–480` | Raw `getInterfaces()` and `getGenericInterfaces()` must have matching order. Resolve a variable through its declaring class and parameter position, without substituting unrelated same-named variables. |
| `internal/bind/ReflectiveTypeAdapterFactory.java:330,392,422` | All declared fields, each field's declaration-level `getGenericType`, and generic superclass traversal. Gson performs contextual substitution itself. |
| `internal/ConstructorConstructor.java:226,409` | Declared constructors and usable construction/access semantics; this is an additional reflection requirement beyond the signature graph. |
| `internal/reflect/ReflectionHelper.java:102–140,235–280` | Declaring classes, executable parameter types, and reflected record APIs; full source compilation requires these surfaces even when the current input does not traverse every branch. |

The frozen application actually uses `new TypeToken<List<Order>>() {}` in
`testfiles/campaign/round02/data/src/main/java/campaign/data2/export/JsonExporter.java:29`.
`Command.lines`/`Order.lines` have `List<LineItem>` signatures; metadata fields use
`Map<String,String>` and `TreeMap<String,String>`. The importer calls
`gson.fromJson(tree, Command.class)`. `SerializedName` values/alternate names are
also required by this application and are not supplied by a generic-Type graph.

## Current compiler/runtime boundary

`stdjava/reflection.go` registers `ClassDescriptor` with class identity, initialization,
public construction callbacks, public erased fields, selected no-argument methods,
and annotation type IDs. `FieldDescriptor` has only name, Go name, erased TypeID,
and finality. `MethodDescriptor` has only names and erased return TypeID.
`transpiler/runtime_metadata.go` currently omits generic-class member metadata,
non-public/static fields, parameterized methods, and most constructors. Its reflection
trigger must also recognize the added API surface. Extending only runtime accessors
would expose invented absence instead of source declarations.

Useful existing source facts are `ClassScope.Superclass` and
`ImplementedInterfaces` in source form; `OwnTypeParameters()` distinguishes declared
parameters from outer parameters carried for generated Go ABI purposes.
`TypeParam.Declaration` supplies compiler-local identity, and `Bounds`,
`Definition.OriginalType`, `DirectTypeParameter`, and `TypeParameterBindings` retain
lexical bindings. Reflection must use these source facts, never emitted Go names,
carried parameter counts, or an erased field's Go `reflect.Type`.

## Proposed graph and publication model

1. Emit an immutable signature graph, separate from physical Go storage. Its nodes
   represent a raw Class reference, a declaration-scoped variable reference, a
   parameterized raw Class plus optional owner and ordered arguments, a generic
   array component, or wildcard upper/lower bounds. Preserve full source signatures
   before erasure and parse nested member owners structurally rather than splitting
   strings at arbitrary dots or commas.
2. Give every generic declaration a stable Java identity. A class uses binary class
   identity; methods/constructors need the declaring class plus a stable JVM member
   descriptor and member kind. Compiler-local `*TypeParamDeclaration` keys map to
   serialized declaration/ordinal IDs; addresses and generated Go binder names must
   not escape into runtime identity. Variable name remains the Java source name.
3. Register declaration shells and all variable IDs before resolving bounds or
   supertypes. Resolve `T extends Comparable<T>`, intersections, and mutually
   referencing bounds through those shells. Publish complete immutable descriptors;
   lazy resolution must be cycle-aware and must not hold a non-reentrant lock across
   recursive resolution. Metadata lookup does not initialize Java classes or execute
   constructors. Add concurrent publication/race and initializer-observation tests.
4. Distinguish metadata absent/unsupported from a known empty declaration. A real
   non-generic class has an empty parameter array; a missing descriptor is not proof
   of non-genericity. Before publishing an accessor as supported, close all graph
   dependencies it can reach or report an explicit unsupported boundary.
5. Runtime node types implement the existing nominal Type protocols and execution
   dispatch. Keep canonical raw `Class` objects and declaration identity. Parameterized
   node interning is optional; Java equality must not depend on object identity.
   Type-variable equality/hash must terminate using declaration and name, without
   recursively hashing bounds. Repeated lookups may share immutable elements but
   return defensive Java reference arrays where the JDK does so.
6. Register real JDK generic signatures for runtime-provided classes reached by the
   graph, from the targeted JDK21 contract. `List`, `Collection`, `Map`, `TreeMap`,
   their ancestors, and used bounds cannot be declared non-generic merely because
   their Go containers are specialized. JDK21's `List<E>` directly extends
   `SequencedCollection<E>`; older assumed hierarchies would misalign Gson's traversal.
   This is JDK metadata, not a handwritten Commons/Gson implementation.

## Semantic contracts to lock down with JVM oracles

- `Class.getTypeParameters()` exposes only that declaration's parameters, in order.
  Shadowed class/method/constructor `T` variables are distinct. An implicit bound is
  genuinely `Object`; intersections keep declaration order. Variable `getName`,
  `getGenericDeclaration`, `getBounds`, equality, and hash remain consistent across
  repeated resolutions and references from fields/supertypes.
- `getGenericSuperclass()` describes the direct declaration, not a recursively
  substituted ancestor. Interfaces, Object, primitives, and void have null superclass;
  arrays have Object. Generic interfaces describe direct declared edges in the same
  order as raw interfaces; arrays expose Cloneable and Serializable. A raw superclass
  stays a Class even when its declaration is generic.
- `Field.getGenericType()` is the field declaration's Type. For `Base<T>.value`, a
  subclass lookup does not eagerly replace `T` with the subclass's argument. Gson's
  `resolve` supplies context. `getType()` remains erased Class identity. Declared-field
  lookup and enumeration need real visibility, modifiers, declaring class, and distinct
  hidden fields; metadata enumeration must not initialize the class.
- Extend methods/constructors with their own parameters/bounds, generic parameter
  types, return type (methods), exception types, and declaring class. Retain erased
  parameter/return/exception classes separately for lookup and invocation. Model
  overloads, bridges/synthetic members, and implicit constructors deliberately.
  Synthetic enclosing-instance constructor parameters and generic signature lengths
  require JVM probes before assuming positional equivalence.
- Parameterized equality/hash includes raw type, owner, and ordered arguments;
  generic arrays use their component; wildcards use both bound arrays. Compare the
  actual Java hash relationships rather than cross-process numeric Class hashes.
  Verify interoperability with source-implemented Type protocols, symmetric equality
  where specified, and unchanged source override/exception dispatch. TypeVariable
  implementation interoperability must be checked against JDK21 rather than inferred
  from the parameterized-node implementation.
- Public metadata arrays must not expose mutable backing slices. Test mutation of
  returned arguments, bounds, parameters, interfaces, and member arrays followed by
  another lookup. Preserve element identity/equality and correct runtime array type.
  Do not impose defensive cloning on user-defined protocol methods: Phase 2 correctly
  returns their actual arrays, including covariant `Class[]` and null.
- `getTypeName`/`toString` distinguish Class formatting from generic type formatting;
  recursive bounds do not expand infinitely. Null owner is meaningful for top-level
  parameterizations. Unsupported annotations/accessibility/record reflection are
  separate visible gaps, not successful empty results.

### Owner and raw/member cases checked against JDK21

A scratch JVM probe using `javac --release 21` observed the following graph shapes
(the temporary class prefix is omitted). These need permanent differential tests
before implementation, not hardcoded handling of these names.

| Field declaration | Reflected shape |
| --- | --- |
| `Outer<String>.Inner<Integer>` | Parameterized Inner; owner is parameterized Outer<String>; one own argument. |
| `Outer<String>.Leaf` where Leaf declares no parameters | Parameterized Leaf; owner is parameterized Outer<String>; **zero** own arguments. |
| `Outer.Static<Long>` | Parameterized Static; owner is raw Outer Class; one own argument. |
| raw `Outer.Inner` | Raw Class, not an empty parameterized node. |
| `List<?>` | Parameterized List; null owner; wildcard argument. |

This rules out treating every zero-argument member as raw, carrying outer arguments
into the inner argument array, or erasing every static member's owner to null.
Local/anonymous declaring-versus-enclosing relationships need separate tests; an
anonymous token must retain its declared parameterized superclass through hoisting.

## Incremental red-to-green sequence

1. **Signature graph and TypeToken capture:** reduced JVM source with anonymous
   `Capture<List<String>>`, raw Capture rejection, nested owners, and source/custom
   ParameterizedType comparisons. Add `Class.getGenericSuperclass` and parameterized
   runtime nodes using genuine emitted signatures. Validate null/array/primitive
   superclass rules. This slice is useful but cannot claim full Gson readiness.
2. **Declaration variables and interface traversal:** class/method/constructor
   shadowing, recursive/intersection bounds, defensive arrays, ordered generic
   interfaces, and inherited substitution inputs. Add actual JDK21 generic metadata
   needed by collection traversal. Test class metadata concurrent reads without
   class initialization and rebuild the earlier Type protocol tests.
3. **Declared fields and executable signatures:** field generic types first for the
   frozen List/Map model; then method/constructor generic declarations and bounds.
   Runtime access/construction, annotation values, access checks, and record APIs
   each get independent JVM reds. Do not hide missing pieces behind fake successful
   getters. Generic-array/wildcard nodes land when their first real signatures require
   them, with equality/hash and defensive-array tests in the same slice.
4. Re-transpile the entire unchanged Gson source closure and record each remaining
   compiler/runtime blocker. Accept only the frozen application output/resource/file
   comparison plus race/stress gate, and rebuild previously accepted apps because
   reflection and erased representations are shared. Source closure and expected
   results remain frozen throughout.

## Ownership and the compiler's canonical-erasure ABI

The runtime agent owns proposed new runtime signature/node files and reflection
intrinsic accessors, plus its existing nominal Type hooks. `stdjava/reflection.go`,
`transpiler/runtime_metadata.go`, and new metadata emission helpers need explicit
serialized ownership before edits. The compiler agent owns parsing/binder preservation,
anonymous/local class hoisting, canonical erased storage, generic method/override
bridges, and expression/callsite projection. Coordinate additions to symbol structures,
member descriptor identity, descriptor generation, and source registration with that
agent before touching them. Builder Reader/Writer files have no necessary overlap.

See [nested-generic-factory.md](nested-generic-factory.md) for the separate ABI plan.
A canonical erased instance must still report the same Java raw Class and generic
class declaration. `Box<String>` and `Box<Integer>` instances share raw Class identity;
reflection does not recover object instantiation arguments from Go type parameters.
Their field declaration metadata remains `T` even when the physical field is Object.
A TypeToken anonymous subclass carries its superclass signature because its source
class declaration does, not because an instance records a magic generic payload.

Both plans must agree on anonymous binary class IDs, declaration-variable identity,
member descriptors, and raw class ownership before generated ABI changes. Reflection
must not project an `Adapter<Number>` pointer into `Adapter<any>`, copy objects, or
repair generic factories by inventing signature arguments. Conversely, canonical
storage must not erase the source facts that reflection emits. This phase cannot
resolve Gson's generic factory ABI blocker by itself.
