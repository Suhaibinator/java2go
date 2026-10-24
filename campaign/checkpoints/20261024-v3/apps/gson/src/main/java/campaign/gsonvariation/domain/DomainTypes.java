package campaign.gsonvariation.domain;
import com.google.gson.reflect.TypeToken;
public final class DomainTypes {
    private DomainTypes() {}
    @SuppressWarnings("unchecked")
    public static <D> TypeToken<Batch<D>> batch(Class<D> element) {
        return (TypeToken<Batch<D>>) (TypeToken<?>) TypeToken.getParameterized(Batch.class, element);
    }
}
