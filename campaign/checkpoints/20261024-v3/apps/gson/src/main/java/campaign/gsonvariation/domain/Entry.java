package campaign.gsonvariation.domain;
import java.util.ArrayList;
import java.util.List;
public final class Entry<T> {
    public int id;
    public T value;
    public List<String> tags = new ArrayList<>();
    public Entry() {}
}
