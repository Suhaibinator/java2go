package probe;import foreign.ByteBuffer;public class Main {public static void main(String[] args){ByteBuffer b=new ByteBuffer();System.out.println(b.isReadOnly()+":"+b.asReadOnlyBuffer());}}
