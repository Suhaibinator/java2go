package prereq;
public class ReplaceProbe {
 static String trace="";
 static final Object lock=new Object();
 static final RuntimeException marker=new RuntimeException("marker");
 static class Text implements CharSequence {
  String id, text;boolean abrupt;
  Text(String id,String text,boolean abrupt){this.id=id;this.text=text;this.abrupt=abrupt;}
  public int length(){throw new AssertionError("length must not be consulted");}
  public char charAt(int i){throw new AssertionError("charAt must not be consulted");}
  public CharSequence subSequence(int a,int b){throw new AssertionError("subSequence must not be consulted");}
  public String toString(){trace+=id+Thread.holdsLock(lock);if(abrupt)throw marker;return text;}
 }
 static String receiver(String text){trace+="r";return text;}
 static CharSequence argument(String id,CharSequence text){trace+=id;return text;}
 static Character character(String id,Character c){trace+=id;return c;}
 static String units(String s){if(s==null)return "null";String r="";for(int i=0;i<s.length();i++){if(i>0)r+=",";r+=(int)s.charAt(i);}return r;}
 static void record(String id,String source,String target,String replacement){String out=source.replace(target,replacement);System.out.println(id+":"+units(out)+":"+(out==source));}
 public static void main(String[] args){
  String source=new String(new char[]{'a',(char)0xd800,'a',(char)0xdc00,0,'a'});
  record("units",source,new String(new char[]{'a',(char)0xd800}),new String(new char[]{(char)0xdc00,0}));
  record("missing",source,"xx","z");record("overlap","aaaaa","aa","b");record("same-one",source,"a","a");record("same-many","abab","ab","ab");
  record("empty",source,"","x");record("empty-both",source,"","");
  System.out.println("char:"+units(source.replace((char)0xd800,(char)0xdc00))+":"+(source.replace('a','a')==source)+":"+(source.replace('z','x')==source));
  synchronized(lock){
   trace="";String out=receiver("aba").replace(argument("a",new Text("T","a",false)),argument("b",new Text("U","z",false)));System.out.println("callbacks:"+out+":"+trace);
   trace="";try{receiver(null).replace(argument("a",new Text("T","a",false)),argument("b",new Text("U","z",false)));}catch(NullPointerException e){System.out.println("receiver-null:"+trace);}
   trace="";try{receiver("aba").replace(argument("a",null),argument("b",new Text("U","z",false)));}catch(NullPointerException e){System.out.println("target-null:"+trace);}
   trace="";try{"aba".replace(new Text("T",null,false),new Text("U","z",false));}catch(NullPointerException e){System.out.println("target-text-null:"+trace);}
   trace="";try{"aba".replace(new Text("T","a",false),(CharSequence)null);}catch(NullPointerException e){System.out.println("replacement-null:"+trace);}
   trace="";try{"aba".replace(new Text("T","a",true),new Text("U","z",false));}catch(RuntimeException e){System.out.println("abrupt:"+(e==marker)+":"+trace);}
   trace="";try{receiver(null).replace(character("a",null),character("b",'x'));}catch(NullPointerException e){System.out.println("char-unbox:"+trace);}
   trace="";System.out.println("builder:"+"aba".replace(new StringBuilder("a"),new StringBuilder("z")));
  }
 }
}
