package com.acme.app;

public class CloseTrace {
    private String value;

    public CloseTrace() {
        this.value = "";
    }

    public void append(String piece) {
        this.value += piece;
    }

    public String get() {
        return this.value;
    }
}
