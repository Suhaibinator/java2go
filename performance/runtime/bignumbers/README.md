# Big-number runtime performance opportunities — unmeasured

This is a read-only analysis of the private runtime candidate captured on 2026-09-28. No candidate optimization was implemented, and no builds, tests, allocation probes, or benchmarks were run for this audit. There is **no measured speedup, allocation reduction, or advantage over the JVM**. The recommendations identify avoidable work visible in source and correctness gates for later isolated experiments.

At the time the parent requested preservation, four math test groups were reported green and the final Unicode/compiler-literal blocker was under repair. That status is supplied by the campaign, not validation performed by this audit. The private candidate can change; the hashes below identify the exact files inspected and must not be read as certification of its current contents. In particular, conversion performance work must wait for the blocked end-to-end oracle to compile and pass.

## Captured input and source evidence

Private candidate root: `/private/tmp/java2go-bignumber-tdd-xfbyyzs9`.

| Inspected file under that root | SHA-256 at capture |
|---|---|
| `stdjava/biginteger.go` | `200e8b766c72e2e0b8c02d38ad3947038dc2cac0a30ba919ec4f2449ab645cf0` |
| `stdjava/bigdecimal.go` | `27aeed236c7bf15ce6d70c3e1103aec7177b31ce510745677365b19c62ad93bf` |
| `stdjava/bignumber_conversion.go` | `e46e904d96c16108c6706d24f0dfce2e0a31f7af35b0715feb6d33ba3dea9bf2` |
| `transpiler/campaign_runtime_bignumber_jdk_test.go` | `be6b295258ac5294a68c691755c5f52b9b038ee04097a2733ccd8c5c6e1a4f92` |

Local primary-source evidence read during the audit:

- `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home/lib/src.zip`, entries `java.base/java/math/BigInteger.java` and `java.base/java/math/BigDecimal.java`. JDK 21 BigInteger float/double conversion works from magnitude bits, including explicit halfway/even rounding and overflow handling. BigDecimal uses compact-coefficient fast paths when scale is zero or an eligible power of ten permits a correctly rounded operation, then falls back. Its long conversion also short-circuits scale <= -64 because the low 64 bits vanish.
- `/opt/homebrew/opt/go/libexec/src/math/big/int.go`, methods `(*Int).Float64` and `(*Int).Bits`. Float64 returns the nearest floating-point value; its larger-value path uses `Float.SetInt(...).Float64()`. Bits exposes the existing little-endian magnitude words without copying and explicitly shares backing storage.
- `/opt/homebrew/opt/go/libexec/src/math/big/float.go`, methods `(*Float).SetInt` and `(*Float).Float32`. SetInt with zero receiver precision raises precision to at least the integer bit length, preserving the integer exactly. It copies magnitude words into the Float. Float32 performs target-width rounding from that value.

These local library paths identify the inspected implementation evidence; they are not benchmark outputs. A future run must record the actual Go/JDK versions and source hashes again. The JDK source demonstrates why removing the Go candidate's decimal formatting roundtrip is a plausible improvement over its own baseline, not proof that it will outperform Java.

## Five isolated candidates

### 1. Remove the BigInteger float/double decimal-string roundtrip

Captured `stdjava/biginteger.go:49–50` routes both conversions through `value.String()` and `strconv.ParseFloat`. That formats the complete magnitude into decimal only to parse it into binary floating point.

A first candidate can use `big.Int.Float64` for double conversion, and an exact zero-precision `big.Float.SetInt` followed by `Float32` for float conversion. A proven signed-integer fast path may avoid temporary magnitude copies for fitting values. The general SetInt path still copies magnitude storage; it must not be described as allocation-free without measurement.

Expected benefit, unmeasured: removal of decimal formatting/parsing work and related temporary storage. Semantic risk: incorrectly choosing an intermediate precision or narrowing through float64. In particular, `float32(integerAsFloat64)` can double-round and is not the general replacement.

Required gates: raw JDK float/double bits for zero, both signs, exact powers of two, values immediately below/at/above halfway points around 24- and 53-bit significands, carry into the next exponent, finite/overflow boundaries, and very large sparse/dense magnitudes. Retain the receiver null exception and verify conversions leave the original value, hash, text, and identity unchanged.

### 2. Hash magnitude words without copying a byte array

Captured `stdjava/bignumber_conversion.go:23` obtains `value.Bytes()` on every hash call. A local read-only walk of `value.Bits()` could process the same most-significant-first 32-bit chunks and avoid the copied big-endian byte buffer. On 64-bit Go words, process the high and low 32-bit portions in that order while walking words from most to least significant; handle 32-bit words separately.

Expected benefit, unmeasured: fewer temporary bytes/allocations on repeated BigInteger and BigDecimal hashing. Semantic risk: word-order mistakes, negative-sign handling, architecture dependence, and accidental mutation of shared magnitude storage.

Required gates: exact Java hashes for zero, both signs, boundaries around 2^32/2^64/2^96, leading and interior zero chunks, sparse/dense large magnitudes, and the BigDecimal combination with its signed scale. Preserve 32-bit overflow arithmetic. Validate both 32-bit and 64-bit word handling. Never mutate, expose, globally retain, or cache the Bits slice. Concurrent reads must remain race-free, and constructors must remain distinct where Java identity requires it.

### 3. Use low-bit arithmetic for negative-scale decimal narrowing

Captured `stdjava/bigdecimal.go:104` constructs a power of ten and a full big-integer product for negative scales from -1 through -63. Long conversion observes only the signed interpretation of the low 64 bits. Unsigned wrapping multiplication of the coefficient's low 64 bits by 10^(-scale), computed modulo 2^64, can produce the same result without constructing that full product. A bounded numeric power table or local modular exponentiation is sufficient; no object cache is required.

Expected benefit, unmeasured: avoid big.Int power/product temporaries and work proportional to coefficient magnitude in this path. Semantic risk: signed arithmetic, overflow, and taking the negative of an extreme int32 scale too early.

Required gates: positive/negative coefficients, coefficients much larger than 64 bits, truncation/wrap boundaries, scales -1/-31/-32/-63/-64 and both int32 extremes, zero, and byte/short/int/long results. Keep the existing scale <= -64 zero shortcut before exponent work. Positive-scale fractional truncation remains unchanged. Validate immutable storage before/after repeated conversions. Width-specific shortcuts for narrower results would need their own gates rather than silently changing LongValue behavior.

### 4. Compare equal-scale decimals directly

Captured `stdjava/bigdecimal.go:27` renders both coefficient magnitudes after checking signs, including when their scales are equal. After validating both receivers, equal scales permit direct coefficient comparison with `big.Int.Cmp`.

Expected benefit, unmeasured: remove two decimal renderings from equal-scale comparisons. This is the narrowest candidate in the set. Semantic risk is principally bypassing a required null check or accidentally normalizing representation.

Required gates: both null positions, positive/negative/zero coefficients, equal and different values, large magnitudes, and equal scales at both int32 extremes. Preserve different-scale numeric comparison, scale-sensitive equals/hash behavior, and constructor identity. Do not rescale or mutate either coefficient to obtain the shortcut.

### 5. Add narrowly proven decimal floating-point fast paths

Captured `stdjava/bigdecimal.go:134–135` always calls canonical String formatting and then ParseFloat. Scale zero can reuse correctly rounded integer conversion. For small nonzero scales, an exactly representable coefficient and an exactly representable target-width power of ten allow one multiplication/division at the target width. JDK 21's compact-coefficient paths provide a useful reference for the proof, not a license to copy its Java-specific float-to-long representability check into Go: out-of-range conversion behavior and boundary handling must be considered separately.

Expected benefit, unmeasured: bypass formatting/parsing on common compact values. Semantic risk is higher than the preceding candidates: double rounding, signed underflow zero, subnormal transitions, overflow, and extreme-scale resource use.

Required gates: exact raw float/double bits for halfway values and immediate neighbors, normal/subnormal boundaries, negative tiny inputs yielding negative zero, positive zero coefficients with varied scale, finite/overflow boundaries, and all extreme scales already exercised by the JDK-derived oracle. The captured tests include `1.000000059604644775390625` and its immediate decimal neighbor precisely to expose float double rounding. Preserve the existing parser fallback for unproven cases. General `coefficientAsFloat * Pow10(-scale)` is not a safe replacement. No unbounded power construction, scale-proportional loop, or identity-changing decimal normalization is acceptable.

## Deferred benchmark and validation plan

First repair the Unicode/compiler-literal blocker and establish exact current JVM/generated-Go output parity for all big-number groups. Then freeze and hash compiler, baseline runtime, isolated candidate, shared Java source, and generated bytes. Run each proposal separately so any result can be attributed to one change. Keep parsing/formatting/scale exceptions, null behavior, and identity assertions in the correctness workload; record failures rather than excluding them from claimed coverage.

Separate these workload classes:

1. Construction/parsing followed by one conversion, to represent one-shot JSON-number consumption.
2. Preconstructed values repeatedly converted, hashed, or compared, to isolate the proposed operation.
3. Mixed realistic Number-typed access with escaped outputs and source-level identity checks, to ensure the optimization survives generated dispatch and compiler behavior.

Use fixed seeds and identical Java inputs: small cached/noncached integers, 64-bit boundaries, 1,024-bit and 16,384-bit sparse/dense magnitudes, same-scale and different-scale decimal comparisons, negative-scale narrowing boundaries, compact decimal float cases, rounding ties/neighbors, and extreme scales. Record results as raw float/double bits or exact integral/hash checksums so constant folding, dead-code elimination, or a semantic mismatch cannot masquerade as improvement.

Collect allocations and allocated bytes separately from retained heap. A faster conversion that retains cached text or global objects is a different tradeoff; none is proposed here. Validate concurrent reads and immutable receiver state, including distinct constructor objects and the existing BigInteger.valueOf cache behavior.

Only after correctness and allocation evidence pass should the parent schedule uncontended timing: matching inputs, processor counts and comparable heap budgets, fresh separate processes, repeated trials with order variation, explicit JDK 21 path, and a warmed steady-state JVM phase reported separately from startup. Compare whole generated workloads as well as isolated operations. Until then, every expected benefit above remains a hypothesis and no JVM speedup is claimed.


The checkpoint15 equal-scale comparison optimization has [measured allocation evidence](equal_scale_verified/README.md), exact probe inputs and focused reproduction instructions. It preserves the C-locale failure and does not claim a speedup over Java. The older `equal_scale/` preparation is excluded from this milestone.
