package wire;
import audit.Trace;
import io.netty.buffer.ByteBuf;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.ByteToMessageDecoder;
import java.util.List;
import model.Packet;
public final class FrameDecoder extends ByteToMessageDecoder {
 private final Trace trace;
 public FrameDecoder(Trace trace) { this.trace = trace; setCumulator(COMPOSITE_CUMULATOR); }
 protected void decode(ChannelHandlerContext ctx, ByteBuf bytes, List<Object> out) {
  if (bytes.readableBytes() < 2) return;
  int length = bytes.getUnsignedByte(bytes.readerIndex() + 1);
  if (bytes.readableBytes() < length + 2) return;
  int id = bytes.readUnsignedByte(); bytes.skipBytes(1);
  Packet packet = new Packet(id, bytes.readRetainedSlice(length));
  trace.add("decode:" + id + ":" + length);
  out.add(packet);
 }
 protected void decodeLast(ChannelHandlerContext ctx, ByteBuf bytes, List<Object> out) {
  trace.add("last:" + bytes.readableBytes());
  decode(ctx, bytes, out);
 }
}
