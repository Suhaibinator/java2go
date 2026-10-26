package localmap;
import java.util.Map;
import java.util.Set;
import java.util.HashMap;
public class LocalAbstractMapPollutionControl {
    static boolean inspect() {
        class Local extends java.util.AbstractMap<String,String> {
            final Map<String,String> storage = new HashMap<>();
            public Set<Map.Entry<String,String>> entrySet() { return storage.entrySet(); }
        }
        Local local = new Local();
        Map<String,String> view = local;
        Map rawStorage = local.storage;
        Integer pollution = Integer.valueOf(42);
        rawStorage.put("bad",pollution);
        Object throughRaw = ((Map)local).get("bad");
        boolean cast = false;
        try { String typed = view.get("bad"); } catch (ClassCastException expected) { cast = true; }
        Object stillPresent = ((Map)local).get("bad");
        Map rawTarget = new HashMap();
        rawTarget.putAll(local);
        return throughRaw == pollution && stillPresent == pollution && rawTarget.get("bad") == pollution && cast && local.size() == 1;
    }
    public static void main(String[] args) { System.out.println("raw-post-write=" + inspect()); }
}
