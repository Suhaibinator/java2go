package probe;import java.io.RandomAccessFile;public class Main {
 static String units(String s){if(s==null)return "null";String out="";for(int i=0;i<s.length();i++){out+=(int)s.charAt(i)+",";}return out;}
 static String observe(String path,String mode){try{RandomAccessFile f=new RandomAccessFile(path,mode);String out="ok:"+f.length()+":"+f.getFilePointer()+":"+(f.getChannel()==f.getChannel());f.close();return out;}catch(Exception e){return e.getClass().getName()+":"+units(e.getMessage());}}
 public static void main(String[] args)throws Exception{
  for(String mode:new String[]{"r","rw","rws","rwd"})System.out.println("mode:"+mode+":"+observe("files/cursor.dat",new String(mode)));
  for(String mode:new String[]{"","x","rww","RW","r\u0000","\uD83D\uDE00","\uD800"})System.out.println("invalid:"+units(mode)+":"+observe("files/cursor.dat",mode));
  System.out.println("null-mode:"+observe("files/cursor.dat",null));
  System.out.println("missing:"+observe("files/missing.dat","r"));System.out.println("parent-missing:"+observe("no-parent/file","rw"));
  System.out.println("directory-r:"+observe("files","r"));System.out.println("directory-rw:"+observe("files","rw"));
  System.out.println("nul:"+observe("files/a\u0000b","rw"));System.out.println("nul-invalid:"+observe("files/a\u0000b","x"));System.out.println("nul-null:"+observe("files/a\u0000b",null));
  System.out.println("unicode:"+observe("files/\u00E9\uD83D\uDE00.dat","r"));System.out.println("surrogate:"+observe("files/\uD800.dat","r"));
  System.out.println("normalized:"+observe("files///cursor.dat///","r"));System.out.println("device:"+observe("/dev/null","r"));
  RandomAccessFile f=new RandomAccessFile(new String("files/cursor.dat"),new String("r"));f.seek(2);int v=f.read();System.out.println("cursor:"+v+":"+f.getFilePointer()+":"+f.length());f.getChannel().close();try{f.getFilePointer();}catch(java.io.IOException e){System.out.println("shared:"+e.getClass().getName());}f.close();
  java.io.File created=new java.io.File("files/new.dat");created.delete();System.out.println("create:"+observe("files/new.dat","rw")+":"+created.exists());created.delete();
 }
}
