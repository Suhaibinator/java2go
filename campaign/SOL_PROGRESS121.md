# Milestone 121: monitor lifetimes and compiler contracts

The monitor registry now uses scalar managed allocation identities and weak monitor values. Active guards, waiters, entrants and legacy mutex handles keep monitors and their objects live. Released guards clear roots; cleanup checks an allocation epoch before deleting an entry. A receiver lifetime fence protects holdsLock lookup. Independent review found and corrected that fence before acceptance. Arbitrary comparable native host values retain their old compatibility path and its documented cycle limitation.

Declared functional superclass/interface edges now resolve in class header context. Explicit Object.super.finalize calls execute the JDK empty body while retaining source overrides and varargs selection. This introduces no automatic finalizer scheduling.

The coordinator rebuilt matching race binaries and independently passed the complete runtime gate:590 parents,zero skips. All25 selected compiler parents passed, including the six original functional-header JVM observations and explicit-finalize/concurrent controls. The monitor JDK21 oracle was rebuilt and repeated three times with exact output. Both original concurrent dependency applications pass nine seed/repeat comparisons and20 additional stress observations each, with generated package and entry race builds. All443 implementation file hashes match the current candidate, unchanged during both campaigns. The scratch runner inherited primary checkout Git revision c1f1ae4; explicit campaign parent and authoritative source hashes are recorded separately.

A scratch infrastructure agent proposed a default parallelism flag while verification was active. It changed no production sources or explicit verification environment. The coordinator preserved the proposed runner separately and restored the original runner; the version change and affected receipts are recorded.

No fullCI, fullWeakReference, GC scheduling, wholeNetty or fullround acceptance is claimed. Latest accepted campaign round remains18. Independent source audits and bounded execution receipts are retained in ignored .campaign/resume-20261002.
