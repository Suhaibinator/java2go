This application was written and run with JDK 21 before changing the runtime.
The original generated Go built but produced three entries for two equal keys,
failed fresh-key lookups and List searches, merged collectors by pointer,
considered distinct plain objects deeply equal, and printed TreeSet in insertion
order. Extending it to ConcurrentHashMap reproduced the same duplicate-key bug
before that implementation was changed.

The fixture exercises deliberately colliding hash codes, original-key retention,
updates/removal, custom equals through List/Set/Map, distinct and collectors,
natural-order TreeSet iteration, and ConcurrentHashMap. Its checked-in stdout
comes from Java; both the focused transpiler test and application parity suite
compile and compare the same source against the JVM when a JDK is available.

HashMap/HashSet retain deterministic encounter order. Collection views remain
snapshots (List.Slice remains an internal iteration alias). Tree containers use
binary search and linear insertion/removal; navigable/live-view APIs and
fail-fast iterators are outside this change. ConcurrentHashMap uses immutable
bucket snapshots and version-checked writes, so callbacks execute outside the
mutex and can perform reentrant reads; competing writers may retry equality.
