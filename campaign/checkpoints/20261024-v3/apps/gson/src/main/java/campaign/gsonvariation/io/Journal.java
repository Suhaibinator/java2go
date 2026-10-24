package campaign.gsonvariation.io;
import java.util.ArrayList;
import java.util.List;
public final class Journal {
    private final List<String> events = new ArrayList<>();
    public void add(String event) { events.add(event); }
    public List<String> snapshot() { return new ArrayList<>(events); }
}
