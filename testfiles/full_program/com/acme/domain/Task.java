package com.acme.domain;

public abstract class Task {
    String id;

    public Task(String id) {
        this.id = id;
    }

    public String name() {
        return this.id;
    }

    public abstract int run(String input);
}
