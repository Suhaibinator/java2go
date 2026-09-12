This second Java application established a JVM oracle before implementing
collection-valued keys and structural List/Set/Map equals/hashCode. Initially,
the generated Go failed to build because those methods had no lowering.

The fixture covers list-valued HashMap keys, order-sensitive List equality,
order-independent Set equality, null value versus absent map key, exact Java
collection hashes, comparison-based TreeSet equivalence, and sorted TreeMap
iteration. Both focused generated-code tests and the application parity corpus
consume this source and the Java-generated expected.stdout.
