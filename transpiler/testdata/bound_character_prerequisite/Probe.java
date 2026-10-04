package boundchar;
public class Probe {
 static final Object lock=new Object();
 static final RuntimeException marker=new RuntimeException("marker");
 static String trace="";
 static boolean failSecond=false;
 static String receiver(String text){trace+="r";return text;}
 static <C> C argument(String id,C value){trace+=id;if(failSecond&&id.equals("b"))throw marker;return value;}
 static <C extends java.lang.Character> String bound(String text,C oldChar,C newChar){return receiver(text).replace(argument("a",oldChar),argument("b",newChar));}
 static String direct(String text,java.lang.Character oldChar,java.lang.Character newChar){return receiver(text).replace(argument("a",oldChar),argument("b",newChar));}
 static String units(String text){if(text==null)return "null";String out="";for(int i=0;i<text.length();i++){if(i>0)out+=",";out+=(int)text.charAt(i);}return out;}
 static void show(String id,String text,java.lang.Character oldChar,java.lang.Character newChar,boolean generic){
  trace="";
  try{String result=generic?bound(text,oldChar,newChar):direct(text,oldChar,newChar);System.out.println(id+":ok:"+units(result)+":"+(result==text)+":"+trace+":"+Thread.holdsLock(lock));}
  catch(Throwable failure){System.out.println(id+":"+failure.getClass().getSimpleName()+":"+(failure==marker)+":"+trace+":"+Thread.holdsLock(lock));}
 }
 static class Character implements CharSequence {
  String id,text;
  Character(String id,String text){this.id=id;this.text=text;}
  public int length(){throw new AssertionError("length");}
  public char charAt(int index){throw new AssertionError("charAt");}
  public CharSequence subSequence(int a,int b){throw new AssertionError("subSequence");}
  public String toString(){trace+=id+Thread.holdsLock(lock);return text;}
 }
 static <C extends Character> String sourceOwner(String text,C oldText,C newText){return receiver(text).replace(argument("a",oldText),argument("b",newText));}
 static <Character extends CharSequence> String shadowBinder(String text,Character oldText,Character newText){return receiver(text).replace(argument("a",oldText),argument("b",newText));}
 public static void main(String[] args){
  int seed=Integer.parseInt(args[0]);char oldChar=(char)('a'+seed%26),newChar=(char)('A'+seed%26);
  String text=new String(new char[]{oldChar,(char)0xd800,0,(char)0xdc00,oldChar});
  synchronized(lock){
   show("bound",text,oldChar,newChar,true);show("direct",text,oldChar,newChar,false);
   show("same",text,oldChar,oldChar,true);show("surrogate",text,(char)0xd800,(char)0xdc00,true);
   show("old-null",text,null,newChar,true);show("new-null",text,oldChar,null,true);show("receiver-null",null,oldChar,newChar,true);
   show("both-null",null,null,null,true);
   failSecond=true;show("old-null-abrupt-second",text,null,newChar,true);show("second-abrupt",text,oldChar,newChar,true);failSecond=false;
   trace="";String changed=sourceOwner("aba",new Character("T","a"),new Character("U","z"));System.out.println("source-owner:"+changed+":"+trace);
   trace="";changed=shadowBinder("aba",new Character("T","a"),new Character("U","z"));System.out.println("shadow-binder:"+changed+":"+trace);
  }
 }
}
