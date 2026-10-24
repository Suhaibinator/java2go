package prereq.factory.impl;

import org.apache.commons.lang3.mutable.MutableInt;
import prereq.factory.api.Adapter;
import prereq.factory.api.Factory;
import prereq.factory.api.Token;
import prereq.factory.cache.Context;
import prereq.factory.domain.Document;

public final class DocumentFactory<Prefix extends CharSequence> implements Factory {
    private final Prefix prefix;
    private final MutableInt creations = new MutableInt();

    public DocumentFactory(Prefix prefix) {
        this.prefix = prefix;
    }

    @Override
    @SuppressWarnings("unchecked")
    public <T> Adapter<T> create(Context context, Token<T> token) {
        context.probe('D', token.kind());
        if (!"document".equals(token.kind())) {
            return null;
        }
        creations.increment();
        return (Adapter<T>) new DocumentAdapter(prefix.toString() + "#" + creations.intValue());
    }

    public int creations() {
        return creations.intValue();
    }

    private static final class DocumentAdapter extends Adapter<Document> {
        private final String suffix;

        private DocumentAdapter(String suffix) {
            this.suffix = suffix;
        }

        @Override
        public Document apply(Document input) {
            recordCall();
            input.append(suffix);
            return input;
        }
    }
}
