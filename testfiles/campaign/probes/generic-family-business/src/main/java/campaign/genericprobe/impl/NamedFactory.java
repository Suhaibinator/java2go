package campaign.genericprobe.impl;

import campaign.genericprobe.api.Adapter;
import campaign.genericprobe.api.Factory;
import campaign.genericprobe.api.Token;

public final class NamedFactory implements Factory {
    private final Adapter<Number> shared;
    private final NumberAdapter bridge;

    public NamedFactory(Adapter<Number> shared, NumberAdapter bridge) {
        this.shared = shared;
        this.bridge = bridge;
    }

    @Override
    @SuppressWarnings("unchecked")
    public <T> Adapter<T> create(Token<T> token) {
        if (token.key().equals("shared")) {
            return (Adapter<T>) shared;
        }
        if (token.key().equals("bridge")) {
            return (Adapter<T>) bridge;
        }
        return null;
    }
}
