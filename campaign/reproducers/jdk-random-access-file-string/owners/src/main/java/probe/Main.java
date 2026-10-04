package probe;public class Main<RandomAccessFile> {
 static int RandomAccessFile=7;static int String=8;
 static <T> T identity(T value){return value;}
 public static void main(java.lang.String[] args)throws Exception{
  java.io.RandomAccessFile f=new java.io.RandomAccessFile(identity(new java.lang.String("files/cursor.dat")),identity(new java.lang.String("r")));System.out.println(f.length()+":"+RandomAccessFile+":"+String);f.close();
  foreign.RandomAccessFile own=new foreign.RandomAccessFile("source","mode");System.out.println(own.value);
 }
}
