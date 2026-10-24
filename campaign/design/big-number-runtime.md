# BigInteger and BigDecimal runtime prerequisites

Status: proposed design, 2026-09-28. This document records a read-only source
inventory. It does not claim implementation, passing tests, or Gson acceptance.
No runtime/compiler changes or builds/tests were performed for this task. The
unchanged complete frozen Gson application remains the acceptance gate.

## Source provenance

The inventory scanned every Java member in the locked Gson 2.14.0 sources archive
`.campaign/cache/gson-2.14.0-sources.jar`, SHA-256
`a4873f0ef88981cab520c3d7449cd89a68e605f15e9fe46e1aa2c77d5f87eb6f`.
All Gson references below identify archive members under `com/google/gson/`.
Dependency source, application fixtures, and JVM oracles must remain unchanged.

Authoritative JDK contracts were inspected in the installed JDK21 archive:
`/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home/lib/src.zip`.
Member hashes make the reviewed source version explicit:

| Archive member | SHA-256 |
| --- | --- |
| `java.base/java/math/BigInteger.java` | `19467dff4950db2cdc5e73782e2d3f868972a92e33eeb00ae70d150021c45d76` |
| `java.base/java/math/BigDecimal.java` | `bc786581cce27c1ad0b4b919f87c7d31242c148c9edf02ae83e46e0f61eec9ce` |
| `java.base/java/math/RoundingMode.java` | `8d0718e1bd3c9c929bd036090f77eeaad730bb3f314c2a6cff91a1b7d74d7149` |
| `java.base/java/lang/Number.java` | `d1bb35c47c9a7f2fca6a03678350f8718b316ee7b66d7387231ea997ef13da12` |

## APIs required by the complete Gson source

| API or boundary | Gson member and lines |
| --- | --- |
| `BigInteger(String)`, `BigDecimal(String)`, `BigDecimal.scale()` | `internal/NumberLimits.java:21–34` |
| `BigInteger.valueOf(long)`, type checks/casts, `BigInteger.equals(Object)`, `BigDecimal.compareTo(BigDecimal)` | `JsonPrimitive.java:182–197, 294–301` |
| `intValue()` and `longValue()` on parsed BigDecimal | `internal/LazilyParsedNumber.java:40–62` |
| Numeric conversions through Number, including long and double | `JsonPrimitive.java:261–272, 304–305` |
| Dynamic `Number.toString()`, actual class identity, Number assignability | `stream/JsonWriter.java:633–657, 730–740` |
| Nullable references, generic adapter parameters, class literals | `internal/bind/TypeAdapters.java:575–625` |
| Big-number reference results and forwarding | `JsonArray.java:292–307`, `JsonElement.java:352–367` |
| BigDecimal parse result through number-reading policy | `ToNumberPolicy.java:103–106` |

The ordinary Object/Number contract also requires correct equality, hashing,
comparison, and all six Number accessors when these objects are consumed through
existing generic runtime protocols. `Number.java:55–119` defines Number's
Serializable ancestry, four abstract numeric conversions, and byte/short
conversions derived from int conversion.

The complete locked Gson source does not directly invoke big-number arithmetic,
`setScale`, `RoundingMode`, `MathContext`, exact integer conversions,
`toPlainString`, or `stripTrailingZeros`. These are future APIs, not justification
for broadening the first implementation. Gson's own input length and scale
limits in NumberLimits must continue executing from its source; do not move those
library policies into the JDK numeric implementation.

## Representation and identity

Use genuine JDK service implementations backed by Go `math/big`, not replacements
for Gson parsers or adapters:

- BigInteger is a distinct immutable reference object with a privately owned
  signed `big.Int` value.
- BigDecimal is a distinct immutable reference object containing a privately
  owned signed integer coefficient and an `int32` scale. Its value is coefficient
  multiplied by ten to the negative scale. A rational or float alone loses scale,
  equality, hash, and canonical formatting information.
- Both implement the existing `JavaNumber` protocol and expose canonical
  `java.math` dynamic identities. Preserve the same allocation through Number,
  Object, arrays, generic fields, and callbacks. Do not turn values into boxed
  primitive substitutes or copy objects merely to change the static view.
- Go big-number methods mutate receivers. Copy mutable operands when constructing
  independently owned results; never expose mutable internals. Avoid lazy caches
  that introduce races and avoid global registries retaining instances.
- JDK21 `BigInteger.valueOf` reuses a bounded cache, including values from -16 to
  16 (`BigInteger.java:1194–1203, 1242`). If modeled for JDK21 identity parity,
  keep it finite and separate from arbitrary constructed objects.
- Neither JDK class is final (`BigInteger.java:137`, `BigDecimal.java:309`).
  Source-defined subclasses and virtual overrides require explicit future
  support; a concrete first slice must not falsely claim their semantics.

## Contracts and pitfalls

**Parsing and scale.** BigInteger parses an optional leading sign and decimal
UTF-16 character digits; embedded signs, missing digits, and invalid characters
have observable failure precedence (`BigInteger.java:484–541`). BigDecimal's
string constructor preserves scale, including trailing fractional zeros and
negative scale, accepts its documented signed decimal/exponent grammar, and
rejects malformed inputs with NumberFormatException. Its implementation checks
exponent/scale overflow (`BigDecimal.java:521–758, 804–855`). Unicode character
handling must follow Java char-based parsing rather than inadvertently accepting
all supplementary digit code points. Nullable strings require Java null behavior.

**Equality, comparison, hash.** BigInteger equality is numeric and type-specific.
BigDecimal `equals` requires matching coefficient value and scale, while
`compareTo` compares numerical value and ignores representational scale
(`BigDecimal.java:3128–3143, 3226–3242`). Preserve this distinction through hash
maps and sorted maps. JDK hashes use signed 32-bit overflow: BigInteger hashes
unsigned 32-bit magnitude words then applies the sign; BigDecimal combines its
unscaled-value hash and scale (`BigInteger.java:4065–4071`,
`BigDecimal.java:3291–3298`). Go hashes or normalized decimal hashes are incorrect.

**Text.** BigDecimal canonical formatting chooses plain notation when scale is
nonnegative and adjusted exponent is at least -6, otherwise scientific notation.
It preserves distinguishable scale forms, uses uppercase E and a positive
exponent sign, and removes a negative sign from numerical zero while retaining
its scale (`BigDecimal.java:3303–3372`). JsonWriter directly consumes this result.
Formatting must not first convert through a floating-point value.

**Conversions.** Integer conversions truncate fractional BigDecimal values toward
zero, then retain low two's-complement bits; they do not saturate. BigInteger has
the corresponding low-bit conversion (`BigInteger.java:4336–4363`,
`BigDecimal.java:3587–3603, 3692–3695`). Handle extreme scales without constructing
unnecessary enormous powers of ten. Float/double conversions must match JDK
rounding, infinities, subnormal boundaries, and signed underflow. Producing float32
through float64 can double-round. Use bit-exact JVM oracles, not decimal tolerance.

**Exceptions and evaluation.** Preserve NumberFormatException versus
ArithmeticException versus NullPointerException, evaluation order, and the
message/cause contracts exposed by the implemented overloads. Do not import
Gson's 10,000-character/scale limits into general JDK parsing. Resource exhaustion
is not permission to return fabricated values.

**Future rounding.** If later required, implement rounding as quotient/remainder
operations over the exact coefficient. Test UP, DOWN, CEILING, FLOOR, HALF_UP,
HALF_DOWN, HALF_EVEN, and UNNECESSARY with signs and ties. `setScale` returns a
value and does not mutate the receiver; UNNECESSARY rejects discarded nonzero
digits (`BigDecimal.java:2846–2876`; `RoundingMode.java:153–383`). MathContext
precision and preferred-scale arithmetic are separate future contracts.

## Proposed lowering and ownership

Runtime owner would add `stdjava/biginteger.go`, `stdjava/bigdecimal.go`, and small
shared parsing/conversion helpers where justified. The existing Number protocol
already dispatches to objects implementing its six numeric accessors; preserve
that path rather than add big-number-specific Gson behavior.

A new `transpiler/intrinsics_bigmath.go` would register exact JDK owners,
constructor/static overloads, numeric methods, scale, comparison, equality, hash,
and string conversion. Coordinate narrow hooks for runtime type mapping,
canonical owner resolution, class descriptors, and nominal
Number/Comparable/Serializable assignability. Source declarations and binders must
continue taking precedence over same-named JDK classes. Expected argument/result
types must preserve nulls and numeric widening at calls and method references.

New uniquely named runtime tests and
`transpiler/campaign_runtime_bignumber_*_test.go` would provide differential
coverage. No shared compiler file ownership or implementation is assigned by this
document; agree those hooks with the compiler owner before edits. The shared
collection-storage design remains an independent prerequisite and is still held.

## Red-to-green sequence

1. Establish JVM oracles for nominal identity, nullable references, Number
   dispatch, constructors, parsing failures, scale, and canonical text.
2. Implement the exact representation and required parsing/formatting surface;
   test large coefficients, Unicode digits, signed zeros, exponent boundaries,
   scale overflow, and immutability/aliasing.
3. Verify equality/hash against comparison with scaled-equivalent decimals,
   negative values, multiword magnitudes, null/wrong types, and map/set use.
4. Verify truncating/wrapping integer conversions and bit-exact floating results,
   including halfway cases, both signs, overflow, subnormal values, and extreme
   scales. Include Object/Number/generic method-reference boundaries.
5. Compile and compare unchanged upstream numeric source paths, then regenerate
   historical applications and run the unchanged full frozen Gson challenge.

A focused green JDK slice is prerequisite evidence only. It does not establish
complete BigInteger/BigDecimal support or acceptance of the full Gson challenge.
