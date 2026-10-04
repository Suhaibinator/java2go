package probe.bridge.domain;

import java.util.List;
import probe.bridge.source.BaseSource;
import probe.bridge.state.Trace;

public final class SequenceSource extends BaseSource<CharSequence> {
    public SequenceSource(List<CharSequence> values, Trace trace) {
        super(values, trace);
    }
}
