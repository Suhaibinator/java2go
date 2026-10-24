# Gson order import and mutation workflow

This fixture imports two UTF-8 JSON Lines resources through real Gson 2.14.0.
It parses each line as a JSON tree, then binds valid objects reflectively to
`Command` using `@SerializedName` aliases, nested line items, and metadata maps.
The seeded shuffle replays creates, quantity increments, renames, tags, and
cancellations against mutable orders. Checked arithmetic validates totals
before committing a quantity change. Malformed JSON, missing fields, invalid
lines, overflow, duplicate orders, unknown SKUs, and missing targets become
deterministic audit rows.

The exporter serializes sorted orders with Gson's generic `TypeToken` and
reflective field adapters. It builds a separate JSON tree for grouped totals
and writes an audit TSV. Inputs cover decomposed and composed Unicode,
Japanese text, emoji, null metadata, HTML-sensitive text, nested collections,
and canonical plus alternate field names. The Java program computes every
output from resources and the seed; no expected output is embedded in it.

`fixture.json` lists all 81 implementation-class files from the locked Gson
source JAR and all 29 implementation-class files from locked ErrorProne
annotations. Only module/package descriptors are excluded from the classpath
source closure; the selected Java classes and their method bodies are intact.
The entire closure compiles under JDK 21 without binary dependencies. The
offline Maven POM pins Gson 2.14.0, ErrorProne annotations 2.48.0, compiler
plugin 3.16.0, and resources plugin 3.5.0.

Seeds 17, 41, and 97 each ran three times in fresh JVMs. Binary Gson and the
complete selected source closure produced identical exit status, stdout,
stderr, and output-file hashes on every run. Frozen hashes (SHA-256):

- `inputs.sha256`: `63d3ddd57b362079cdc20b001eae9ac6cfc1938e56b2f14f09293c43a0bb2ffa`
- `oracle.json`: `0ca4e286707c2a46f20728f4c39f39e768cc031eb42f569477b177daa47b0780`
- `source_oracle.json`: `c41cf6b236654e61d5fb3c7b8fa811ec0c1948bb434d67d44c5ba113f69e6f46`

Do not alter the frozen application, selected upstream sources, resources, or
expected observations to make translation pass. A reduced reproducer may
supplement diagnosis but cannot replace this acceptance fixture.
