package probe;

import java.io.Serializable;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.atomic.AtomicIntegerArray;
import java.util.concurrent.atomic.AtomicLongArray;

public class AtomicArraysProbe {
  static String evaluations = "";
  static int index() { evaluations += "i"; return -1; }
  static int intValue() { evaluations += "v"; return 9; }
  static long longValue() { evaluations += "v"; return 9L; }
  static void failure(String name, Runnable action) {
    try { action.run(); System.out.println(name + ":missing"); }
    catch (Throwable e) { System.out.println(name + ":" + e.getClass().getName()); }
  }

  public static void main(String[] args) throws Exception {
    int[] seedI = {Integer.MIN_VALUE, Integer.MAX_VALUE, 7};
    long[] seedL = {Long.MIN_VALUE, Long.MAX_VALUE, 9007199254740993L};
    AtomicIntegerArray ints = new AtomicIntegerArray(seedI);
    AtomicLongArray longs = new AtomicLongArray(seedL);
    seedI[2] = 88; seedL[2] = 88L;
    System.out.println("copy:" + ints.length() + ":" + ints.get(2) + ":" + longs.length() + ":" + longs.get(2));
    ints.set(2, 11); longs.set(2, 9007199254741001L);
    Object erasedI = ints; Object erasedL = longs;
    AtomicIntegerArray aliasI = (AtomicIntegerArray) erasedI;
    AtomicLongArray aliasL = (AtomicLongArray) erasedL;
    Serializable serialI = ints; Serializable serialL = longs;
    System.out.println("identity:" + (aliasI == ints) + ":" + (aliasL == longs) + ":" + (serialI == erasedI) + ":" + (serialL == erasedL));
    System.out.println("nominal:" + (erasedI instanceof AtomicIntegerArray) + ":" + (erasedI instanceof AtomicLongArray) + ":" + (erasedL instanceof Serializable));
    System.out.println("classes:" + erasedI.getClass().getName() + ":" + erasedL.getClass().getName());
    aliasI.set(2, 19); aliasL.set(2, 23L);
    System.out.println("alias:" + ints.get(2) + ":" + longs.get(2) + ":source:" + seedI[2] + ":" + seedL[2]);
    AtomicIntegerArray emptyI = new AtomicIntegerArray(0);
    AtomicLongArray emptyL = new AtomicLongArray(0);
    System.out.println("empty:" + emptyI.length() + ":" + emptyL.length() + ":" + (emptyI == new AtomicIntegerArray(0)) + ":" + (emptyL == new AtomicLongArray(0)));
    System.out.println("overflowI:" + ints.getAndAdd(1, 1) + ":" + ints.get(1) + ":" + ints.addAndGet(1, -1));
    System.out.println("overflowL:" + longs.getAndAdd(1, 1L) + ":" + longs.get(1) + ":" + longs.addAndGet(1, -1L));
    System.out.println("exchange:" + ints.getAndSet(2, 31) + ":" + longs.getAndSet(2, 37L));
    System.out.println("cas:" + ints.compareAndSet(2, 31, 41) + ":" + ints.compareAndSet(2, 31, 43) + ":" + longs.compareAndSet(2, 37L, 47L) + ":" + longs.compareAndSet(2, 37L, 53L));

    failure("negativeI", () -> { new AtomicIntegerArray(-1); });
    failure("negativeL", () -> { new AtomicLongArray(-1); });
    failure("nullCopyI", () -> { new AtomicIntegerArray((int[]) null); });
    failure("nullCopyL", () -> { new AtomicLongArray((long[]) null); });
    failure("getNegativeI", () -> { ints.get(-1); });
    failure("getEndL", () -> { longs.get(longs.length()); });
    failure("setEndI", () -> { ints.set(ints.length(), 1); });
    failure("setNegativeL", () -> { longs.set(-1, 1L); });
    failure("casEndI", () -> { ints.compareAndSet(ints.length(), 0, 1); });
    failure("addNegativeL", () -> { longs.getAndAdd(-1, 1L); });
    failure("badCastI", () -> { AtomicIntegerArray bad = (AtomicIntegerArray) erasedL; System.out.println(bad.length()); });
    failure("badCastL", () -> { AtomicLongArray bad = (AtomicLongArray) erasedI; System.out.println(bad.length()); });
    Object nullObject = null;
    System.out.println("nullCast:" + ((AtomicIntegerArray) nullObject == null) + ":" + ((AtomicLongArray) nullObject == null));
    AtomicIntegerArray nullI = null; AtomicLongArray nullL = null;
    evaluations = "";
    failure("nullSetI", () -> { nullI.set(index(), intValue()); });
    System.out.println("evaluationI:" + evaluations);
    evaluations = "";
    failure("nullSetL", () -> { nullL.set(index(), longValue()); });
    System.out.println("evaluationL:" + evaluations);

    // Start gate predates writes; publication is through atomic array stores.
    CountDownLatch ready = new CountDownLatch(2);
    CountDownLatch start = new CountDownLatch(1);
    int[] ordinaryI = new int[1]; long[] ordinaryL = new long[1];
    AtomicIntegerArray flagsI = new AtomicIntegerArray(2);
    AtomicLongArray flagsL = new AtomicLongArray(2);
    Thread writer = new Thread(() -> {
      ready.countDown();
      try { start.await(); } catch (InterruptedException e) { throw new AssertionError(e); }
      ordinaryI[0] = 101; ordinaryL[0] = 9007199254740997L;
      flagsI.set(0, 1); flagsL.set(0, 1L);
    });
    Thread reader = new Thread(() -> {
      ready.countDown();
      try { start.await(); } catch (InterruptedException e) { throw new AssertionError(e); }
      while (flagsI.get(0) == 0 || flagsL.get(0) == 0L) { }
      flagsI.set(1, ordinaryI[0]); flagsL.set(1, ordinaryL[0]);
    });
    writer.start(); reader.start(); ready.await(); start.countDown(); writer.join(); reader.join();
    System.out.println("publication:" + flagsI.get(1) + ":" + flagsL.get(1));

    CountDownLatch contenders = new CountDownLatch(2);
    CountDownLatch release = new CountDownLatch(1);
    AtomicIntegerArray raceI = new AtomicIntegerArray(2);
    AtomicLongArray raceL = new AtomicLongArray(2);
    CountDownLatch casFinished = new CountDownLatch(2);
    Runnable contender = () -> {
      contenders.countDown();
      try { release.await(); } catch (InterruptedException e) { throw new AssertionError(e); }
      if (raceI.compareAndSet(0, 0, 1)) raceI.getAndAdd(1, 1);
      if (raceL.compareAndSet(0, 0L, 1L)) raceL.getAndAdd(1, 1L);
      casFinished.countDown();
      try { casFinished.await(); } catch (InterruptedException e) { throw new AssertionError(e); }
      for (int n = 0; n < 1000; n++) { raceI.getAndAdd(0, 1); raceL.getAndAdd(0, 1L); }
    };
    Thread first = new Thread(contender); Thread second = new Thread(contender);
    first.start(); second.start(); contenders.await(); release.countDown(); first.join(); second.join();
    System.out.println("contended:" + raceI.get(0) + ":" + raceI.get(1) + ":" + raceL.get(0) + ":" + raceL.get(1));
  }
}
