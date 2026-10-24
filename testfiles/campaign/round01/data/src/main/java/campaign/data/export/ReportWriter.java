package campaign.data.export;

import campaign.data.model.Account;
import campaign.data.model.Ledger;
import campaign.data.model.Reject;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Comparator;
import java.util.List;
import java.util.Map;
import java.util.TreeMap;
import java.util.stream.Collectors;
import org.apache.commons.codec.binary.Hex;
import org.apache.commons.codec.digest.DigestUtils;
import org.apache.commons.codec.net.URLCodec;

public final class ReportWriter {
    private static final URLCodec URL = new URLCodec();

    public String write(Path directory, Ledger ledger, int closed, long bytesRead) throws IOException {
        Files.createDirectories(directory);
        List<Account> accounts = ledger.accounts().values().stream()
                .sorted(Comparator.comparing(account -> account.id)).collect(Collectors.toList());
        StringBuilder rows = new StringBuilder("id\tlabel-url\tlabel-hex\tbalance\trevision\n");
        for (Account account : accounts) {
            String encoded = URL.encode(account.label, "UTF-8");
            String hex = Hex.encodeHexString(account.label.getBytes(StandardCharsets.UTF_8));
            rows.append(account.id).append('\t').append(encoded).append('\t').append(hex)
                    .append('\t').append(account.balance).append('\t').append(account.revision).append('\n');
        }

        Map<String, Integer> totals = accounts.stream().collect(Collectors.groupingBy(
                account -> account.label, TreeMap::new, Collectors.summingInt(account -> account.balance)));
        StringBuilder summary = new StringBuilder();
        summary.append("accounts=").append(accounts.size()).append('\n');
        summary.append("applied=").append(ledger.applied()).append('\n');
        summary.append("rejected=").append(ledger.rejects().size()).append('\n');
        summary.append("resourcesClosed=").append(closed).append('\n');
        summary.append("bytesRead=").append(bytesRead).append('\n');
        for (Map.Entry<String, Integer> entry : totals.entrySet()) {
            summary.append("group=").append(URL.encode(entry.getKey(), "UTF-8"))
                    .append("\ttotal=").append(entry.getValue())
                    .append("\tsha256=").append(DigestUtils.sha256Hex(entry.getKey()).substring(0, 16))
                    .append('\n');
        }
        StringBuilder errors = new StringBuilder("location\tcode\tfingerprint\n");
        ledger.rejects().stream().sorted(Comparator.comparing((Reject reject) -> reject.location)
                .thenComparing(reject -> reject.code)).forEach(reject -> errors.append(reject.toLine()).append('\n'));

        Files.writeString(directory.resolve("accounts.tsv"), rows, StandardCharsets.UTF_8);
        Files.writeString(directory.resolve("summary.txt"), summary, StandardCharsets.UTF_8);
        Files.writeString(directory.resolve("rejects.tsv"), errors, StandardCharsets.UTF_8);
        return DigestUtils.sha256Hex(rows.toString() + summary + errors);
    }
}
