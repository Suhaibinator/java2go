# Frozen round02 business challenge

JDK: `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home` (21.0.6). Maven dependencies are the campaign-pinned Apache Commons Lang 3.20.0 and Commons Codec 1.22.1 JARs. The manifest selects eleven unmodified upstream Java implementation files: three Lang mutable classes/interfaces and eight Codec classes/interfaces. All application and selected dependency sources compiled together with JDK-only classpath. Their output matched the published JAR run byte for byte for all three seeds.

Hashes use SHA-256 over each listed path, a NUL byte, its file bytes, and another NUL byte. Challenge files are all application Java files sorted by path, then `pom.xml`, then `fixture.json`. Dependency files are the selected Lang files followed by selected Codec files in manifest order. Oracle files are `expected.seed-17.stdout`, `expected.seed-41.stdout`, `expected.seed-97.stdout` in order.

| Group | Files | SHA-256 |
| --- | ---: | --- |
| challenge | 13 | `dc40238e9b885b330aa53d4028505c0d0f12f219792dee1741dabfdb8eeb964f` |
| dependency | 11 | `33d165baa3e7be14bc400fa523b4c2d9f600c82ca1d06436f892f689f5acfcfa` |
| oracle | 3 | `4f9a19759e0af4e925337dd7b3531f7463d208b5ba85e21709afe83a10a6f287` |

Each seed was run three times in a fresh JVM with identical stdout and empty stderr. Each run exited zero and ended in `balanced=true`.

| Seed | Each stdout SHA-256 |
| --- | --- |
| 17 | `83a03f019223fa42251ca52487b34b69bb45d449a37b3ca96541f42155c9e52e` |
| 41 | `66129f1b571f5279f923116de062d23618b131c4c6e7b108af6e23c6babb7274` |
| 97 | `7ea5abc97d78462079a4bc30846dacfb1b75787d1e4e4d90dfdc824a2c7d5e2c` |
