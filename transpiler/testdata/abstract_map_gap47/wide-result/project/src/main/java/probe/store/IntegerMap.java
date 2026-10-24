package probe.store;
import java.util.AbstractMap;
import java.util.Map;
import java.util.Set;
import probe.entry.Slot;
public final class IntegerMap extends AbstractMap<String,Integer> {
    private final Cells cells = new Cells();
    @SuppressWarnings({"rawtypes", "unchecked"})
    public Set<Map.Entry<String,Integer>> entrySet() { return (Set) cells; }
    public void seed(String key, Object value) {
        Slot slot = new Slot(key, Integer.valueOf(0));
        slot.setValue(value);
        cells.add(slot);
    }
}
