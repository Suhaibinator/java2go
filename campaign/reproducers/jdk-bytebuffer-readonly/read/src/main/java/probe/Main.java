package probe;
import java.io.*;import java.nio.*;import java.nio.channels.*;import java.util.concurrent.*;
public class Main {
 static String kind(FileChannel c,ByteBuffer b) {try{return "ok:"+c.read(b);}catch(Exception e){return e.getClass().getName()+":"+(e.getMessage()==null ? "null" : e.getMessage());}}
 public static void main(String[] args) throws Exception {
  RandomAccessFile f=new RandomAccessFile("readonly-consumer.dat","rw");FileChannel c=f.getChannel();ByteBuffer b=ByteBuffer.allocate(2).asReadOnlyBuffer();
  System.out.println("open:"+kind(c,b)+":"+b.position()+":"+c.isOpen());b.limit(0);System.out.println("empty:"+kind(c,b)+":"+b.position());c.close();System.out.println("closed-ro:"+kind(c,b));System.out.println("closed-null:"+kind(c,null));
  RandomAccessFile g=new RandomAccessFile("readonly-consumer.dat","r");FileChannel q=g.getChannel();System.out.println("readmode:"+kind(q,ByteBuffer.allocate(0).asReadOnlyBuffer()));g.close();
  RandomAccessFile h=new RandomAccessFile("readonly-consumer.dat","rw");FileChannel z=h.getChannel();Thread.currentThread().interrupt();System.out.println("interrupt:"+kind(z,ByteBuffer.allocate(0).asReadOnlyBuffer())+":"+z.isOpen());
  try {new CountDownLatch(1).await();System.out.println("retained:false");}catch(InterruptedException expected){System.out.println("retained:true");}
  try{h.getFilePointer();System.out.println("shared:false");}catch(IOException expected){System.out.println("shared:true");}
  RandomAccessFile j=new RandomAccessFile("readonly-consumer.dat","rw");FileChannel k=j.getChannel();Thread.currentThread().interrupt();System.out.println("interrupt-null:"+kind(k,null)+":"+k.isOpen());try{new CountDownLatch(1).await();}catch(InterruptedException expected){}j.close();
 }
}
