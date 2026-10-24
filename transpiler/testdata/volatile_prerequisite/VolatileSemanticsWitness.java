import java.util.concurrent.CountDownLatch;

public final class VolatileSemanticsWitness {
    static class Parent {
        volatile int count;
        volatile long wide;
        volatile Object reference;
        static volatile Object shared;
        Parent() { wide = 4294967301L; }
        long read() { return wide; }
    }
    static final class Child extends Parent {
        int increment() { return ++count; }
    }
    static volatile boolean ready;
    static int payload;
    static Parent destination;
    static CountDownLatch loaded;
    static CountDownLatch release;
    static int rendezvous() {
        loaded.countDown();
        try { release.await(); } catch (InterruptedException e) { throw new AssertionError(e); }
        return 1;
    }
    public static void main(String[] args) throws Exception {
        Child child = new Child();
        Parent base = child;
        System.out.println("zero|" + base.count + "|" + (base.reference == null) + "|" + (Parent.shared == null));
        System.out.println("init|" + child.read());
        base.wide = Long.MIN_VALUE + 7;
        Object marker = new Object();
        child.reference = marker;
        Child.shared = marker;
        System.out.println("alias|" + child.wide + "|" + (base.reference == marker) + "|" + (Parent.shared == marker));
        base.count = 3;
        int result = child.count += (base.count = 9);
        int old = base.count++;
        int next = ++child.count;
        System.out.println("compound|" + result + "|" + old + "|" + next + "|" + base.count);
        destination = child;
        child.count = 0;
        loaded = new CountDownLatch(2);
        release = new CountDownLatch(1);
        Thread first = new Thread(() -> destination.count += rendezvous());
        Thread second = new Thread(() -> destination.count += rendezvous());
        first.start(); second.start();
        loaded.await(); release.countDown(); first.join(); second.join();
        // Both compound loads happen before release; both stores write one.
        System.out.println("lost-update|" + child.count);
        ready = false;
        Thread publisher = new Thread(() -> { payload = 73; ready = true; });
        publisher.start();
        while (!ready) { Thread.yield(); }
        System.out.println("publication|" + payload);
        publisher.join();
        Parent absent = null;
        int[] effects = new int[1];
        try { absent.count = ++effects[0]; } catch (NullPointerException expected) {}
        System.out.println("putfield-null-order|" + effects[0]);
        try { absent.count += ++effects[0]; } catch (NullPointerException expected) {}
        System.out.println("getfield-null-order|" + effects[0]);
    }
}
