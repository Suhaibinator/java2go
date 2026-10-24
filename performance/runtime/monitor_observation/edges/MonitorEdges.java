class ObservationBase {
    public boolean checkAlias(Object other) {
        synchronized (this) { return Thread.holdsLock(other); }
    }
}
class ObservationDerived extends ObservationBase {}

public final class MonitorEdges {
    public static String run() {
        ObservationDerived derived = new ObservationDerived();
        ObservationBase base = derived;
        boolean baseHeld;
        boolean derivedHeld;
        synchronized (base) {
            baseHeld = Thread.holdsLock(base);
            derivedHeld = Thread.holdsLock(derived);
        }
        String absent = null;
        String nullResult;
        try { nullResult = "returned-" + Thread.holdsLock(absent); }
        catch (NullPointerException expected) { nullResult = "NPE"; }
        return base.checkAlias(derived) + ":"
                + baseHeld + ":" + derivedHeld + ":" + nullResult;
    }
}
