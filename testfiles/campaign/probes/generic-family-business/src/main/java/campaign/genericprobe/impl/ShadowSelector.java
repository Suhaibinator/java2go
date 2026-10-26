package campaign.genericprobe.impl;

import campaign.genericprobe.api.Adapter;
import campaign.genericprobe.api.Factory;
import campaign.genericprobe.api.Token;

public final class ShadowSelector<S> {
    private final Adapter<S> fallback;

    public ShadowSelector(Adapter<S> fallback) {
        this.fallback = fallback;
    }

    @SuppressWarnings("unchecked")
    public <S> Adapter<S> select(Token<S> token, Factory factory) {
        Adapter<S> result = factory.create(token);
        return result == null ? (Adapter<S>) fallback : result;
    }
}
