package probe.base;
import java.util.AbstractMap;
public abstract class Base<K,V> extends AbstractMap<K,V> {
    public V inheritedPut(K key, V value) { return put(key, value); }
}
