package rawfn;
import java.util.function.Function;
class Object{int code=7;}
class java{static class lang{static class Object{int code=9;}}}
public class Main{
 static int finals;
 public static void main(String[]args){
  Object first=new Object();java.lang.Object second=new java.lang.Object();
  Function one=key->{try{return first;}finally{finals++;}};
  Function two=key->second;
  Function nil=key->null;
  System.out.println((one.apply(null)==first)+":"+(two.apply(null)==second)+":"+(nil.apply(null)==null)+":"+first.code+":"+second.code+":"+finals);
 }
}
