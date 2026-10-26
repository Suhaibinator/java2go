package campaign.data2.export;

import campaign.data2.domain.Order;
import campaign.data2.ingest.Issue;
import campaign.data2.ingest.Loaded;
import campaign.data2.workflow.OrderWorkflow;
import com.google.gson.Gson;
import com.google.gson.JsonArray;
import com.google.gson.JsonObject;
import com.google.gson.reflect.TypeToken;
import java.io.IOException;
import java.lang.reflect.Type;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Comparator;
import java.util.List;
import java.util.Map;
import java.util.TreeMap;

public final class JsonExporter {
    private final Gson gson;

    public JsonExporter(Gson gson) { this.gson = gson; }

    public String write(Path directory, long seed, Loaded loaded, OrderWorkflow workflow) throws IOException {
        Files.createDirectories(directory);
        List<Order> orders = workflow.sortedOrders();
        Type orderListType = new TypeToken<List<Order>>() {}.getType();
        String ordersJson = gson.toJson(orders, orderListType) + "\n";
        Files.writeString(directory.resolve("orders.json"), ordersJson, StandardCharsets.UTF_8);

        TreeMap<String, Integer> totals = new TreeMap<>();
        for (Order order : orders) {
            totals.merge(order.customer, order.subtotalCents, Math::addExact);
        }
        JsonObject stats = new JsonObject();
        stats.addProperty("seed", seed);
        stats.addProperty("resources", loaded.resourceCount);
        stats.addProperty("lines", loaded.lineCount);
        stats.addProperty("parsed", loaded.commands.size());
        stats.addProperty("applied", workflow.applied());
        stats.addProperty("rejected", workflow.issues().size());
        JsonArray groups = new JsonArray();
        for (Map.Entry<String, Integer> entry : totals.entrySet()) {
            JsonObject group = new JsonObject();
            group.addProperty("customer", entry.getKey());
            group.addProperty("subtotal_cents", entry.getValue());
            groups.add(group);
        }
        stats.add("groups", groups);
        String statsJson = gson.toJson(stats) + "\n";
        Files.writeString(directory.resolve("stats.json"), statsJson, StandardCharsets.UTF_8);

        StringBuilder audit = new StringBuilder("location\tcode\tfingerprint\n");
        workflow.issues().stream()
                .sorted(Comparator.comparing((Issue issue) -> issue.location).thenComparing(issue -> issue.code))
                .forEach(issue -> audit.append(issue.row()).append('\n'));
        Files.writeString(directory.resolve("audit.tsv"), audit, StandardCharsets.UTF_8);
        return Integer.toHexString((ordersJson + statsJson + audit).hashCode());
    }
}
