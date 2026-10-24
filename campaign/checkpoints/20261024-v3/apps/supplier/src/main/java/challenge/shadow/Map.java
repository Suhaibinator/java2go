package challenge.shadow;

// Intentionally unrelated to java.util.Map and java.util.Map.Entry.
public final class Map {
    public static final class Entry<K, V> {
        private final K key;
        private final V value;
        public Entry(K key, V value) { this.key = key; this.value = value; }
        public K getKey() { return key; }
        public V getValue() { return value; }
    }
}
