# Exercised reflection access and annotation contracts

This milestone adds platform implementations for the exercised AccessibleObject, reflection member, class-metadata and annotation operations. Source-generated metadata and declared access context drive lookup and invocation; dependency implementation code is still translated.

The tested contracts include caller-sensitive field and constructor access, access overrides, protected constructor restrictions, typed long field reads/writes, inherited and declared metadata, annotation retention/target/inheritance rules, fresh returned arrays, and source declaration ownership. Unsupported metadata and annotation-value forms keep explicit boundaries. These changes do not establish complete reflection or annotation-proxy behavior.

Compiler changes preserve source initialization, record identities and accessors, synthetic inherited dispatch and anonymous lexical ownership. The current integration also preserves the published generic source-class storage fix: receiver canonicalization filters the exact synthetic receiver and does not rewrite initializer functions or unrelated binders.

Fresh verification on the checkpoint 244 base passed all 23 stages: two compiler builds, two compiled inventories, nine native test groups per mode and lint. Each mode passed 80 compiler parents and 412 total test nodes with no failures or skips. Both binaries registered 2,749 tests, retaining all 2,719 published names and adding 30 reflection/dispatch controls; this is inventory evidence, not a full 2,749-test run. Whole JVM/Go helper observations, original caps and owned process closures passed independent actual review and coordinator receipt checks. The separately qualified checkpoint 243 runtime evidence retains that source scope; all 725 runtime paths are byte-identical in the current integration. The two initial dispatcher inventory stops remain recorded, with successful payloads adopted without semantic reruns.

The real six-source Gson prerequisite and focused JVM/Go observations are acceptance evidence for the exercised cases. The full 112-source Gson application, full Netty translation and application parity remain pending.
