package snapshot.app;

import java.io.BufferedReader;
import java.io.FileReader;
import java.io.FileWriter;
import snapshot.service.SnapshotReplay;

public final class Main {
    private static void write(String path, String text) throws Exception {
        try (FileWriter writer = new FileWriter(path)) { writer.write(text); }
    }
    public static void main(String[] args) throws Exception {
        SnapshotReplay replay = new SnapshotReplay(Integer.parseInt(args[2]));
        try (BufferedReader reader = new BufferedReader(new FileReader(args[0]))) {
            String line;
            while ((line = reader.readLine()) != null) {
                if (!line.isBlank() && !line.startsWith("#")) replay.apply(line);
            }
        }
        write(args[1] + "/first.json", replay.firstJson());
        write(args[1] + "/second.json", replay.secondJson());
        write(args[1] + "/copied.json", replay.copiedJson());
        write(args[1] + "/summary.json", replay.summary());
        write(args[1] + "/audit.tsv", replay.audit());
        System.out.println(replay.status());
    }
}
