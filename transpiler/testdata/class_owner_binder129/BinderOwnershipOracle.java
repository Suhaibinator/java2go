package binderproof;
import foreign.Cell;
public class BinderOwnershipOracle {
 static class Box<T>{T value;Box(T value){this.value=value;}}
 static class Class<T>{T value;Class(T value){this.value=value;}}
 static class Base{String tag(){return "base";}}
 static class Holder<Box>{Box value;Holder(Box value){this.value=value;}Box echo(Box next){value=next;return value;}<Box>Box shadow(Box next){return next;}}
 static <Box>Box method(Box value){return value;}
 static <Class extends Base>Class bounded(Class value){return value;}
 static <Box>Box[] binderArray(Box[] value){return value;}
 static Object raw(Box value){return value;}
 static Object wildcard(Box<?> value){return value;}
 static Object sourceArray(Box[] value){return value;}
 static <Box>Object qualified(binderproof.BinderOwnershipOracle.Box<?> value){return value;}
 static <Cell>Cell importedBinder(Cell value){return value;}
 static Object importedRaw(Cell value){return value;}
 static Object importedWildcard(Cell<?> value){return value;}
 static <Class>java.lang.Class<Class> canonical(java.lang.Class<Class> token){return token;}
 public static void main(String[] args){
  String token="token";System.out.println("method="+(method(token)==token)+":"+method(token));
  Base base=new Base();System.out.println("bound="+(bounded(base)==base)+":"+bounded(base).tag());
  Holder<String> holder=new Holder<>(token);System.out.println("class="+(holder.echo(token)==token)+":"+holder.value);
  Integer number=7;System.out.println("shadow="+(holder.shadow(number)==number)+":"+holder.shadow(number));
  String[] values=new String[]{token};System.out.println("binder-array="+(binderArray(values)==values)+":"+(binderArray(values)[0]==token));
  Box<String> box=new Box<>(token);System.out.println("source="+(raw(box)==box)+":"+(wildcard(box)==box));
  Box<?>[] boxes=new Box<?>[]{box};System.out.println("source-array="+(sourceArray(boxes)==boxes)+":"+(boxes[0]==box));
  System.out.println("qualified="+(BinderOwnershipOracle.<String>qualified(box)==box));
  Cell<String> cell=new Cell<>(token);System.out.println("imports="+(importedBinder(token)==token)+":"+(importedRaw(cell)==cell)+":"+(importedWildcard(cell)==cell));
  System.out.println("canonical="+(canonical(String.class)==String.class));
 }
}
