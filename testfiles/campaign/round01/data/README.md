# Stateful Unicode ledger with Apache Commons Codec

This fixture reads two UTF-8 classpath resources, checks a truncated SHA-256
signature on every event, decodes form URL fields, normalizes labels to NFC,
and replays the valid events in a Java `Random(seed)` shuffle. The replay is
stateful: `PUT` replaces an account, `ADD` requires an existing account with
the same normalized label and uses checked addition, and `DEL` removes an
existing account. Seed order changes the final balances and rejection reasons.

The resources include decomposed Unicode, Japanese text, an emoji, spaces,
plus signs, slashes, percent signs, large integers, and malformed URL, hex,
signature, number, and column cases. Both resource streams are tracked through
try-with-resources. The exported summary exposes byte and close counts, and
`out/rejects.tsv` exposes parse and replay failures. The other declared
outputs contain accounts and grouped totals, using Commons Codec URL, hex,
and digest implementations. There are no canned outputs in the Java program.

The pinned dependency is Apache Commons Codec 1.22.1. The manifest lists the
source closure required for the fixture; the campaign must translate those
real dependency sources alongside the application sources. Java 21 is required
at `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home`.

The oracle runs seeds 17, 41, and 97 three times each in fresh process and
working directories. The expected stdout and file hashes are frozen after
successful JVM runs. Change neither input nor oracle to hide a translation
discrepancy; use a separate reduced reproducer for diagnosis.

Frozen verification hashes (SHA-256):

- `inputs.sha256`: `63d50dbbf9dc4f9743fdfb42ffa13e76279b007c419417ae6325c1a83b93ea38`
- `oracle.json`: `c2b6f0547737c40bf991da03391c1bb0b5a22e71d65fd58f6bcfe8a16067924c`
- pinned Codec binary: `78a5d732fbd715e2d10bd7150d2f8030bae57267f8aacc5c88f642cb6c2e5d3f`
- pinned Codec source JAR: `96444ee2fa5a41dc4d3b7b95f46b67495795d458e2050fc08bb88ab7f87ac021`
