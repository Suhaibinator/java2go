# Sol progress91: typed Boolean parsing

Boolean.parseBoolean now accepts canonical JavaString values through the existing UTF16 helper, including null and ASCII case-insensitive true. Canonical static imports resolve through declared String signatures and exact one-argument applicability. Source-defined Boolean classes retain their own behavior; no duplicate runtime helper was added.

Frozen native, canonical-call and static-import failures precede the general repair. All five fresh gates pass against progress90, including the unchanged analytics application JVM/Go comparison. The independent reviewer and coordinator verify exact six-file preimages, original Java observations, helper identity, raw command streams and cleanup. The original analytics files and expected output are unchanged.

This is scoped ingress evidence. Full accumulated CI, original Commons27, Netty and Gson remain unaccepted; checkpoint18 remains the latest accepted full round. The legacy e2e runner does not establish intermediate generated-file immutability guards. Continue the parallel Sol-only campaign.
