package probe;

import java.io.RandomAccessFile;
import java.nio.ByteBuffer;
import java.nio.channels.FileChannel;
import java.util.concurrent.CountDownLatch;

public final class Main {
    private static String trace = "";
    private static FileChannel receiver(FileChannel c, boolean fail) {
        trace += "R";
        if (fail) throw new IllegalStateException("receiver");
        return c;
    }
    private static ByteBuffer source(ByteBuffer b, boolean fail) {
        trace += "S";
        if (fail) throw new IllegalStateException("source");
        return b;
    }
    private static long offset(long p, boolean fail) {
        trace += "P";
        if (fail) throw new IllegalStateException("position");
        return p;
    }
    private static void exception(Throwable t) {
        String message = t.getMessage();
        System.out.println("exception=" + t.getClass().getName() + ";message=" + message
            + ";null=" + (message == null) + ";negativeIdentity=" + (message == "Negative position")
            + ";ISE=" + (t instanceof IllegalStateException)
            + ";RTE=" + (t instanceof RuntimeException)
            + ";IO=" + (t instanceof java.io.IOException));
    }
    private static boolean consumeInterrupt() throws Exception {
        try { new CountDownLatch(0).await(); return false; }
        catch (InterruptedException e) { return true; }
    }
    private static void state(String tag, RandomAccessFile r, FileChannel c, ByteBuffer b) throws Exception {
        System.out.println(tag + ";open=" + c.isOpen() + ";position=" + b.position()
            + ";limit=" + b.limit() + ";capacity=" + b.capacity() + ";readonly=" + b.isReadOnly());
        if (c.isOpen()) System.out.println("cursor=" + r.getFilePointer() + ";length=" + r.length());
    }
    private static void bytes(RandomAccessFile r, long position, int count) throws Exception {
        long saved = r.getFilePointer();
        r.seek(position);
        String text = "";
        for (int i = 0; i < count; i++) text += (i == 0 ? "" : ",") + r.read();
        r.seek(saved);
        System.out.println("bytes@" + position + "=" + text + ";restoredCursor=" + r.getFilePointer());
    }
    private static void ordinary(String path, int seed) throws Exception {
        String[] modes = {"rw", "rws", "rwd"};
        for (int k = 0; k < modes.length; k++) {
            RandomAccessFile r = new RandomAccessFile(path, modes[k]);
            FileChannel c = r.getChannel();
            try {
                r.seek(seed % 11);
                byte[] a = {(byte)seed, 2, 3, 4, 5, 6, 7, 8};
                ByteBuffer owner = ByteBuffer.wrap(a, 1, 6);
                owner.mark();
                ByteBuffer view = k == 0 ? owner.duplicate() : k == 1 ? owner.slice() : owner.asReadOnlyBuffer();
                view.mark();
                int writtenBytes = 0;
                long p = 9 + k;
                writtenBytes += c.write(view, p + writtenBytes);
                System.out.println("mode=" + modes[k] + ";count=" + writtenBytes + ";sameChannel=" + (c == r.getChannel()));
                state("written", r, c, view);
                view.reset(); owner.reset();
                System.out.println("marks=" + view.position() + "," + owner.position() + ";alias=" + owner.get(2));
                bytes(r, p, writtenBytes);
                ByteBuffer empty = ByteBuffer.allocate(0).asReadOnlyBuffer();
                long length = r.length();
                System.out.println("emptyMax=" + c.write(empty, Long.MAX_VALUE) + ";growth=" + (r.length() - length));
            } finally { c.close(); r.close(); }
            System.out.println("closed=" + !c.isOpen());
        }
        RandomAccessFile r = new RandomAccessFile(path, "rw");
        FileChannel c = r.getChannel();
        try {
            r.seek(7);
            ByteBuffer b = ByteBuffer.wrap(new byte[] {11, 12}).asReadOnlyBuffer();
            long p = (1L << 31) + seed;
            int n = c.write(b, p);
            System.out.println("wideOffset=" + p + ";count=" + n + ";position=" + b.position()
                + ";cursor=" + r.getFilePointer() + ";length=" + r.length());
            bytes(r, p, n);
        } finally { r.close(); }
        System.out.println("aliasClose=" + !c.isOpen());
    }
    private static void validation(String path) throws Exception {
        String[] cases = {"nullNegative", "negative", "closedNull", "closedNegative", "closedEmpty",
            "readNull", "readNegative", "readEmpty", "readValid", "preNull", "preNegative",
            "preRead", "preClosed", "preValid", "preEmpty"};
        for (String name : cases) {
            RandomAccessFile r = new RandomAccessFile(path, name.startsWith("read") || name.equals("preRead") ? "r" : "rw");
            FileChannel c = r.getChannel();
            ByteBuffer b = ByteBuffer.wrap(new byte[] {1, 2, 3});
            b.position(1); b.mark();
            if (name.endsWith("Empty")) b.limit(1);
            ByteBuffer actual = name.contains("Null") || name.equals("nullNegative") ? null : b;
            long p = name.contains("Negative") || name.equals("negative") ? -1 : 0;
            if (name.startsWith("closed") || name.equals("preClosed")) c.close();
            if (name.startsWith("pre")) Thread.currentThread().interrupt();
            System.out.println("case=" + name);
            try { System.out.println("returned=" + c.write(actual, p)); }
            catch (Throwable t) { exception(t); }
            finally {
                System.out.println("pending=" + consumeInterrupt());
                state("after", r, c, b);
                b.reset(); System.out.println("mark=" + b.position());
                r.close(); c.close();
            }
            System.out.println("cleanup=" + !c.isOpen());
        }
        java.nio.channels.NonWritableChannelException canonical = new java.nio.channels.NonWritableChannelException();
        foreign.NonWritableChannelException foreign = new foreign.NonWritableChannelException();
        SourceException source = new SourceException();
        System.out.println("canonicalDescriptor"); exception(canonical);
        System.out.println("foreignDescriptor"); exception(foreign);
        System.out.println("sourceDescriptor"); exception(source);
    }
    private static void evaluation(String path) throws Exception {
        RandomAccessFile r = new RandomAccessFile(path, "rw");
        FileChannel c = r.getChannel();
        try {
            for (int i = 0; i < 5; i++) {
                trace = "";
                ByteBuffer b = ByteBuffer.wrap(new byte[] {21, 22});
                System.out.println("eval=" + i);
                try { System.out.println("returned=" + receiver(i == 4 ? null : c, i == 0)
                    .write(source(b, i == 1), offset(4, i == 2))); }
                catch (Throwable t) { exception(t); }
                System.out.println("trace=" + trace + ";position=" + b.position() + ";cursor=" + r.getFilePointer());
            }
        } finally { r.close(); }
    }
    private static void partial(String path, int seed) throws Exception {
        RandomAccessFile r = new RandomAccessFile(path, "rw");
        FileChannel c = r.getChannel();
        try {
            r.seek(seed % 11);
            byte[] a = {(byte)seed, 31, 32, 33, 34, 35, 36, 37};
            ByteBuffer owner = ByteBuffer.wrap(a);
            owner.position(1); owner.mark();
            ByteBuffer b = owner.asReadOnlyBuffer(); b.mark();
            int n = c.write(b, 4093L);
            System.out.println("partialCount=" + n + ";seed=" + seed);
            state("partial", r, c, b);
            System.out.println("aliasPosition=" + owner.position() + ";hasArray=" + b.hasArray());
            bytes(r, 4093L, n);
            System.out.println("boundaryWrite");
            try { System.out.println("returned=" + c.write(b, 4096L)); }
            catch (Throwable t) { exception(t); }
            state("boundary", r, c, b);
            b.reset(); owner.reset();
            System.out.println("marks=" + b.position() + "," + owner.position() + ";backing=" + owner.get(1));
        } finally { r.close(); c.close(); }
        System.out.println("cleanup=" + !c.isOpen() + ";pending=" + consumeInterrupt());
    }
    public static void main(String[] args) throws Exception {
        int seed = Integer.parseInt(args[1]);
        System.out.println("workflow=" + args[0] + ";seed=" + seed);
        if (args[0].equals("partial")) partial(args[2], seed);
        else { ordinary(args[2], seed); validation(args[2]); evaluation(args[2]); }
    }
    static final class SourceException extends RuntimeException { }
}
