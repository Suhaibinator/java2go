package buffers;
import audit.Trace;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.CompositeByteBuf;
import io.netty.buffer.Unpooled;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
public final class CompositeFlow {
 public static void run(int seed) {
  ByteBuf a = Unpooled.buffer(4, 4), b = Unpooled.buffer(4, 4);
  CompositeByteBuf all = Unpooled.compositeBuffer(4);
  ByteBuf held = null;
  try {
   a.writeInt(seed * 65537); b.writeInt(seed ^ 0x12345678);
   all.addComponents(true, a, b);
   held = all.retainedSlice(2, 4).asReadOnly();
   Trace.record("composite.start", all.numComponents() + ":" + all.readableBytes() + ":" + all.refCnt());
   ByteBuffer[] parts = all.nioBuffers(0, 8);
   ByteBuffer first = parts[0].duplicate().order(ByteOrder.LITTLE_ENDIAN);
   first.mark(); int before = first.getInt(); first.reset(); first.putInt(before ^ seed);
   Trace.record("composite.nio", parts.length + ":" + before + ":" + first.position() + ":" + parts[0].position());
   Trace.record("composite.share", all.getInt(0) + ":" + held.getUnsignedShort(0) + ":" + held.isReadOnly());
   all.release();
   Trace.record("composite.lifetime", all.refCnt() + ":" + held.getInt(0));
   held.release(); held = null;
   Trace.record("composite.cleanup", all.refCnt() + ":" + a.refCnt() + ":" + b.refCnt());
   Trace.require(all.refCnt() == 0 && a.refCnt() == 0 && b.refCnt() == 0, "composite leak");
  } finally {
   if (held != null && held.refCnt() > 0) held.release();
   if (all.refCnt() > 0) all.release(all.refCnt());
  }
 }
}
