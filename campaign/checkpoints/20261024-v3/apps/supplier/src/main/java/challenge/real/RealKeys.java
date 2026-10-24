package challenge.real;

import java.util.Map;
import challenge.state.Audit;

public final class RealKeys<T> {
    private final T ownerToken;
    public RealKeys(T ownerToken) { this.ownerToken = ownerToken; }
    public T ownerToken() { return ownerToken; }

    // Method T is a distinct declaration from the receiver's T. The signature
    // Map is the imported JDK declaration; this body's local Map cannot own it.
    public <T> T pick(Map.Entry<T, Integer> value, Audit audit) {
        class Map { void touch() { audit.localCalls++; } }
        new Map().touch();
        Runnable action = () -> { audit.lambdaCalls++; };
        action.run();
        audit.realCalls++;
        return value.getKey();
    }
}
