package prereq.factory.impl;

import org.apache.commons.lang3.mutable.MutableInt;
import prereq.factory.api.Adapter;
import prereq.factory.api.Factory;
import prereq.factory.api.Token;
import prereq.factory.cache.Context;
import prereq.factory.domain.Order;

public final class OrderFactory implements Factory {
    private final int batch;
    private final MutableInt creations = new MutableInt();

    public OrderFactory(int batch) {
        this.batch = batch;
    }

    @Override
    @SuppressWarnings("unchecked")
    public <T> Adapter<T> create(Context context, Token<T> token) {
        context.probe('O', token.kind());
        if (!"order".equals(token.kind())) {
            return null;
        }
        creations.increment();
        return (Adapter<T>) new OrderAdapter(batch);
    }

    public int creations() {
        return creations.intValue();
    }

    private static final class OrderAdapter extends Adapter<Order> {
        private final int batch;

        private OrderAdapter(int batch) {
            this.batch = batch;
        }

        @Override
        public Order apply(Order input) {
            recordCall();
            if (!input.allocate(batch)) {
                throw new IllegalStateException("allocation rejected");
            }
            return input;
        }
    }
}
