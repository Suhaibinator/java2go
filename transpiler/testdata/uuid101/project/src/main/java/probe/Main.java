package probe;
import java.util.UUID;
import support.Effects;
public class Main {
 static String units(String value) {
  if(value==null)return "null";
  String result="";
  for(int i=0;i<value.length();i++)result=result+Integer.toHexString(value.charAt(i))+",";
  return result;
 }
 static void error(String label,RuntimeException error) {
  System.out.println(label+"="+error.getClass().getName()+":"+units(error.getMessage())+":"+(error.getCause()==null));
 }
 static void parse(String text) {
  try {UUID u=UUID.fromString(Effects.text(text));System.out.println("parse="+units(text)+":"+u.toString()+":"+u.getMostSignificantBits()+":"+u.getLeastSignificantBits()+":"+u.hashCode());}
  catch(RuntimeException error){error("parse-"+units(text),error);}
 }
 public static void main(String[] args) {
  long seed=Long.parseLong(args[0]);
  UUID a=new UUID(seed*123456789L,-seed*987654321L);
  UUID b=UUID.fromString(a.toString());
  Object object=b;
  System.out.println("bits="+a.toString()+":"+a.getMostSignificantBits()+":"+a.getLeastSignificantBits()+":"+a.hashCode()+":"+a.equals(object)+":"+(a==b)+":"+a.compareTo(b));
  System.out.println("order="+new UUID(Long.MIN_VALUE,0L).compareTo(new UUID(Long.MAX_VALUE,0L))+":"+new UUID(0L,Long.MIN_VALUE).compareTo(new UUID(0L,Long.MAX_VALUE)));
  System.out.println("nominal="+(object instanceof UUID)+":"+(object instanceof java.io.Serializable)+":"+(object instanceof Comparable)+":"+UUID.class.getName());
  byte[] bytes=new byte[]{(byte)seed,(byte)(seed>>8),(byte)-1,(byte)0,(byte)1};
  UUID named=UUID.nameUUIDFromBytes(Effects.bytes(bytes));
  bytes[0]=(byte)0;
  System.out.println("name="+named.toString()+":"+named.hashCode()+":"+UUID.nameUUIDFromBytes(new byte[0]).toString());
  String[] cases=new String[]{"1-1-1-1-1","00000001-0001-0001-0001-000000000001","ABCDEF12-3456-789A-BCDE-F0123456789A","+1-+1-+1-+1-+1","100000000-1-1-1-1","7fffffffffffffff-1-1-1-1","\uff11-\uff21-\uff41-\uff11-\uff11","","x","1-2-3-4","-1-1-1-1-1","1-1-1-1-1-1","1--1-1-1","1-1-1-1-","+-1-1-1-1","g-1-1-1-1","8000000000000000-1-1-1-1","1-1-1-1-9223372036854775808","\ud800-1-1-1-1","00000000-0000-0000-0000-0000000000000",null};
  for(String text:cases)parse(text);
  try {UUID.nameUUIDFromBytes(Effects.bytes(null));}catch(RuntimeException error){error("name-null",error);}
  try {a.compareTo(Effects.value(null));}catch(RuntimeException error){error("compare-null",error);}
  UUID absent=null;
  try {absent.compareTo(Effects.abrupt());}catch(RuntimeException error){error("argument-before-null",error);}
  System.out.println("equals="+a.equals(null)+":"+a.equals("foreign")+":"+object.equals(a)+":"+object.hashCode());
  System.out.println("shadow="+shadow.UUID.fromString("x").toString()+":"+Effects.count);
 }
}
