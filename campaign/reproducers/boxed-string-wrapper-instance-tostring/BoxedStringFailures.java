public class BoxedStringFailures {
 static String trace="";static Object lock=new Object();static Thread owner;static RuntimeException failure=new IllegalStateException("exact");
 static String units(String value){if(value==null)return "null";String out="";for(int i=0;i<value.length();i++)out+=(int)value.charAt(i)+",";return out;}
 static String text(int mode){trace+="T";if(Thread.currentThread()!=owner||!Thread.holdsLock(lock))throw new AssertionError("text execution");return mode==0?null:mode==1?new String(new char[]{'1',(char)0xd800}):"１２";}
 static int radix(int mode){trace+="R";if(Thread.currentThread()!=owner||!Thread.holdsLock(lock))throw new AssertionError("radix execution");if(mode==2)throw failure;return mode==0||mode==3?1:10;}
 static String parse(int mode){try{if(mode==0)return new Byte("１２８").toString();if(mode==1)return new Short("３２７６８").toString();if(mode==2)return new Integer((String)null).toString();if(mode==3)return new Long(new String(new char[]{'1',(char)0xdfff})).toString();if(mode==4)return Float.valueOf((String)null).toString();return new Double((String)null).toString();}catch(RuntimeException ex){return ex.getClass().getSimpleName()+":"+units(ex.getMessage());}}
 public static String run(){synchronized(lock){owner=Thread.currentThread();String out="";for(int mode=0;mode<4;mode++){trace="";try{Integer.valueOf(text(mode),radix(mode));out+="missing";}catch(RuntimeException ex){out+=trace+":"+(ex==failure)+":"+ex.getClass().getSimpleName()+":"+units(ex.getMessage());}out+="|";}for(int mode=0;mode<6;mode++)out+=parse(mode)+"|";return out;}}
 public static void main(String[] args){System.out.print(run());}
}
