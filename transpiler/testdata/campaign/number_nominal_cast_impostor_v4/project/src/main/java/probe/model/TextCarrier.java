package probe.model;

import probe.api.Carrier;

public final class TextCarrier extends Carrier<String> {
    private String value;
    @Override public String get() { return value; }
    @Override public void put(String value) { this.value = value; remember(value); }
}
