package flow;
import audit.Trace;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.Unpooled;
import io.netty.buffer.UnpooledByteBufAllocator;
import io.netty.channel.ChannelFuture;
import io.netty.channel.ChannelFutureListener;
import io.netty.channel.ChannelHandlerContext;
import io.netty.channel.ChannelPromise;
import io.netty.channel.embedded.EmbeddedChannel;
import io.netty.handler.codec.DecoderException;
import io.netty.handler.codec.EncoderException;
import io.netty.handler.codec.MessageToMessageDecoder;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ExecutionException;
import model.Packet;
import model.Problem;
import wire.FrameDecoder;
import wire.ReplyEncoder;
public final class Interaction {
 private final int seed;
 private final Trace trace = new Trace();
 private final Problem inboundProblem = new Problem("inbound"), outboundProblem = new Problem("outbound");
 private final List<ByteBuf> fragments = new ArrayList<ByteBuf>();
 private final List<Packet> consumed = new ArrayList<Packet>(), replies = new ArrayList<Packet>();
 private final List<ChannelPromise> promises = new ArrayList<ChannelPromise>();
 private final EmbeddedChannel channel;
 private int outputs, inboundFailures, outboundFailures;
 public Interaction(int seed) {
  this.seed = seed;
  channel = new EmbeddedChannel();
  channel.config().setAllocator(new UnpooledByteBufAllocator(false));
  channel.pipeline().addLast("frames", new FrameDecoder(trace));
  channel.pipeline().addLast("reply", new ReplyEncoder(trace, outboundProblem, true));
  channel.pipeline().addLast("sink", new Sink());
 }
 private final class Sink extends MessageToMessageDecoder<Packet> {
  protected void decode(final ChannelHandlerContext ctx, Packet packet, List<Object> out) {
   consumed.add(packet); trace.add("sink:" + packet.id + ":" + packet.refCnt());
   if (packet.id == 255) throw inboundProblem;
   final Packet reply = new Packet(packet.id, packet.content().copy()); replies.add(reply);
   final ChannelPromise p = ctx.newPromise(); promises.add(p);
   p.addListener(new ChannelFutureListener() {
    public void operationComplete(ChannelFuture f) {
     trace.add("complete:" + reply.id + ":" + f.isSuccess() + ":" + reply.refCnt());
     f.addListener(new ChannelFutureListener() {
      public void operationComplete(ChannelFuture nested) { trace.add("nested:" + reply.id + ":" + nested.isDone()); }
     });
    }
   });
   ctx.executor().execute(new Runnable() {
    public void run() { trace.add("task:" + reply.id); ctx.writeAndFlush(reply, p); }
   });
  }
 }
 private void feed(int id, int size) {
  byte[] wire = new byte[size + 2]; wire[0] = (byte) id; wire[1] = (byte) size;
  for (int i = 0; i < size; i++) wire[i + 2] = (byte) (seed * 13 + id + i * 7);
  int offset = 0;
  while (offset < wire.length) {
   int n = Math.min(wire.length - offset, 1 + (seed + offset) % 3);
   ByteBuf fragment = Unpooled.buffer(n, n); fragment.writeBytes(wire, offset, n); fragments.add(fragment);
   try { channel.writeInbound(fragment); }
   catch (DecoderException e) {
    Trace.require(e.getCause() == inboundProblem, "decoder cause identity"); inboundFailures++;
    trace.add("inbound-cause:" + e.getClass().getSimpleName() + ":" + (e.getCause() == inboundProblem));
   }
   offset += n;
  }
  channel.runPendingTasks();
  ByteBuf reply;
  while ((reply = channel.readOutbound()) != null) {
   try { trace.add("out:" + reply.readUnsignedByte() + ":" + reply.readUnsignedByte() + ":" + reply.readUnsignedShort()); outputs++; }
   finally { reply.release(); }
  }
 }
 public void run() throws Exception {
  try {
   feed(1, 3 + seed % 4); feed(2, 4 + seed % 3); feed(255, 2); feed(254, 5);
   ChannelPromise failed = promises.get(promises.size() - 1);
   Throwable cause = failed.cause();
   Trace.require(cause instanceof EncoderException && cause.getCause() == outboundProblem, "encoder cause identity");
   try { failed.get(); throw new IllegalStateException("failure get returned"); }
   catch (ExecutionException e) { trace.add("get-cause:" + (e.getCause() == cause) + ":" + (e.getCause().getCause() == outboundProblem)); outboundFailures++; }
   channel.pipeline().replace("reply", "reply-recovered", new ReplyEncoder(trace, outboundProblem, false));
   feed(254, 1 + seed % 5);
   ByteBuf incomplete = Unpooled.buffer(1, 1).writeByte(9); fragments.add(incomplete); channel.writeInbound(incomplete);
   boolean leftovers = channel.finishAndReleaseAll();
   trace.add("finish:" + leftovers + ":" + channel.isOpen());
   Trace.record("workflow.events", trace.joined());
   Trace.record("workflow.counts", consumed.size() + ":" + replies.size() + ":" + outputs);
   Trace.record("workflow.failures", inboundFailures + ":" + outboundFailures);
   StringBuilder states = new StringBuilder();
   for (ChannelPromise p : promises) { if (states.length() > 0) states.append(';'); states.append(p.isDone()).append(':').append(p.isSuccess()).append(':').append(p.isCancelled()); }
   Trace.record("workflow.promises", states.toString());
   int fragmentRefs = 0, inputRefs = 0, replyRefs = 0;
   for (ByteBuf f : fragments) fragmentRefs += f.refCnt();
   for (Packet p : consumed) inputRefs += p.refCnt();
   for (Packet p : replies) replyRefs += p.refCnt();
   Trace.record("workflow.ownership", fragmentRefs + ":" + inputRefs + ":" + replyRefs);
   Trace.require(fragmentRefs == 0 && inputRefs == 0 && replyRefs == 0, "workflow leak");
   Trace.record("workflow.final", channel.isOpen() + ":" + (channel.readInbound() == null) + ":" + (channel.readOutbound() == null));
  } finally { channel.finishAndReleaseAll(); }
 }
}
