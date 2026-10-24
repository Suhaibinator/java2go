package challenge.app;

import campaign.data.model.Account;
import control.factory.MapFactories;
import challenge.real.RealKeys;
import challenge.decoy.LookalikeKeys;
import challenge.state.Audit;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.function.Supplier;
import java.util.stream.Collectors;

// Additive Supplier28 probe. Compile against the frozen original28 source graph;
// neither control.app.Main nor its source/dependency bytes are replaced.
public final class OwnerProbe {
    private static String key(Account row, RealKeys<Integer> real,
                              LookalikeKeys decoy, Audit audit) {
        MapFactories.Named entryFactory = new MapFactories.Named(0);
        entryFactory.product.put(row.label, row.balance);
        Map.Entry<String, Integer> actual = entryFactory.product.entrySet().iterator().next();
        challenge.shadow.Map.Entry<String, Integer> source =
            new challenge.shadow.Map.Entry<String, Integer>(row.label, row.balance);
        int nativeRoute = decoy.route(actual, audit);
        int sourceRoute = decoy.route(source, audit);
        if (nativeRoute != 7 || sourceRoute != 11)
            throw new IllegalStateException("formal-owner-route");
        String result = real.pick(actual, audit);
        if (result != row.label) throw new IllegalStateException("generic-key-identity");
        return result;
    }

    private static <M extends Map<String, Integer>> M collect(
            List<Account> rows, Supplier<M> factory, RealKeys<Integer> real,
            LookalikeKeys decoy, Audit audit, int throwOn) {
        return rows.stream().collect(Collectors.groupingBy(
            (Account row) -> {
                audit.classifiers++;
                return key(row, real, decoy, audit);
            }, factory,
            Collectors.summingInt((Account row) -> {
                audit.values++;
                if (audit.values == throwOn) throw audit.failure;
                return row.balance;
            })));
    }

    private static String map(Map<String, Integer> value) {
        StringBuilder result = new StringBuilder();
        for (Map.Entry<String, Integer> entry : value.entrySet()) {
            String label = entry.getKey();
            result.append(label.length()).append(':').append(label).append('=');
            result.append(entry.getValue()).append(';');
        }
        return result.toString();
    }

    private static String counters(Audit audit) {
        return audit.classifiers + ":" + audit.values + ":" + audit.realCalls +
            ":" + audit.lambdaCalls + ":" + audit.localCalls + ":" +
            audit.sourceRoutes + ":" + audit.objectRoutes + ":" + audit.canonicalBodyDonors;
    }

    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        List<Account> rows = new ArrayList<>();
        for (int i = 1; i < args.length; i++) {
            String[] fields = args[i].split("\\|", -1);
            if (fields.length != 3) throw new IllegalArgumentException("row-shape");
            rows.add(new Account(fields[0], fields[1], Integer.parseInt(fields[2])));
        }
        if (rows.size() < 4) throw new IllegalArgumentException("need-four-rows");
        Integer receiverToken = Integer.valueOf(seed);
        RealKeys<Integer> real = new RealKeys<Integer>(receiverToken);
        LookalikeKeys decoy = new LookalikeKeys();
        if (real.ownerToken() != receiverToken) throw new IllegalStateException("receiver-token");
        MapFactories.SourceProduct factory = new MapFactories.SourceProduct(seed);
        Supplier<?> erased = factory;
        Object before = erased.get();
        Audit firstCalls = new Audit();
        MapFactories.SourceMap<String, Integer> first = collect(rows, factory, real, decoy, firstCalls, -1);
        String snapshot = map(first);
        System.out.println("first=" + snapshot);
        System.out.println("first-effects=" + counters(firstCalls) + ":factory=" + factory.calls +
            ":same=" + (first == before && first == factory.product) +
            ":compute=" + first.computeCalls + ":replace=" + first.replaceCalls);

        // Reuse the same supplier and map after mutating the original Account
        // objects. A cached key/result or replacement supplier changes this row.
        rows.get(0).label = rows.get(0).label + "-r";
        rows.get(0).balance += seed % 7;
        rows.get(0).revision++;
        first.clear();
        List<Account> prefix = new ArrayList<>(rows.subList(0, (rows.size() + 1) / 2));
        Audit secondCalls = new Audit();
        MapFactories.SourceMap<String, Integer> second = collect(prefix, factory, real, decoy, secondCalls, -1);
        System.out.println("second=" + map(second));
        System.out.println("snapshot=" + snapshot);
        System.out.println("second-effects=" + counters(secondCalls) + ":factory=" + factory.calls +
            ":same=" + (second == first && second == before) +
            ":compute=" + second.computeCalls + ":replace=" + second.replaceCalls);

        second.clear();
        Audit emptyCalls = new Audit();
        MapFactories.SourceMap<String, Integer> empty = collect(Collections.emptyList(), factory, real, decoy, emptyCalls, -1);
        System.out.println("empty=" + empty.size() + ":" + counters(emptyCalls) +
            ":factory=" + factory.calls + ":same=" + (empty == before) +
            ":compute=" + empty.computeCalls + ":replace=" + empty.replaceCalls);

        MapFactories.SourceProduct aborted = new MapFactories.SourceProduct(seed);
        Audit stopped = new Audit();
        try {
            collect(rows, aborted, real, decoy, stopped, 2);
            throw new IllegalStateException("missing-mapper-stop");
        } catch (IllegalArgumentException expected) {
            System.out.println("abort=" + expected.getMessage() + ":same=" + (expected == stopped.failure) + ":" + counters(stopped) +
                ":factory=" + aborted.calls + ":compute=" + aborted.product.computeCalls +
                ":replace=" + aborted.product.replaceCalls);
            Map<?, ?> partial = aborted.product;
            for (Map.Entry<?, ?> entry : partial.entrySet()) {
                int[] accumulator = (int[]) entry.getValue();
                System.out.println("partial=" + entry.getKey() + ":" + accumulator[0]);
            }
        }
    }
}
