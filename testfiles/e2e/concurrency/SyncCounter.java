public class SyncCounter {
    private int count = 0;

    synchronized void increment() {
        count++;
    }

    int get() {
        return count;
    }

    public static void main(String[] args) throws InterruptedException {
        SyncCounter counter = new SyncCounter();
        int threadCount = 4;
        int perThread = 1000;

        Thread[] threads = new Thread[threadCount];
        for (int t = 0; t < threadCount; t++) {
            threads[t] = new Thread(new Runnable() {
                public void run() {
                    for (int i = 0; i < perThread; i++) {
                        counter.increment();
                    }
                }
            });
        }

        for (int t = 0; t < threadCount; t++) {
            threads[t].start();
        }
        for (int t = 0; t < threadCount; t++) {
            threads[t].join();
        }

        System.out.println(counter.get());
    }
}
