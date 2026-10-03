package model;
import io.netty.buffer.ByteBuf;
import io.netty.buffer.DefaultByteBufHolder;
public final class Packet extends DefaultByteBufHolder {
 public final int id;
 public Packet(int id, ByteBuf bytes) { super(bytes); this.id = id; }
}
