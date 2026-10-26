import java.nio.ByteBuffer;

public class HeapAliasFlow {
    static String bytes(byte[] values) {
        String result = "";
        for (int i = 0; i < values.length; i++) result += ":" + (values[i] & 255);
        return result;
    }
    public static String run() {
        byte[] backing = {7, 16, 32, 48, 64, 80, 96, 8};
        ByteBuffer root = ByteBuffer.wrap(backing, 1, 6);
        ByteBuffer window = root.slice();
        ByteBuffer source = window.duplicate().position(0).limit(4);
        ByteBuffer target = window.duplicate().position(2).limit(6);
        target.put(source);
        String overlap = bytes(backing);
        ByteBuffer nested = window.position(1).limit(5).slice();
        nested.put(0, (byte) 90);
        int absolute = nested.get(0) & 255;
        int before = nested.position();
        nested.get(backing, 0, 3);
        String afterAliasGet = bytes(backing);
        nested.clear().put(backing, 0, 3);
        ByteBuffer absoluteWindow = root.slice(2, 3);
        byte[] output = new byte[3];
        absoluteWindow.get(output);
        return "overlap" + overlap + "|alias-get" + afterAliasGet + "|final" + bytes(backing)
            + "|absolute:" + absolute + ":" + before + "|cursors:" + root.position() + ":" + window.position() + ":" + source.position() + ":" + target.position() + ":" + nested.position()
            + "|nested:" + nested.capacity() + ":" + nested.limit() + ":" + nested.arrayOffset()
            + "|indexed-slice:" + absoluteWindow.arrayOffset() + bytes(output);
    }
}
