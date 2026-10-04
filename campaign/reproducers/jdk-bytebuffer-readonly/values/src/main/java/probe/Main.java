package probe;
import java.nio.*;import java.security.*;
public class Main {
 static String fail(int op) {
  byte[] bytes={10,11,12,13,14,15}; ByteBuffer mutable=ByteBuffer.wrap(bytes); mutable.position(1);mutable.limit(4);mutable.mark();ByteBuffer b=mutable.asReadOnlyBuffer();
  String error="none";
  try { switch(op) {
   case 0:b.put((byte)9);break;case 1:b.put(-1,(byte)9);break;
   case 2:b.put((byte[])null,-1,-1);break;case 3:b.put((byte[])null);break;
   case 4:b.put(b);break;case 5:b.put(-1,(ByteBuffer)null,-1,-1);break;
   case 6:b.put(-1,(byte[])null,-1,-1);break;case 7:b.put(-1,(byte[])null);break;
   case 8:b.put(new byte[0]);break;case 9:b.put(4,new byte[0]);break;
   case 10:b.put(0,b,0,0);break;case 11:b.position(b.limit());b.put((byte)9);break;
   case 12:b.compact();break;case 13:b.array();break;case 14:b.arrayOffset();break;
   case 15:b.putChar('x');break;case 16:b.putChar(-1,'x');break;
   case 17:b.putShort((short)2);break;case 18:b.putShort(-1,(short)2);break;
   case 19:b.putInt(2);break;case 20:b.putInt(-1,2);break;
   case 21:b.putLong(2L);break;case 22:b.putLong(-1,2L);break;
   case 23:b.putFloat(2.0f);break;case 24:b.putFloat(-1,2.0f);break;
   case 25:b.putDouble(2.0d);break;case 26:b.putDouble(-1,2.0d);break;
  }} catch (Exception e) {error=e.getClass().getName()+":"+(e instanceof UnsupportedOperationException)+":"+(e instanceof ReadOnlyBufferException ? e.getMessage()==null : true);}
  int p=b.position();b.reset();return op+":"+error+":"+p+":"+b.position()+":"+b.limit()+":"+bytes[0]+":"+bytes[1]+":"+bytes[2]+":"+bytes[3];
 }
 public static void main(String[] args) throws Exception {
  for(int i=0;i<27;i++)System.out.println(fail(i));
  byte[] a={0,1,2,3,4,5,6,7};ByteBuffer m=ByteBuffer.wrap(a,2,4);m.mark();m.order(ByteOrder.LITTLE_ENDIAN);ByteBuffer r=m.asReadOnlyBuffer();ByteBuffer d=r.duplicate();ByteBuffer s=r.slice();ByteBuffer t=r.slice(1,3);ByteBuffer rr=r.asReadOnlyBuffer();
  System.out.println("views:"+r.isReadOnly()+":"+d.isReadOnly()+":"+s.isReadOnly()+":"+t.isReadOnly()+":"+r.hasArray()+":"+(r==rr)+":"+(r==m)+":"+r.position()+":"+r.limit()+":"+r.capacity()+":"+(r.order()==ByteOrder.BIG_ENDIAN)+":"+(d.order()==ByteOrder.BIG_ENDIAN)+":"+s.get(0)+":"+t.get(0));
  m.put(2,(byte)42);System.out.println("alias:"+r.get(2)+":"+s.get(0)+":"+t.get(1)+":"+r.position()+":"+m.position());r.position(4);r.reset();System.out.println("mark:"+r.position()+":"+d.position());
  r.order(null);System.out.println("order-null:"+(r.order()==ByteOrder.LITTLE_ENDIAN));r.limit(5).position(3).mark().rewind();r.clear().position(2).flip();System.out.println("state:"+r.position()+":"+r.limit()+":"+r.isReadOnly());
  ByteBuffer dest=ByteBuffer.wrap(a);ByteBuffer src=dest.asReadOnlyBuffer();src.position(1);src.limit(4);dest.position(2);dest.put(src);System.out.println("source:"+dest.position()+":"+src.position()+":"+a[2]+":"+a[3]+":"+a[4]);
  ByteBuffer abs=ByteBuffer.wrap(new byte[]{0,1,2,3,4,5});abs.position(2);abs.put(1,abs,0,4);abs.put(0,new byte[]{8,9,10},1,2);abs.put(4,new byte[]{6,7});System.out.println("bulk:"+abs.position()+":"+abs.get(0)+":"+abs.get(1)+":"+abs.get(2)+":"+abs.get(3)+":"+abs.get(4)+":"+abs.get(5));
  ByteBuffer chars=ByteBuffer.allocate(4);chars.order(ByteOrder.LITTLE_ENDIAN);chars.putChar('\uD800');chars.putChar(2,'\uFFFF');chars.position(4);chars.flip();System.out.println("char:"+(int)chars.getChar()+":"+(int)chars.getChar(2)+":"+chars.position());
  ByteBuffer digest=ByteBuffer.wrap(new byte[]{99,1,2,3,88});digest.position(1);digest.limit(4);ByteBuffer ro=digest.asReadOnlyBuffer();MessageDigest md=MessageDigest.getInstance("SHA-256");md.update(ro);byte[] hash=md.digest();System.out.println("digest:"+ro.position()+":"+ro.limit()+":"+digest.position()+":"+hash[0]+":"+hash[1]+":"+digest.get(0));
 }
}
