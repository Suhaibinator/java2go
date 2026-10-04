package probe.impl;
import probe.base.Base;
import java.util.Iterator;
import java.util.Map;
import java.util.Set;
public final class StringMap extends Base<String,String> {
    private final Cells cells = new Cells();
    private int puts;
    public Set<Map.Entry<String,String>> entrySet() { return cells; }
    public int putCalls() { return puts; }
    @Override public String put(String key, String value) {
        puts++;
        Iterator<Map.Entry<String,String>> cursor = cells.iterator();
        while (cursor.hasNext()) {
            Map.Entry<String,String> cell = cursor.next();
            String found = cell.getKey();
            if (key == null ? found == null : key.equals(found)) return cell.setValue(value);
        }
        cells.add(new Cell(key, value));
        return null;
    }
    public String superPut(String key, String value) { return super.put(key, value); }
}
