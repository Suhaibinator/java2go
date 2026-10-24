package campaign.gsonvariation.state;
import campaign.gsonvariation.domain.Batch;
import campaign.gsonvariation.domain.Entry;
import java.time.LocalDate;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
public final class Ledger {
    private final Map<String, List<Entry<LocalDate>>> groups = new LinkedHashMap<>();
    private int rows;
    private boolean aliasesPreserved = true;
    public void apply(Batch<LocalDate> batch, int seed) {
        List<Entry<LocalDate>> staged = new ArrayList<>(batch.entries);
        for (Entry<LocalDate> entry : staged) {
            if (entry.value != null) entry.value = entry.value.plusDays(seed % 5);
        }
        if (!staged.isEmpty()) staged.get(0).tags.add("seed-" + seed);
        List<Entry<LocalDate>> target = groups.get(batch.region);
        if (target == null) { target = new ArrayList<>(); groups.put(batch.region, target); }
        int first = target.size();
        target.addAll(staged);
        if (!staged.isEmpty()) aliasesPreserved &= target.get(first) == batch.entries.get(0);
        rows += staged.size();
    }
    public Map<String, List<Entry<LocalDate>>> view() { return groups; }
    public int rows() { return rows; }
    public boolean aliasesPreserved() { return aliasesPreserved; }
}
