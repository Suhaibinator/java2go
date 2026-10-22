These are the independently observed facts for the bounded JDK21 generic-parent graph. The eight generic declarations and three nongeneric terminal declarations are exactly the original verified inventory. Every native parameter reference records its actual declaring owner and index; recipes are not inferred from a shared spelling such as E.

`GenericParents.java` uses only public reflection on the explicitly named JDK classes, without initialization or application class loading. Three identical exports were independently checked against the pinned java.base classfile Signature attributes and src.zip declarations. `PROVENANCE.json` records all JDK/tool/source hashes. The original distribution license is retained in `JDK-DISTRIBUTION-LICENSE.txt`; no JDK implementation method bodies are copied here.

Regenerate or check the production Go data from these facts:

    python3 generate.py --check --gofmt /path/to/gofmt

Omit `--check` to write the generated file. This operation formats metadata only, with no JVM or application execution. Facts are consumed only for nominal parent substitution; they do not certify runtime method implementations or full application parity.
