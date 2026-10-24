# Sol checkpoint 231: Preserve complete raw-inner constructor arguments for canonical aliases

A raw qualifier whose physical representation is a canonical generic alias cannot infer enclosing Go type parameters from its erased hidden receiver. Keep the already-normalized declaration-aware class arguments for that representation; invariant owners retain inference from the same hidden object.

The original raw mutation regression and two new constructor/typed-alias interactions pass, alongside existing raw-bound, constructor, carried-bound, shadowing and member-reference guards. Eighteen parents and five subcases pass in each mode; six fresh JDK21 witnesses match the expected observations. Independent review: 7400a6d58bfc3455f3319efed0b315e3e1f84633a8a2cb1dbb882e889b7fc9a0.

The joint current230 integration passes all 22 parents and 12 subcases in each mode and all-package golangci-lint2.15. A combined nine-parent race shard hit its cumulative 55-second deadline; that failed run is retained. Every parent then passed in its own shard with the same source, binary, environment and 55-second limit. No timeout is credited as a pass. Published-head CI and full dependency application acceptance remain pending.
