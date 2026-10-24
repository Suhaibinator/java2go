package probe.entry;
import java.util.Map;
public final class Slot implements Map.Entry<String,Object> {
    private final String key;
    private Object value;
    public Slot(String key, Object value) { this.key = key; this.value = value; }
    public String getKey() { Trace.add("key"); return key; }
    public Object getValue() { Trace.add("value"); return value; }
    public Object setValue(Object next) { Trace.add("set"); Object old = value; value = next; return old; }
}
