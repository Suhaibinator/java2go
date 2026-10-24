package campaign.data2.ingest;

import campaign.data2.domain.LocatedCommand;
import java.util.ArrayList;
import java.util.List;

public final class Loaded {
    public final List<LocatedCommand> commands = new ArrayList<>();
    public final List<Issue> issues = new ArrayList<>();
    public int resourceCount;
    public int lineCount;
}
