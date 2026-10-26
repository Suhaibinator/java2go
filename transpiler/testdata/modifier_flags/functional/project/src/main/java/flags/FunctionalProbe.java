package flags;
import java.lang.reflect.Modifier;
import java.util.function.Predicate;
public class FunctionalProbe {
 interface ByteFlag {boolean test(byte value);}
 interface ShortFlag {boolean test(short value);}
 interface CharFlag {boolean test(char value);}
 public static void main(String[]args){
  Predicate<Integer> boxed=Modifier::isStatic;
  Predicate<Byte> boxedByte=Modifier::isStatic;
  Predicate<Short> boxedShort=Modifier::isStatic;
  Predicate<Character> boxedChar=Modifier::isStatic;
  ByteFlag narrowByte=Modifier::isStatic;
  ShortFlag narrowShort=Modifier::isStatic;
  CharFlag narrowChar=Modifier::isStatic;
  System.out.println("boxed="+boxed.test(8)+":"+boxedByte.test((byte)8)+":"+boxedShort.test((short)8)+":"+boxedChar.test((char)8));
  System.out.println("narrow="+narrowByte.test((byte)8)+":"+narrowShort.test((short)8)+":"+narrowChar.test((char)8));
  System.out.println("false="+boxed.test(Integer.MIN_VALUE)+":"+narrowByte.test((byte)0)+":"+narrowChar.test((char)0));
  try{boxed.test(null);System.out.println("null=false");}catch(NullPointerException expected){System.out.println("null=true");}
 }
}
