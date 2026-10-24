package transpiler

import "testing"

func TestCampaignMapEntrySourceArgumentApplicabilityJVM51(t *testing.T) {
 runCampaignCompilerStrictProjectOracle(t, map[string]string{
"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>entry-applicability</artifactId><version>1</version></project>
`,
"src/main/java/probe/app/Main.java": `package probe.app;
import java.util.Map;
import probe.impl.Cell;
import probe.entry.Slot;
import probe.entry.Trace;
public final class Main {
    public static void main(String[] args) {
        probe.impl.Cells strings = new probe.impl.Cells();
        Cell cell = new Cell("key", "old");
        System.out.println("add:" + strings.add(cell) + ":" + strings.size());
        Map.Entry<String,String> alias = strings.iterator().next();
        System.out.println("alias:" + (alias == cell) + ":" + alias.setValue("next") + ":" + cell.getValue());
        probe.store.Cells objects = new probe.store.Cells();
        Slot slot = new Slot("wide", Integer.valueOf(7));
        System.out.println("wide-add:" + objects.add(slot) + ":" + objects.size());
        Map.Entry<String,Object> wide = objects.iterator().next();
        Object old = wide.setValue("done");
        System.out.println("wide-alias:" + (wide == slot) + ":" + old + ":" + slot.getValue());
        System.out.println("trace:" + Trace.take());
    }
}
`,
"src/main/java/probe/entry/Slot.java": `package probe.entry;
import java.util.Map;
public final class Slot implements Map.Entry<String,Object> {
    private final String key;
    private Object value;
    public Slot(String key, Object value) { this.key = key; this.value = value; }
    public String getKey() { Trace.add("key"); return key; }
    public Object getValue() { Trace.add("value"); return value; }
    public Object setValue(Object next) { Trace.add("set"); Object old = value; value = next; return old; }
}
`,
"src/main/java/probe/entry/Trace.java": `package probe.entry;
public final class Trace {
    private static String events = "";
    public static void add(String event) { events += event + ","; }
    public static String take() { String found = events; events = ""; return found; }
}
`,
"src/main/java/probe/impl/Cell.java": `package probe.impl;
import java.util.Map;
public final class Cell implements Map.Entry<String,String> {
    private final String key;
    private String value;
    public Cell(String key, String value) { this.key = key; this.value = value; }
    public String getKey() { return key; }
    public String getValue() { return value; }
    public String setValue(String next) { String old = value; value = next; return old; }
}
`,
"src/main/java/probe/impl/Cells.java": `package probe.impl;
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
`,
"src/main/java/probe/store/Cells.java": `package probe.store;
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
`,
}, "probe.app.Main", "add:true:1\nalias:true:old:next\nwide-add:true:1\nwide-alias:true:7:done\ntrace:next,set,value,\n")
}
