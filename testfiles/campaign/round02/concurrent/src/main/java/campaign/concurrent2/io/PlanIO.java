package campaign.concurrent2.io;

import campaign.concurrent2.model.Job;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;

public final class PlanIO {
    private PlanIO() {}

    public static List<Job> read(int seed) throws Exception {
        byte[] bytes = Files.readAllBytes(Path.of("jobs.tsv"));
        String text = new String(bytes, StandardCharsets.UTF_8);
        List<Job> jobs = new ArrayList<>();
        for (String row : text.split("\\n")) {
            if (row.isEmpty()) { continue; }
            int separator = row.indexOf('|');
            if (separator <= 0 || separator == row.length() - 1) {
                throw new IllegalArgumentException("bad plan row: " + row);
            }
            String id = row.substring(0, separator);
            String body = row.substring(separator + 1) + "#" + (seed + jobs.size() * 13);
            jobs.add(new Job(id, body));
        }
        if (jobs.size() != 5) { throw new AssertionError("wrong plan size " + jobs.size()); }
        return jobs;
    }

    public static void write(String value) throws Exception {
        Files.write(Path.of("results.txt"), value.getBytes(StandardCharsets.UTF_8));
    }
}
