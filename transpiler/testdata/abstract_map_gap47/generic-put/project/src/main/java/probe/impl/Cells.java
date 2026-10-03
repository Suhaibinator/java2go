package probe.impl;
import java.util.AbstractSet;
import java.util.ArrayList;
import java.util.Iterator;
import java.util.Map;
public final class Cells extends AbstractSet<Map.Entry<String,String>> {
    private final ArrayList<Map.Entry<String,String>> data = new ArrayList<>();
    public Iterator<Map.Entry<String,String>> iterator() { return data.iterator(); }
    public int size() { return data.size(); }
    public boolean add(Map.Entry<String,String> cell) { return data.add(cell); }
}
