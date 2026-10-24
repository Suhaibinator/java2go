package prereq;
import java.text.ParseException;
public class ConstructorProbe {
 static String trace="";
 static String message(String s){trace+="m";return s;}
 static int offset(){trace+="o";return 19;}
 static String units(String s){if(s==null)return "null";String r="";for(int i=0;i<s.length();i++){if(i>0)r+=",";r+=(int)s.charAt(i);}return r;}
 static void record(Throwable e,String message){Throwable cause=new IllegalStateException("cause");boolean empty=e.getCause()==null;boolean same=e.initCause(cause)==e;boolean linked=e.getCause()==cause;boolean blocked=false;try{e.initCause(null);}catch(IllegalStateException expected){blocked=expected.getCause()==e;}System.out.println(e.getClass().getSimpleName()+":"+(e.getMessage()==message)+":"+units(e.getMessage())+":"+empty+":"+same+":"+linked+":"+blocked);}
 static <T extends String> Throwable number(T text){return new NumberFormatException(text);}
 public static void main(String[] args){
  String[] messages={null,"",new String(new char[]{'x',(char)0xd800,0,(char)0xdc00}),new String(new char[]{(char)0xd800,(char)0xdc00})};
  for(String text:messages){record(new IndexOutOfBoundsException(text),text);record(new NumberFormatException(text),text);record(new ParseException(text,19),text);}
  record(new IndexOutOfBoundsException(),null);record(new NumberFormatException(),null);String bound=new String("bound");record(number(bound),bound);
  trace="";ParseException parsed=new ParseException(message(messages[2]),offset());System.out.println("evaluation:"+trace+":"+parsed.getErrorOffset()+":"+(parsed.getMessage()==messages[2]));
 }
}
