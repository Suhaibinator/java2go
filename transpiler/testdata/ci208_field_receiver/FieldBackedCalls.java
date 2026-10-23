
package parity.analytics.fields;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class FieldBackedCalls {
    private Parser parser;
    private Worker worker;
    private List<String> values;
    private Map<String, String> byKey;

    public FieldBackedCalls(Parser parser, Worker worker) {
        this.parser = parser;
        this.worker = worker;
        this.values = new ArrayList<String>();
        this.byKey = new HashMap<String, String>();
    }

    public String use() {
        this.values.add(this.parser.parse());
        this.byKey.put("worker", this.worker.name());
        return this.values.get(0) + this.byKey.get("worker") + this.values.size();
    }
}
