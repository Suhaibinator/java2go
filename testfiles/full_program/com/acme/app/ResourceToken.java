package com.acme.app;

public class ResourceToken implements AutoCloseable {
    private CloseTrace trace;
    private String label;

    public ResourceToken(CloseTrace trace, String label) {
        this.trace = trace;
        this.label = label;
    }

    public void close() {
        trace.append(label);
    }
}
