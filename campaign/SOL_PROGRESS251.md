# Sol milestone 251 — Math.floorMod

Support Java 21 Math.floorMod(int,int), (long,int) and (long,long) through the runtime and native-bound compiler calls, static imports and method references. Preserve evaluation order, signed-width adaptation, zero-divisor behavior and source/value/binder shadows.

Validation: all 12 added test parents have normal and race RUN/PASS evidence (24 source-bound links). The component retains 60 passing functional stages, followed by a test-only lint correction and seven fresh checks. Six fresh default/F1 build, test-compilation and lint profiles pass; both lint results report zero issues. Independent reports: parent gate 6302c2b7, V2 component 5d532f7e, six-profile actual 97cbb299.

The campaign remains active. Full compiler native execution, F1 native tests and complete application acceptance remain separate gates. Milestone 249 was fully reverted by 250 after CI exposed an omitted whole-Gson-helper test and an allocation-budget failure; the feature and unchanged tests remain in the private repair checkpoint. This milestone contains the six Math component paths and campaign documentation.
