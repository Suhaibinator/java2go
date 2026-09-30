package campaign.genericprobe.impl;

import campaign.genericprobe.api.Adapter;

public final class NumberAdapter extends Adapter<Number> {
    public NumberAdapter(Number value) {
        super(value);
    }

    @Override
    public Integer read() {
        return super.read().intValue();
    }

    @Override
    public String label() {
        return "number";
    }
}
