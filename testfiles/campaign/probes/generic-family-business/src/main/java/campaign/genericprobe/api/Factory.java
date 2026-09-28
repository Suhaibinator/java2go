package campaign.genericprobe.api;

public interface Factory {
    <T> Adapter<T> create(Token<T> token);
}
