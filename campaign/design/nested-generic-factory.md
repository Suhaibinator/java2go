# Nested generic factory ABI: evidence and proposed representation

Evidence: the reduced source below is independently JDK21-valid; output is `same=true,missing=true,value=7` and `updated=42,missing2=true`. It preserves shared Adapter<Number> object identity and mutation across repeated `<T> Adapter<T> create(Token<T>)` calls, plus a null Adapter<String> result. Current generated interface/callback contains free T; the anonymous body uses *T. Frozen Gson supplies the same signature through TypeAdapterFactory.

Current boundaries:
- genericMethodHasErasedEntry intentionally accepts only bare method-variable positions, rejecting nested invariant parameterizations. Removing that guard would produce invalid or semantically wrong *Token[any]/*Adapter[any] casts.
- scopeForAnonymousMethod currently omits method TypeParameters; preserving those definitions is necessary for lexical binding but insufficient because Go closures cannot bind a universal type parameter.
- *Adapter[Number] and *Adapter[any] are different Go types. They must not be copied or asserted between each other. Java's unchecked Adapter<T> cast checks the erased Adapter class only.
- Existing class_type_parameter_erasure.go and override bridge planner handle bounded direct-owner shapes atomically, but deliberately exclude unrelated nested containers, interface/abstract families, and unmodeled anonymous subclasses. Reuse its representation-plan discipline rather than weakening predicates.

Proposed sound foundation:
1. Plan each affected raw generic class hierarchy as one erased instance layout. Store type-variable fields at JVM erasure (Object or leftmost bound), independent of source instantiation. A generic typed facade may retain static type information, but every facade points to the same canonical erased object/storage and reference identity. Prefer a shared non-generic underlying Go storage/alias when the complete family allows it; never convert between invariant generic pointers by unsafe assertion.
2. Expose the generic factory's JVM method descriptor using erased nominal reference handles: create(Token-erased)->Adapter-erased. Token and Adapter keep distinct nominal descriptors even if runtime payload interfaces are both any. Method-local type variables never appear in the emitted interface method set or closure.
3. Lower an anonymous factory body against its own declaration scope and those erased physical parameters/results. The implementation's <T> is a lexical Java binder, not an invented Go free type. Explicit `(Adapter<T>) shared` checks only Adapter's raw nominal descriptor and preserves the canonical object.
4. At typed consumers, project the same object into the planned static view. Reads/calls cast at the same consumption boundary javac does; writes store erased values and retain Java heap-pollution timing. In particular, a raw Adapter alias can store a String into an Adapter<Number>; the write succeeds and the typed read throws ClassCastException. An eager assertion to the old Go field type is wrong.
5. T[] remains the existing reified Java array object; erased storage cannot relax array-store checks. Class/type metadata keeps original Java binary names and raw class identity. Preserve nullable views, getClass, instanceof, identityHashCode, monitor ownership and source override bridges.
6. Apply the plan atomically to the whole connected class/interface/subclass family: fields, constructors, method descriptors, helper bodies, overrides, anonymous subclasses, method references, casts, raw accesses, and every callsite. Unsupported shapes remain explicit until all boundary operations have a coherent plan; no class-name exceptions.

Required gates before accepting a factory fix:
- minimal factory probe, repeated identity and writes through two references;
- named and anonymous factory implementations selected at runtime;
- raw heap-pollution write followed by failing typed read (correct timing);
- nested return/argument types and nulls, bounded variables, shadowed binders;
- inheritance/covariant overrides plus erased bridge cast timing;
- reified arrays, runtime Class/TypeToken metadata, Java object identity;
- existing bare-generic dispatch, raw-view, dependent-bound and generic SCC suites;
- full unchanged Gson fixture.

This is a distinct ABI project, not a one-line signature erasure. The immediate identifier bug is fixed independently so the full campaign can expose further concrete compiler/runtime prerequisites without falsely claiming generic factory support.

## Frozen reduced reproducer

This diagnoses the unchanged round02 Gson application and does not replace its acceptance gate.

```java
public final class GsonGenericFactoryRepro {
  static final class Token<T> {
    final String label;
    Token(String label) { this.label = label; }
  }
  static final class Adapter<T> {
    private T value;
    Adapter(T value) { this.value = value; }
    T read() { return value; }
    void write(T value) { this.value = value; }
  }
  interface Factory {
    <T> Adapter<T> create(Token<T> type);
  }
  static Factory newFactory() {
    Adapter<Number> shared = new Adapter<Number>(7);
    return new Factory() {
      @SuppressWarnings("unchecked")
      public <T> Adapter<T> create(Token<T> type) {
        return type.label.equals("number") ? (Adapter<T>) shared : null;
      }
    };
  }
  public static void main(String[] args) {
    Factory factory = newFactory();
    Token<Number> number = new Token<Number>("number");
    Token<String> string = new Token<String>("string");
    Adapter<Number> first = factory.create(number);
    Adapter<Number> again = factory.create(number);
    Adapter<String> missing = factory.create(string);
    System.out.println("same=" + (first == again) + ",missing=" + (missing == null) + ",value=" + first.read());
    first.write(42);
    System.out.println("updated=" + again.read() + ",missing2=" + (factory.create(string) == null));
  }
}
```
