package campaign.business.domain;

public abstract class Entity<K> {
    private final K id;

    protected Entity(K id) {
        this.id = id;
    }

    public final K id() {
        return id;
    }
}
