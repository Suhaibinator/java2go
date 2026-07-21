# Constructor-delegation TDD fixture

This application isolates Java `this(...)` constructor delegation. The target
constructor must operate on the same object, instance field initialization must
run exactly once before its body, and the delegating constructor body must run
last.

Generated Go currently emits an unresolved synthetic constructor call for the
delegation. Once that call is resolved, the byte-exact trace and initialization
counter will guard object identity, initialization count, and ordering.
