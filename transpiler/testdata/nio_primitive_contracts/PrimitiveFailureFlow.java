import java.nio.ByteBuffer;
import java.nio.ByteOrder;

public class PrimitiveFailureFlow {
    static String bytes(byte[] values) {
        String result = "";
        for (int i = 0; i < values.length; i++) result += ":" + (values[i] & 255);
        return result;
    }
    public static String run() {
        byte[] backing = {1,2,3,4,5,6,7,8,9,10,11,12};
        ByteBuffer root = ByteBuffer.wrap(backing).position(2).limit(10);
        ByteBuffer view = root.slice().order(ByteOrder.LITTLE_ENDIAN).position(7).mark();
        ByteBuffer duplicate = view.duplicate();
        String failures = "";
        try { view.getShort(); failures += ":accepted"; } catch (java.nio.BufferUnderflowException expected) { failures += ":Short-under"; }
        try { view.putShort((short)123); failures += ":accepted"; } catch (java.nio.BufferOverflowException expected) { failures += ":Short-over"; }
        try { view.getShort(-1); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Short-neg"; }
        try { view.putShort(2147483647, (short)123); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Short-huge"; }
        try { view.getShort(7); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Short-limit"; }
        try { view.putShort(7, (short)123); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Short-write"; }
        try { view.getInt(); failures += ":accepted"; } catch (java.nio.BufferUnderflowException expected) { failures += ":Int-under"; }
        try { view.putInt(123456); failures += ":accepted"; } catch (java.nio.BufferOverflowException expected) { failures += ":Int-over"; }
        try { view.getInt(-1); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Int-neg"; }
        try { view.putInt(2147483647, 123456); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Int-huge"; }
        try { view.getInt(5); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Int-limit"; }
        try { view.putInt(5, 123456); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Int-write"; }
        try { view.getLong(); failures += ":accepted"; } catch (java.nio.BufferUnderflowException expected) { failures += ":Long-under"; }
        try { view.putLong(123456789L); failures += ":accepted"; } catch (java.nio.BufferOverflowException expected) { failures += ":Long-over"; }
        try { view.getLong(-1); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Long-neg"; }
        try { view.putLong(2147483647, 123456789L); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Long-huge"; }
        try { view.getLong(1); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Long-limit"; }
        try { view.putLong(1, 123456789L); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Long-write"; }
        try { view.getFloat(); failures += ":accepted"; } catch (java.nio.BufferUnderflowException expected) { failures += ":Float-under"; }
        try { view.putFloat(1.5f); failures += ":accepted"; } catch (java.nio.BufferOverflowException expected) { failures += ":Float-over"; }
        try { view.getFloat(-1); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Float-neg"; }
        try { view.putFloat(2147483647, 1.5f); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Float-huge"; }
        try { view.getFloat(5); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Float-limit"; }
        try { view.putFloat(5, 1.5f); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Float-write"; }
        try { view.getDouble(); failures += ":accepted"; } catch (java.nio.BufferUnderflowException expected) { failures += ":Double-under"; }
        try { view.putDouble(2.25d); failures += ":accepted"; } catch (java.nio.BufferOverflowException expected) { failures += ":Double-over"; }
        try { view.getDouble(-1); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Double-neg"; }
        try { view.putDouble(2147483647, 2.25d); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Double-huge"; }
        try { view.getDouble(1); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Double-limit"; }
        try { view.putDouble(1, 2.25d); failures += ":accepted"; } catch (IndexOutOfBoundsException expected) { failures += ":Double-write"; }
        String cursors = root.position() + ":" + view.position() + ":" + duplicate.position();
        view.reset();
        ByteBuffer exact = root.slice(0, 8).order(ByteOrder.LITTLE_ENDIAN);
        exact.putLong(0, -1L).position(0).mark();
        long last = exact.getLong();
        exact.reset();
        exact.limit(7);
        String limitFailure = "accepted";
        try { exact.getLong(0); } catch (IndexOutOfBoundsException expected) { limitFailure = "limit"; }
        return "failures" + failures + "|cursors:" + cursors + "|marks:" + view.position() + ":" + exact.position()
            + "|last:" + last + "|limit:" + limitFailure + "|bytes" + bytes(backing);
    }
}
