package shadow;
import static java.lang.reflect.Modifier.isStatic;
class Modifier {static boolean isStatic(int value){return value==123;}}
class Facade {boolean isStatic(int value){return value==123;}}
public class ShadowProbe {
 static boolean isStatic(int value){return value==123;}
 public static void main(String[] args){
  Facade Modifier=new Facade();
  System.out.println("value="+Modifier.isStatic(8)+":"+Modifier.isStatic(123));
  System.out.println("source="+shadow.Modifier.isStatic(8)+":"+shadow.Modifier.isStatic(123));
  System.out.println("qualified="+java.lang.reflect.Modifier.isStatic(8)+":"+java.lang.reflect.Modifier.isStatic(123));
  System.out.println("lexical="+isStatic(8)+":"+isStatic(123));
 }
}
