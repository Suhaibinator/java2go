public class ArithmeticMessageProbe {
 static int evaluated=0;static String message(String text){evaluated++;return text;}
 static String units(String text){if(text==null)return "null";String out="";for(int i=0;i<text.length();i++)out+=(int)text.charAt(i)+",";return out;}
 static <T extends String> ArithmeticException bounded(T text){return new ArithmeticException(message(text));}
 static String record(ArithmeticException value,String text){boolean initial=value.getCause()==null;RuntimeException cause=new RuntimeException("cause");boolean same=value.initCause(cause)==value;boolean blocked=false;try{value.initCause(null);}catch(IllegalStateException expected){blocked=true;}return (value.getMessage()==text)+":"+units(value.getMessage())+":"+units(value.toString())+":"+initial+":"+same+":"+(value.getCause()==cause)+":"+blocked;}
 public static String run(){String odd=new String(new char[]{'x',(char)0xd800,0,(char)0xdc00});String out=record(new ArithmeticException(),null)+"\n"+record(new ArithmeticException(null),null)+"\n"+record(new ArithmeticException(""),"")+"\n"+record(new ArithmeticException(message(odd)),odd)+"\n"+record(bounded(odd),odd);RuntimeException existing=new IllegalArgumentException(odd);NumberFormatException number=new NumberFormatException(odd);NullPointerException npe=new NullPointerException(odd);return out+"\n"+(existing.getMessage()==odd)+":"+(number.getMessage()==odd)+":"+(npe.getMessage()==odd)+":"+evaluated;}
 public static void main(String[] args){System.out.print(run());}
}
