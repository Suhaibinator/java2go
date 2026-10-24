# Generic adapter family probe

A named and an anonymous universal factory return the same stateful anonymous `Adapter<Number>` through `Adapter<Number>` and `Adapter<String>` static views. Four seeded writes alternate factory implementations and are read through the other factory. A selector's method type parameter shadows its class type parameter. A named `NumberAdapter` supplies a covariant `Integer read()` through the generic abstract base and `Slot<Number>` interface. Raw writes pollute both adapters; the write succeeds and the later typed read must throw `ClassCastException`. This is one focused reducer outside the frozen Gson acceptance fixture.

JDK: `/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home` (Java 21.0.6), compiled with `javac --release 21`. Each of seeds 17, 41, and 97 ran in three fresh JVM processes with exit zero, empty stderr, and identical stdout per seed. The saved outputs are `expected.seed-*.stdout`. The source SHA-256 is `6b730b66213255892714985469dfa31bfc4ef9def924685d4aa7a6663eadb71d` (eight sorted Java paths then POM); the oracle SHA-256 is `15cb700943a6da86d978abed7d12c1f836789c0e14c15acca9012e266e41d264` (seed order 17, 41, 97). Each hash feeds path, NUL, file bytes, NUL.

| Seed | Each JVM stdout SHA-256 |
| --- | --- |
| 17 | `fd4a5bdd6718b9f6dc434dad259c1bca09fed7b31701e8e8361c7cccf8dbe2dc` |
| 41 | `a43d9f0d750456f0f29b8a1709701c85599df75d9050b4444bbc619cf0ca1a89` |
| 97 | `8f59e236e8bbe77b29b6274ae307204b9518fceae0a92ddcdbd0ddfc58ad88ea` |

Initial source-only project conversion succeeded using `go run ./cmd/java2go -maven testfiles/campaign/probes/generic-family-business -main-class campaign.genericprobe.app.Main -runtime <repo> -module campaign.probe/genericfamily -output /tmp/java2go-generic-family-go`. The first Go build failed at `j_campaign/j_genericprobe/j_api/Factory.go:6:22: undefined: T`. The generated interface and factory callback contain a free method type variable in their `*Token[T]` and `*Adapter[T]` signatures. This is a focused mismatch; the full Gson application remains its separate acceptance gate.
