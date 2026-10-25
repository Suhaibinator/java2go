package audit;
import model.Value;
public class Audit {
 public static Value current;
 public static int calls;
 public static int number(String tag,int n){System.out.println(tag);return n;}
 public static String label(String tag,String s){System.out.println(tag);return s;}
 public static void before(Value v){calls++;current=v;System.out.println("before="+v.number()+":"+(v.label()==null));}
 public static void same(String tag,Value v){System.out.println(tag+"-same="+(v==current));}
}
