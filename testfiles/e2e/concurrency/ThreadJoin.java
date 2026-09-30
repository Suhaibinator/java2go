class Worker extends Thread {
    private int id;
    private int[] results;

    Worker(int id, int[] results) {
        this.id = id;
        this.results = results;
    }

    public void run() {
        int total = 0;
        for (int i = 1; i <= id; i++) {
            total += i;
        }
        results[id] = total;
    }
}

public class ThreadJoin {
    public static void main(String[] args) throws InterruptedException {
        int n = 5;
        int[] results = new int[n];
        Worker[] workers = new Worker[n];

        for (int i = 0; i < n; i++) {
            workers[i] = new Worker(i, results);
        }
        for (int i = 0; i < n; i++) {
            workers[i].start();
        }
        for (int i = 0; i < n; i++) {
            workers[i].join();
        }

        int grand = 0;
        for (int i = 0; i < n; i++) {
            System.out.println("worker " + i + " = " + results[i]);
            grand += results[i];
        }
        System.out.println("grand " + grand);
    }
}
