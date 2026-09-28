package prereq.generic.domain;

import org.apache.commons.lang3.mutable.MutableInt;

public final class Entry<Label extends CharSequence> {
    private final Label label;
    private final String kind;
    private final MutableInt touches = new MutableInt();

    public Entry(Label label, String kind) {
        this.label = label;
        this.kind = kind;
    }

    public String touch() {
        touches.increment();
        return label.toString() + "/" + kind;
    }

    public int touches() {
        return touches.intValue();
    }
}
