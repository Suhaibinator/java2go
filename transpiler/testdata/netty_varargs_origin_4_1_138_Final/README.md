# Netty varargs declaration-origin fixtures

These complete, unchanged Java files are from `io.netty:netty-buffer:4.1.138.Final`, source JAR SHA256 `57ec5e8742518e5fb9b3814316713eaf605132d4b3aff16a9d952ae208952013`, upstream tag `netty-4.1.138.Final`, commit `26e68a2c3935e459b374891c87760f6d6c5e2550`.

`PROVENANCE.json` pins both file hashes and original license-header hashes. The Netty Project's 2012 copyright and Apache License 2.0 headers are preserved byte-exact. `LICENSE-APACHE-2.0.txt` contains the standard Apache 2.0 license text. Source provenance: https://repo.maven.apache.org/maven2/io/netty/netty-buffer/4.1.138.Final/netty-buffer-4.1.138.Final-sources.jar

The regression parses these two genuine source files, resolves their symbols, and inspects the actual `CompositeByteBuf` spread-parameter forwarding call and array overload. It does not substitute reduced bodies, download files, skip if absent, or claim complete Netty compilation. Full campaign macro inputs remain separate.
