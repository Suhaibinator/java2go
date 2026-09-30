package parity.analytics.report;

import java.util.ArrayList;
import java.util.List;

public class StableRanker<T extends Ranked> {
    public List<T> sort(List<T> values) {
        List<T> sorted = new ArrayList<T>();
        for (T value : values) {
            sorted.add(value);
        }

        for (int i = 1; i < sorted.size(); i++) {
            T current = sorted.get(i);
            int cursor = i - 1;
            while (cursor >= 0 && comesBefore(current, sorted.get(cursor))) {
                sorted.set(cursor + 1, sorted.get(cursor));
                cursor--;
            }
            sorted.set(cursor + 1, current);
        }
        return sorted;
    }

    private boolean comesBefore(T left, T right) {
        if (left.primaryScore() != right.primaryScore()) {
            return left.primaryScore() > right.primaryScore();
        }
        return left.stableKey().compareTo(right.stableKey()) < 0;
    }
}
