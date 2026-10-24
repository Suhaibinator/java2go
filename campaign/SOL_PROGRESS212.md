# Sol checkpoint 212: reproducible CI JDK data baseline

CI now requests Temurin release 21.0.6+7 through its published Adoptium SemVer, `21.0.6+7.0.LTS`. All six Java setup steps use this shared value. The prior floating `21` selected Temurin 21.0.12, whose currency data includes XAD; the complete-code test then failed in its fresh JVM oracle before transpilation because the repository deliberately locks the earlier 232-code data revision 177. Runtime tables, Java sources, expected observations, assertions and skips are unchanged.

The full build suffix matters: setup-java compares build identifiers when supplied. Official GA metadata and the actual action code, with its locked SemVer 7.7.1 implementation, accept the full published selector and reject the abbreviated `21.0.6+7`. The discarded private proposal is preserved as diagnostic evidence. No JDK archive was installed during local verification.

Coordinator and independent verification passed complete YAML parsing, all six shared Temurin consumers, exact selector and near-mismatch controls, the deterministic currency generator check, and forward/inverse patch checks. Installed Oracle 21.0.6+8 contains currency data matching the locked resource; that is not represented as a local Temurin run. The next normal hosted run must confirm actual Temurin adoption and the unchanged JVM/Go parity tests. No hosted GREEN or support for newer JDK currency datasets is claimed in this checkpoint.

The continuing Sol-only campaign retains the original application gates for ArrayList collectors, wildcard overloads, source Map views and RandomAccessFile, along with Supplier28, Gson112 and Netty1220. This CI configuration repair changes no compiler or runtime implementation.
