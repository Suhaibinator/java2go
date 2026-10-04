import java.nio.ByteBuffer;

public class HeapFailureFlow {
    static String bytes(byte[] values) {
        String result = "";
        for (int i = 0; i < values.length; i++) result += ":" + (values[i] & 255);
        return result;
    }
    public static String run() {
        byte[] backing = {11, 22, 33, 44, 55, 66};
        ByteBuffer root = ByteBuffer.wrap(backing).position(2).mark();
        ByteBuffer copy = root.duplicate();
        root.position(1);
        String mark = "accepted";
        try { root.reset(); } catch (java.nio.InvalidMarkException expected) { mark = "invalid"; }
        copy.reset();
        copy.limit(1);
        String truncated = "accepted";
        try { copy.reset(); } catch (java.nio.InvalidMarkException expected) { truncated = "invalid"; }
        String limit = "accepted";
        try { copy.limit(7); } catch (IllegalArgumentException expected) { limit = "illegal"; }
        ByteBuffer source = root.duplicate().position(1).limit(5);
        ByteBuffer target = root.duplicate().position(4).limit(6);
        String overflow = "accepted";
        try { target.put(source); } catch (java.nio.BufferOverflowException expected) { overflow = "overflow"; }
        String arrayRange = "accepted";
        try { target.put(backing, -1, 4); } catch (IndexOutOfBoundsException expected) { arrayRange = "range"; }
        String nullArray = "accepted";
        try { target.put((byte[]) null, -1, 4); } catch (NullPointerException expected) { nullArray = "null"; }
        String self = "accepted";
        try { target.put(target); } catch (IllegalArgumentException expected) { self = "self"; }
        String absolute = "accepted";
        try { target.put(6, (byte) 99); } catch (IndexOutOfBoundsException expected) { absolute = "range"; }
        byte[] output = {71, 72, 73};
        String getRange = "accepted";
        try { target.get(output, -1, 3); } catch (IndexOutOfBoundsException expected) { getRange = "range"; }
        String underflow = "accepted";
        try { target.get(output); } catch (java.nio.BufferUnderflowException expected) { underflow = "underflow"; }
        String nullBuffer = "accepted";
        try { target.put((ByteBuffer) null); } catch (NullPointerException expected) { nullBuffer = "null"; }
        String failedCursors = source.position() + ":" + target.position() + ":" + target.limit();
        target.mark().clear();
        String cleared = "accepted";
        try { target.reset(); } catch (java.nio.InvalidMarkException expected) { cleared = "invalid"; }
        target.position(3).mark().flip();
        String flipped = "accepted";
        try { target.reset(); } catch (java.nio.InvalidMarkException expected) { flipped = "invalid"; }
        target.position(1).mark().rewind();
        String rewound = "accepted";
        try { target.reset(); } catch (java.nio.InvalidMarkException expected) { rewound = "invalid"; }
        return "marks:" + mark + ":" + truncated + ":" + cleared + ":" + flipped + ":" + rewound
            + "|failures:" + limit + ":" + overflow + ":" + arrayRange + ":" + nullArray + ":" + self + ":" + absolute + ":" + getRange + ":" + underflow + ":" + nullBuffer
            + "|cursors:" + root.position() + ":" + copy.position() + ":" + copy.limit() + ":" + source.position() + ":" + target.position() + ":" + target.limit()
            + "|failed-cursors:" + failedCursors + "|backing" + bytes(backing) + "|output" + bytes(output);
    }
}
