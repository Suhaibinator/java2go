package campaign.data.ingest;

import campaign.data.model.Event;
import campaign.data.model.Reject;
import java.util.ArrayList;
import java.util.List;

public final class Loaded {
    public final List<Event> events = new ArrayList<>();
    public final List<Reject> rejects = new ArrayList<>();
    public int resourcesClosed;
    public long bytesRead;
}
