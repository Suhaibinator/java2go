package probe;
import java.io.RandomAccessFile;import java.nio.ByteBuffer;import java.nio.channels.FileChannel;
public class Main {
 static String message(FileChannel c,ByteBuffer b)throws Exception {try{c.read(b);return null;}catch(IllegalArgumentException expected){return expected.getMessage();}}
 public static void main(String[] args)throws Exception {
  RandomAccessFile f=new RandomAccessFile("message.dat","rw");FileChannel c=f.getChannel();
  String first=message(c,ByteBuffer.allocate(0).asReadOnlyBuffer());String second=message(c,ByteBuffer.allocate(2).asReadOnlyBuffer());
  System.out.println((first==second)+":"+(first=="Read-only buffer")+":"+c.isOpen());f.close();
 }
}
