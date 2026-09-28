# Initial source translation discrepancy

The frozen JVM oracle and source-only JDK 21 compilation pass. Initial
source-only Maven conversion failed before Go generation with a nil-pointer
panic at `symbol/parsing.go:482`, parsing the upstream Apache Commons Codec
1.22.1 `DecoderException(final String message, final Object... args)`
constructor. `DigestUtils.java` also has modifier-bearing varargs and fails
at the same parser location.

The complete fixture was converted using a temporary Maven wrapper made from
the exact sources listed in `fixture.json` and a minimal POM for Codec:

```sh
go run ./cmd/java2go \
  -maven /tmp/campaign-data-project/app \
  -dependency-source commons-codec:commons-codec=/tmp/campaign-data-project/codec \
  -main-class campaign.data.app.Main \
  -runtime /Users/suhaib/.codex/worktrees/adversarial-java/java2go \
  -module campaign.test/data \
  -output /tmp/campaign-data-go
```

A supplementary 107-byte reproducer at `/tmp/CodecVarargsRepro.java`
contains `public class CodecVarargsRepro { public CodecVarargsRepro(final
String message, final Object... args) {} }`. It reproduces the same panic
with `java2go -q` and has SHA-256
`c6b356ee5d5c3753e0d63a99b7038502f3c8db8119a34f466273235cce768aa6`.
The full fixture remains the acceptance target.
