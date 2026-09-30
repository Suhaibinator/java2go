package campaign.data2.app;

import campaign.data2.export.JsonExporter;
import campaign.data2.ingest.JsonImporter;
import campaign.data2.ingest.Loaded;
import campaign.data2.workflow.OrderWorkflow;
import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import java.nio.file.Path;

public final class Main {
    public static void main(String[] args) throws Exception {
        if (args.length < 1 || args.length > 2) throw new IllegalArgumentException("usage: Main SEED [OUTPUT_DIR]");
        long seed = Long.parseLong(args[0]);
        Path output = args.length == 2 ? Path.of(args[1]) : Path.of("out");
        Gson gson = new GsonBuilder().serializeNulls().disableHtmlEscaping().setPrettyPrinting().create();
        Loaded loaded = new JsonImporter(gson).load();
        OrderWorkflow workflow = new OrderWorkflow(loaded.issues);
        workflow.replay(loaded.commands, seed);
        String fingerprint = new JsonExporter(gson).write(output, seed, loaded, workflow);
        System.out.println("seed=" + seed + " parsed=" + loaded.commands.size()
                + " applied=" + workflow.applied() + " rejected=" + workflow.issues().size()
                + " orders=" + workflow.sortedOrders().size() + " exportHash=" + fingerprint);
    }
}
