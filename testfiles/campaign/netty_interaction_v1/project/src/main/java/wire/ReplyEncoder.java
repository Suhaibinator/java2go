package wire;
import audit.Trace;
import io.netty.buffer.ByteBuf;
import io.netty.channel.ChannelHandlerContext;
import io.netty.handler.codec.MessageToMessageEncoder;
import java.util.List;
import model.Packet;
import model.Problem;
public final class ReplyEncoder extends MessageToMessageEncoder<Packet> {
 private final Trace trace;
 private final Problem problem;
 private final boolean reject;
 public ReplyEncoder(Trace trace, Problem problem, boolean reject) { this.trace = trace; this.problem = problem; this.reject = reject; }
 protected void encode(ChannelHandlerContext ctx, Packet packet, List<Object> out) {
  trace.add("encode:" + packet.id + ":" + packet.refCnt());
  if (reject && packet.id == 254) throw problem;
  ByteBuf source = packet.content(); int sum = 0;
  for (int i = source.readerIndex(); i < source.writerIndex(); i++) sum = (sum * 31 + source.getUnsignedByte(i)) & 65535;
  ByteBuf reply = ctx.alloc().heapBuffer(4, 4);
  reply.writeByte(packet.id).writeByte(source.readableBytes()).writeShort(sum);
  out.add(reply);
 }
}
