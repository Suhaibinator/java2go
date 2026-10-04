package probe;import java.io.RandomAccessFile;public class Main {
 static String units(String s){if(s==null)return "null";String out="";for(int i=0;i<s.length();i++){out+=(int)s.charAt(i)+",";}return out;}
 static String observe(String path,String mode){try{RandomAccessFile f=new RandomAccessFile(path,mode);String out="ok:"+f.length()+":"+f.getFilePointer()+":"+(f.getChannel()==f.getChannel());f.close();return out;}catch(Exception e){return e.getClass().getName()+":"+units(e.getMessage());}}
 public static void main(String[] args){
 System.out.println("empty:"+observe("","r"));System.out.println("notdir:"+observe("files/cursor.dat/child","rw"));System.out.println("readonly-r:"+observe("files/readonly.dat","r"));System.out.println("readonly-rw:"+observe("files/readonly.dat","rw"));
 System.out.println("missing-surrogate:"+observe("files/missing-\uD800.dat","r"));System.out.println("missing-unicode:"+observe("files/missing-\u00E9\uD83D\uDE00.dat","r"));
 System.out.println("normalized-missing:"+observe("files///missing.dat///","r"));String longName="files/";for(int i=0;i<300;i++)longName+="a";System.out.println("long:"+observe(longName,"rw"));
 }
}
