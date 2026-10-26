package campaign.genericprobe.api;

public abstract class Adapter<T> implements Slot<T> {
    private T value;

    protected Adapter(T value) {
        this.value = value;
    }

    @Override
    public T read() {
        return value;
    }

    public void write(T value) {
        this.value = value;
    }

    public abstract String label();
}
