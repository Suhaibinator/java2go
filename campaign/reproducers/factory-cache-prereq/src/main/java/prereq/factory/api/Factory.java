package prereq.factory.api;

import prereq.factory.cache.Context;

public interface Factory {
    <T> Adapter<T> create(Context context, Token<T> token);
}
