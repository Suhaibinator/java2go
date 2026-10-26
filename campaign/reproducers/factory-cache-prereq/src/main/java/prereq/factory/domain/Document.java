package prereq.factory.domain;

import org.apache.commons.lang3.mutable.MutableInt;

public final class Document {
    private final String id;
    private final StringBuilder body;
    private final MutableInt revisions = new MutableInt();

    public Document(String id, String initialBody) {
        this.id = id;
        this.body = new StringBuilder(initialBody);
    }

    public void append(String suffix) {
        body.append('|').append(suffix);
        revisions.increment();
    }

    public String snapshot() {
        return id + ":" + body + ":" + revisions.intValue();
    }
}
