package campaign.genericprobe.app;

import campaign.genericprobe.api.Adapter;
import campaign.genericprobe.api.Factory;
import campaign.genericprobe.api.Slot;
import campaign.genericprobe.api.Token;
import campaign.genericprobe.impl.NamedFactory;
import campaign.genericprobe.impl.NumberAdapter;
import campaign.genericprobe.impl.ShadowSelector;

public final class Main {
    @SuppressWarnings({"rawtypes", "unchecked"})
    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        Adapter<Number> shared = new Adapter<Number>(seed + 1) {
            @Override
            public String label() {
                return "shared-anonymous";
            }
        };
        NumberAdapter bridge = new NumberAdapter(seed + 3);
        Factory named = new NamedFactory(shared, bridge);
        Factory anonymous = new Factory() {
            @Override
            @SuppressWarnings("unchecked")
            public <T> Adapter<T> create(Token<T> token) {
                return token.key().equals("shared") ? (Adapter<T>) shared : null;
            }
        };
        Token<Number> number = new Token<>("shared");
        Token<String> string = new Token<>("shared");
        Token<Integer> absent = new Token<>("absent");
        Token<Number> bridgeToken = new Token<>("bridge");
        ShadowSelector<Number> selector = new ShadowSelector<>(shared);

        int sum = 0;
        boolean identities = true;
        System.out.println("seed=" + seed);
        for (int step = 0; step < 4; step++) {
            Factory current = step % 2 == 0 ? named : anonymous;
            Factory other = step % 2 == 0 ? anonymous : named;
            Adapter<Number> numeric = current.create(number);
            Adapter<String> textView = current.create(string);
            Adapter<Number> otherView = other.create(number);
            identities &= (Object) numeric == (Object) textView && numeric == otherView;
            int next = seed + step * 7 + 1;
            numeric.write(next);
            int observed = otherView.read().intValue();
            sum += observed;
            System.out.println("step=" + step + ",source=" + (step % 2 == 0 ? "named" : "anonymous")
                    + ",value=" + observed + ",label=" + textView.label());
        }

        boolean missing = named.create(absent) == null && anonymous.create(absent) == null;
        boolean shadowed = (Object) selector.select(string, named) == shared
                && (Object) selector.select(absent, named) == shared;
        Slot<Number> throughInterface = bridge;
        Adapter<Number> throughBase = named.create(bridgeToken);
        int bridgeBefore = throughInterface.read().intValue();
        throughBase.write(seed + 21);
        int bridgeAfter = throughInterface.read().intValue();
        boolean bridgeIdentity = throughBase == (Adapter<Number>) bridge;

        Adapter rawShared = shared;
        rawShared.write("polluted-" + seed);
        boolean delayedSharedRead = false;
        try {
            Number ignored = shared.read();
            System.out.println("unexpected-number=" + ignored);
        } catch (ClassCastException expected) {
            delayedSharedRead = true;
        }
        String throughString = ((Adapter<String>) (Adapter) shared).read();

        Adapter rawBridge = bridge;
        rawBridge.write("bridge-polluted");
        boolean delayedBridgeRead = false;
        try {
            Number ignored = throughInterface.read();
            System.out.println("unexpected-bridge=" + ignored);
        } catch (ClassCastException expected) {
            delayedBridgeRead = true;
        }

        System.out.println("identity=" + identities + ",missing=" + missing + ",shadow=" + shadowed);
        System.out.println("bridge=" + bridgeBefore + "," + bridgeAfter + ",same=" + bridgeIdentity);
        System.out.println("pollution=" + delayedSharedRead + "," + delayedBridgeRead
                + ",text=" + throughString + ",sum=" + sum);
    }
}
