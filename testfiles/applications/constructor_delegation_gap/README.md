# Constructor-delegation TDD fixture

This application isolates Java `this(...)` constructor delegation. The target
constructor must operate on the same object, instance field initialization must
run exactly once before its body, and the delegating constructor body must run
last.

This formerly failed when generated Go emitted an unresolved synthetic
constructor call for the delegation. The fixture now passes; its byte-exact
trace and initialization counter guard same-object delegation, a single
initializer run, argument effects, and constructor ordering against regression.
