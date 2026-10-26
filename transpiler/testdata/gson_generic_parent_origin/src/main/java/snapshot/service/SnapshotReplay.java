package snapshot.service;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonArray;
import com.google.gson.JsonObject;
import snapshot.codec.EntryAdapter;
import snapshot.model.Entry;

public final class SnapshotReplay {
    private final int seed;
    private final Gson baseline;
    private final Gson first;
    private final Gson second;
    private final Gson copiedFirst;
    private final JsonArray firstRows = new JsonArray();
    private final JsonArray secondRows = new JsonArray();
    private final JsonArray copiedRows = new JsonArray();
    private final StringBuilder audit = new StringBuilder("row\torder-id\tdelta\tbalance\tfirst\tsecond\tcopied\tvalues-equal\n");
    private int count;
    private int balance;
    public SnapshotReplay(int seed) {
        this.seed = seed;
        balance = seed;
        GsonBuilder builder = new GsonBuilder().serializeNulls().disableHtmlEscaping().setPrettyPrinting();
        baseline = builder.create();
        builder.registerTypeAdapter(Entry.class, new EntryAdapter("first-" + seed));
        first = builder.create();
        builder.registerTypeAdapter(Entry.class, new EntryAdapter("second-" + seed));
        second = builder.create();
        copiedFirst = first.newBuilder().create();
    }
    public void apply(String line) {
        JsonObject tree = baseline.fromJson(line, JsonObject.class);
        Entry a = first.fromJson(tree, Entry.class);
        Entry b = second.fromJson(tree, Entry.class);
        Entry c = copiedFirst.fromJson(tree, Entry.class);
        balance += a.delta;
        count++;
        boolean same = a.orderId.equals(b.orderId) && a.orderId.equals(c.orderId) && a.delta == b.delta && a.delta == c.delta;
        audit.append(count).append('\t').append(a.orderId).append('\t').append(a.delta).append('\t').append(balance);
        audit.append('\t').append(a.parsedBy).append('\t').append(b.parsedBy).append('\t').append(c.parsedBy).append('\t').append(same).append('\n');
        firstRows.add(first.toJsonTree(a));
        secondRows.add(second.toJsonTree(b));
        copiedRows.add(copiedFirst.toJsonTree(c));
    }
    public String firstJson() { return baseline.toJson(firstRows); }
    public String secondJson() { return baseline.toJson(secondRows); }
    public String copiedJson() { return baseline.toJson(copiedRows); }
    public String audit() { return audit.toString(); }
    public String summary() {
        JsonObject result = new JsonObject();
        result.addProperty("seed", seed);
        result.addProperty("processed", count);
        result.addProperty("balance", balance);
        return baseline.toJson(result);
    }
    public String status() { return "seed=" + seed + " processed=" + count + " balance=" + balance; }
}
