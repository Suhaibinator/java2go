package probe;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.InvalidMarkException;
import java.security.MessageDigest;
public class CompactFlow {
 static String bytes(byte[] a) { String s=""; for(int i=0;i<a.length;i++) s+=":"+(a[i]&255); return s; }
 static String state(ByteBuffer b) { return b.capacity()+":"+b.position()+":"+b.limit()+":"+b.arrayOffset(); }
 static String mark(ByteBuffer b) { try { b.reset(); return "valid:"+b.position(); } catch(InvalidMarkException e) { return "invalid"; } }
 public static void main(String[] args) throws Exception {
  byte[] wire={91,92,1,2,3,4,5,6,7,8,93,94};
  ByteBuffer root=ByteBuffer.wrap(wire,2,8).order(ByteOrder.LITTLE_ENDIAN);
  ByteBuffer view=root.slice().order(ByteOrder.LITTLE_ENDIAN).position(2).mark().limit(7);
  ByteBuffer sibling=view.duplicate().order(ByteOrder.LITTLE_ENDIAN).position(1).mark();
  ByteBuffer result=view.compact();
  String shifted=bytes(wire);
  String cursors=state(root)+"/"+state(view)+"/"+state(sibling);
  String marks=mark(view)+"/"+mark(sibling);
  int word=view.getShort(0)&65535;
  view.put((byte)11).put((byte)12).flip();
  MessageDigest digest=MessageDigest.getInstance("SHA-256"); digest.update(view);
  String hash=bytes(digest.digest());
  ByteBuffer nested=sibling.position(2).limit(6).slice().position(1).mark();
  nested.compact(); String nestedBytes=bytes(wire); String nestedState=state(nested)+"/"+state(sibling);
  ByteBuffer empty=ByteBuffer.allocate(4).order(ByteOrder.LITTLE_ENDIAN).position(4).mark();
  empty.compact(); String emptyState=state(empty)+":"+mark(empty)+":"+(empty.order()==ByteOrder.LITTLE_ENDIAN);
  ByteBuffer zero=ByteBuffer.allocate(0).mark(); zero.compact();
  ByteBuffer full=ByteBuffer.wrap(new byte[]{17,18,19}).mark(); full.compact();
  String nullResult="accepted"; try { ((ByteBuffer)null).compact(); } catch(NullPointerException e) { nullResult="null"; }
  System.out.print("same:"+(result==view)+":"+(view.array()==wire)+":"+(view.order()==ByteOrder.LITTLE_ENDIAN)+"|shift"+shifted+"|state:"+cursors+"|marks:"+marks+"|word:"+word+"|digest"+hash+"|consumed:"+state(view)+"|nested"+nestedBytes+"|nested-state:"+nestedState+":"+mark(nested)+"|empty:"+emptyState+"|zero:"+state(zero)+":"+mark(zero)+"|full:"+state(full)+":"+mark(full)+bytes(full.array())+"|null:"+nullResult);
 }
}
