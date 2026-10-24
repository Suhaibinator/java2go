package probe.impl;
import java.util.Map;
public final class Cell implements Map.Entry<String,String> {
    private final String key;
    private String value;
    public Cell(String key, String value) { this.key = key; this.value = value; }
    public String getKey() { return key; }
    public String getValue() { return value; }
    public String setValue(String next) { String old = value; value = next; return old; }
}
