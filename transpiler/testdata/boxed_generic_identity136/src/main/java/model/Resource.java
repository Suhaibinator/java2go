package model;

public class Resource implements AutoCloseable {
    private final String name;
    public Resource(String name) { this.name = name; State.opened++; State.mark("+" + name); }
    public void close() { State.closed++; State.mark("-" + name); }
}
