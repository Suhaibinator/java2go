package challenge.decoy;

import challenge.shadow.Map;
import challenge.state.Audit;

public final class LookalikeKeys {
    // This source-owned formal must remain unrelated to the canonical JDK body
    // occurrence. A JDK argument selects the Object overload below.
    public <T> int route(Map.Entry<T, Integer> value, Audit audit) {
        control.factory.MapFactories.Named donorFactory = new control.factory.MapFactories.Named(0);
        donorFactory.product.put("body-only", 0);
        java.util.Map.Entry<String, Integer> canonical = donorFactory.product.entrySet().iterator().next();
        Runnable donor = () -> {
            if (canonical.getKey().length() == 9) audit.canonicalBodyDonors++;
        };
        donor.run();
        T key = value.getKey();
        if (key == null) throw new IllegalStateException("source-key-null");
        audit.sourceRoutes++;
        return 11;
    }

    public int route(Object value, Audit audit) {
        if (value == null) throw new IllegalStateException("object-argument-null");
        audit.objectRoutes++;
        return 7;
    }
}
