import java.nio.ByteBuffer;
import java.nio.ByteOrder;

public class PrimitiveEndianFlow {
    static String bytes(byte[] values) {
        String result = "";
        for (int i = 0; i < values.length; i++) result += ":" + (values[i] & 255);
        return result;
    }
    static String one(ByteOrder order) {
        byte[] backing = new byte[38];
        ByteBuffer root = ByteBuffer.wrap(backing).position(3).limit(35).order(order);
        ByteBuffer view = root.slice().order(order);
        view.position(1).mark();
        ByteBuffer duplicate = view.duplicate().order(order);
        view.putShort((short)-12345).putInt(-123456789).putLong(-1234567890123456789L);
        String afterPut = view.position() + ":" + duplicate.position() + ":" + root.position();
        duplicate.mark();
        short s = duplicate.getShort();
        int i = duplicate.getInt();
        long l = duplicate.getLong();
        duplicate.reset();
        view.putShort(17, (short)0x1234).putInt(19, 0x12345678).putLong(23, 0x1020304050607080L);
        String absolute = view.getShort(17) + ":" + view.getInt(19) + ":" + view.getLong(23);
        String stable = view.position() + ":" + duplicate.position() + ":" + root.position();
        root.putInt(5, 0x01020304);
        int alias = view.getInt(2);
        return order.toString() + "|values:" + s + ":" + i + ":" + l
            + "|absolute:" + absolute + "|cursors:" + afterPut + ":" + stable
            + "|alias:" + alias + "|offset:" + view.arrayOffset() + "|bytes" + bytes(backing);
    }
    public static String run() {
        ByteBuffer root = ByteBuffer.allocate(16).order(ByteOrder.LITTLE_ENDIAN).position(2).mark();
        ByteBuffer slice = root.slice();
        ByteBuffer indexed = root.slice(1, 8);
        ByteBuffer duplicate = root.duplicate();
        String defaults = ByteBuffer.allocate(1).order().toString() + ":" + ByteBuffer.wrap(new byte[1]).order().toString();
        String derived = root.order().toString() + ":" + slice.order().toString() + ":" + indexed.order().toString() + ":" + duplicate.order().toString();
        slice.order(ByteOrder.LITTLE_ENDIAN);
        String independent = root.order().toString() + ":" + duplicate.order().toString();
        root.order(null);
        String nullOrder = root.order().toString();
        root.clear().order(ByteOrder.LITTLE_ENDIAN).putInt(0x01020304).flip();
        root.mark().getInt(); root.reset();
        return "default:" + defaults + "|derived:" + derived + "|independent:" + independent
            + "|null-order:" + nullOrder + "|state-order:" + root.order().toString() + ":" + root.position()
            + "|big:" + one(ByteOrder.BIG_ENDIAN) + "|little:" + one(ByteOrder.LITTLE_ENDIAN);
    }
}
