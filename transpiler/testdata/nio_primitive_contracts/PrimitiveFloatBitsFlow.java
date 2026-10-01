import java.nio.ByteBuffer;
import java.nio.ByteOrder;

public class PrimitiveFloatBitsFlow {
    static String bytes(byte[] values) {
        String result = "";
        for (int i = 0; i < values.length; i++) result += ":" + (values[i] & 255);
        return result;
    }
    static String one(ByteOrder order) {
        byte[] backing = new byte[68];
        ByteBuffer root = ByteBuffer.wrap(backing).position(3).limit(67).order(order);
        ByteBuffer view = root.slice().order(order);
        view.putFloat(-0.0f).putDouble(-0.0d);
        view.putFloat(12, 1.5f).putDouble(16, -2.25d);
        view.putInt(24, 0x7fc12345).putLong(28, 0x7ff8123456789abcL);
        float fnan = view.getFloat(24);
        double dnan = view.getDouble(28);
        view.position(36).putFloat(fnan).putDouble(dnan);
        ByteBuffer copy = view.duplicate().order(order).position(0);
        float fzero = copy.getFloat();
        double dzero = copy.getDouble();
        view.putFloat(48, fzero).putDouble(52, dzero);
        copy.position(24);
        view.position(0).putFloat(copy.getFloat()).putDouble(copy.getDouble());
        String bits = view.getInt(0) + ":" + view.getLong(4) + ":" + view.getInt(12) + ":" + view.getLong(16)
            + ":" + view.getInt(36) + ":" + view.getLong(40) + ":" + view.getInt(48) + ":" + view.getLong(52);
        return order.toString() + "|bits:" + bits + "|cursors:" + root.position() + ":" + view.position() + ":" + copy.position()
            + "|bytes" + bytes(backing);
    }
    public static String run() { return one(ByteOrder.BIG_ENDIAN) + "|little:" + one(ByteOrder.LITTLE_ENDIAN); }
}
