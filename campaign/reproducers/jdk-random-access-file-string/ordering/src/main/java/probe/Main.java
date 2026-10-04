package probe;import java.io.RandomAccessFile;public class Main {
 static String units(String s){if(s==null)return "null";String out="";for(int i=0;i<s.length();i++){out+=(int)s.charAt(i)+",";}return out;}
 static String observe(String path,String mode){try{RandomAccessFile f=new RandomAccessFile(path,mode);String out="ok:"+f.length()+":"+f.getFilePointer()+":"+(f.getChannel()==f.getChannel());f.close();return out;}catch(Exception e){return e.getClass().getName()+":"+units(e.getMessage());}}
 static String log="";
 static String path(boolean fail){log+="P";if(fail)throw new IllegalStateException("path");return "files/cursor.dat";}
 static String mode(boolean fail){log+="M";if(fail)throw new IllegalStateException("mode");return "r";}
 static void order(boolean p,boolean m){log="";try{RandomAccessFile f=new RandomAccessFile(path(p),mode(m));f.close();System.out.println(log+":ok");}catch(Exception e){System.out.println(log+":"+e.getClass().getName()+":"+units(e.getMessage()));}}
 public static void main(String[] args){System.out.println("null-valid:"+observe(null,"r"));System.out.println("null-invalid:"+observe(null,"x"));System.out.println("null-null:"+observe(null,null));order(false,false);order(true,false);order(false,true);}
}
