This fixture contains the exact original 114 Java sources used by the Gson generic-parent declaration regression: 4 campaign application sources, 81 Gson 2.14.0 sources and 29 Error Prone annotation 2.48.0 sources. SOURCE-MANIFEST.json pins every original byte. Upstream copyright and Apache-2.0 headers are unchanged; licenses/ retains the license text and published POM licensing metadata. PROVENANCE.json records archive and author-observation hashes.

The Go test parses all files, registers all symbols, resolves the complete batch once, and inspects the original Gson/GsonBuilder calls. It runs no JVM and uses no binary dependency jars. The fixture is selected by default. An explicitly supplied JAVA2GO_GSON_ORIGIN_INPUT may select another root only when every one of the same 114 source bytes matches this manifest.

This declaration regression does not establish full Gson application parity. The original application, resources, POM and complete observations remain required acceptance.
