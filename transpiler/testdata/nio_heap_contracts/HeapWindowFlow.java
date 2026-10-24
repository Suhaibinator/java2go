import java.nio.ByteBuffer;

public class HeapWindowFlow {
    static String bytes(byte[] values) {
        String result = "";
        for (int i = 0; i < values.length; i++) result += ":" + (values[i] & 255);
        return result;
    }
    public static String run() {
        String text = "A😀Ω";
        byte[] wire = new byte[4 + text.length() * 2];
        wire[0] = 91; wire[1] = 92; wire[wire.length - 2] = 93; wire[wire.length - 1] = 94;
        for (int i = 0; i < text.length(); i++) {
            int unit = text.charAt(i);
            wire[2 + 2 * i] = (byte) (unit >>> 8);
            wire[3 + 2 * i] = (byte) unit;
        }
        ByteBuffer original = ByteBuffer.wrap(wire, 2, text.length() * 2);
        String initial = original.capacity() + ":" + original.position() + ":" + original.limit() + ":" + original.arrayOffset();
        ByteBuffer view = original.slice();
        ByteBuffer duplicate = view.duplicate();
        String window = view.capacity() + ":" + view.position() + ":" + view.limit() + ":" + view.arrayOffset() + ":" + (view.array() == wire);
        view.position(2).mark();
        byte[] first = new byte[2];
        view.get(first);
        view.reset();
        duplicate.position(6).limit(8);
        duplicate.put((byte) 3).put((byte) 88);
        ByteBuffer tail = duplicate.duplicate().clear().position(6).limit(8).slice();
        ByteBuffer merge = ByteBuffer.allocate(8);
        ByteBuffer head = view.duplicate().position(0).limit(6);
        merge.put(head).put(tail).flip();
        byte[] merged = new byte[8];
        merge.get(merged);
        byte[] untouched = new byte[2];
        original.get(untouched);
        int high = ((merged[2] & 255) << 8) | (merged[3] & 255);
        int low = ((merged[4] & 255) << 8) | (merged[5] & 255);
        return "units:" + text.length() + ":" + (int) text.charAt(1) + ":" + (int) text.charAt(2)
            + "|wrap:" + initial + "|view:" + window
            + "|cursors:" + original.position() + ":" + view.position() + ":" + duplicate.position() + ":" + head.position() + ":" + tail.position() + ":" + merge.position()
            + "|surrogates:" + high + ":" + low + "|first" + bytes(first) + "|merged" + bytes(merged) + "|wire" + bytes(wire);
    }
}
