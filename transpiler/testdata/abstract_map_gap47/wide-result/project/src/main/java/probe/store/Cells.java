package probe.store;
import java.util.AbstractSet;
import java.util.ArrayList;
import java.util.Iterator;
import java.util.Map;
import probe.entry.Trace;
public final class Cells extends AbstractSet<Map.Entry<String,Object>> {
    private final ArrayList<Map.Entry<String,Object>> data = new ArrayList<>();
    public boolean add(Map.Entry<String,Object> entry) { return data.add(entry); }
    public int size() { return data.size(); }
    public Iterator<Map.Entry<String,Object>> iterator() {
        final Iterator<Map.Entry<String,Object>> cursor = data.iterator();
        return new Iterator<Map.Entry<String,Object>>() {
            public boolean hasNext() { Trace.add("has"); return cursor.hasNext(); }
            public Map.Entry<String,Object> next() { Trace.add("next"); return cursor.next(); }
            public void remove() { Trace.add("remove"); cursor.remove(); }
        };
    }
}
