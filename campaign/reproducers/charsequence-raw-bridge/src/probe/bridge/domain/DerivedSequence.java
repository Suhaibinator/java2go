package probe.bridge.domain;

import probe.bridge.source.SequenceBase;
import probe.bridge.state.Trace;

public final class DerivedSequence extends SequenceBase {
    public DerivedSequence(String label, String value, Trace trace, Runnable conversionAction) {
        super(label, value, trace, conversionAction);
    }
}
