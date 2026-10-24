package campaign.gsonvariation.domain;
import java.util.ArrayList;
import java.util.List;
public final class Batch<T> {
    public String region;
    public List<Entry<T>> entries = new ArrayList<>();
    public Batch() {}
}
