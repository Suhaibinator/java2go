# Milestone 109: exhaustive compiler CI shards

The compiler regression package now runs in six parallel CI jobs. Each run enumerates the actual Go Test/Fuzz/Example inventory and hashes exact parent names into one shard. Anchored selectors run every descendant; future tests enter the partition automatically. Listing failures and empty shards fail. Race, coverage, JDK21, the existing twenty-minute package and twenty-five-minute job deadlines, and the aggregate required check are preserved.

Seven Python checks pass. Independent review reconciled all 1,358 baseline parents with six actual Go selectors; the coordinator separately verified all 1,359 current parents exactly once, with counts 226 / 240 / 183 / 239 / 230 / 241. These were listing checks, not regression executions. Compiler and runtime code are unchanged.

The prior unit timeout and all semantic failures remain recorded. Hosted shard timings and full CI success are unverified until the new workflow runs. Durable raw evidence and source hashes are under ignored .campaign/resume-20261002. Full campaign acceptance remains round18.
