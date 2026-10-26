package campaign.data.app;

import campaign.data.export.ReportWriter;
import campaign.data.ingest.EventLoader;
import campaign.data.ingest.Loaded;
import campaign.data.model.Ledger;
import java.nio.file.Path;

public final class Main {
    public static void main(String[] args) throws Exception {
        if (args.length < 1 || args.length > 2) throw new IllegalArgumentException("usage: Main SEED [OUTPUT_DIR]");
        long seed = Long.parseLong(args[0]);
        Path output = args.length == 2 ? Path.of(args[1]) : Path.of("out");
        Loaded loaded = new EventLoader().load();
        Ledger ledger = new Ledger(loaded.rejects);
        ledger.replay(loaded.events, seed);
        String digest = new ReportWriter().write(output, ledger, loaded.resourcesClosed, loaded.bytesRead);
        System.out.println("seed=" + seed + " accepted=" + loaded.events.size()
                + " applied=" + ledger.applied() + " rejected=" + ledger.rejects().size()
                + " closed=" + loaded.resourcesClosed + " bytes=" + loaded.bytesRead
                + " exportSha256=" + digest);
    }
}
