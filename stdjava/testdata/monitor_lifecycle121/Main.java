import java.util.concurrent.CountDownLatch;

public final class Main {
    static class Base { }
    static final class Leaf extends Base { }

    public static void main(String[] args) throws Exception {
        Leaf leaf = new Leaf();
        Base base = leaf;
        System.out.println("initial:" + Thread.holdsLock(leaf));
        synchronized (base) {
            System.out.println("base:" + Thread.holdsLock(base));
            System.out.println("leaf:" + Thread.holdsLock(leaf));
            synchronized (leaf) {
                System.out.println("reentrant:" + Thread.holdsLock(base));
            }
            Thread other = new Thread(() -> System.out.println("other:" + Thread.holdsLock(leaf)));
            other.start();
            other.join();
            System.out.println("outer:" + Thread.holdsLock(leaf));
        }
        System.out.println("released:" + Thread.holdsLock(base));
        try {
            synchronized ((Object) null) { }
        } catch (NullPointerException expected) {
            System.out.println("null:NullPointerException");
        }
        try {
            synchronized (leaf) {
                throw new IllegalStateException();
            }
        } catch (IllegalStateException expected) {
            System.out.println("abrupt:" + Thread.holdsLock(base));
        }
        CountDownLatch ready = new CountDownLatch(1);
        boolean[] notified = { false };
        Thread waiter = new Thread(() -> {
            synchronized (base) {
                synchronized (leaf) {
                    ready.countDown();
                    try {
                        while (!notified[0]) leaf.wait();
                    } catch (InterruptedException failure) {
                        throw new AssertionError(failure);
                    }
                    System.out.println("wait-restored:" + Thread.holdsLock(base));
                }
                System.out.println("wait-outer:" + Thread.holdsLock(leaf));
            }
        });
        waiter.start();
        ready.await();
        synchronized (leaf) {
            System.out.println("notifier:" + Thread.holdsLock(base));
            notified[0] = true;
            base.notifyAll();
        }
        waiter.join();
        System.out.println("after-wait:" + Thread.holdsLock(base));
    }
}
