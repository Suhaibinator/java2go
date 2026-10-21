package classproof;
public class CanonicalClassStorageOracle {
 public static class Base { public Base(){} }
 public static class Child extends Base { public Child(){} }
 static class Slot<T> {
  Class<? super T> raw; Class<T> exact; Class<T>[] entries;
  Slot(Class<? super T> raw,Class<T> exact,Class<T>[] entries){this.raw=raw;this.exact=exact;this.entries=entries;}
  Class<? super T> get(){return raw;}void put(Class<? super T> next){raw=next;}
  Class<T> exact(){return exact;} Class<T>[] entries(){return entries;}
  <T> Class<T> shadow(Class<T> token){return token;}
 }
 static class CapturedOuter<T> {
  Class<T> token;CapturedOuter(Class<T> token){this.token=token;}
  class Capture<U> {Class<U> own;Capture(Class<U> own){this.own=own;}Class<T> outer(){return CapturedOuter.this.token;}Class<U> own(){return own;}<T>Class<T> shadow(Class<T> value){return value;}}
 }
 static int effects;static Object wrong(){effects++;return "bad";}
 static ThreadLocal<String> caller=new ThreadLocal<>();
 static Class<?> callback(Class<?> token){System.out.println("execution="+caller.get()+":"+Thread.holdsLock(token));return token;}
 @SuppressWarnings({"unchecked","rawtypes"})
 public static void main(String[] args)throws Exception {
  Class<Child>[] storage=(Class<Child>[])new Class<?>[]{Child.class,null};
  Slot<Child> slot=new Slot<>(Base.class,Child.class,storage);
  System.out.println("fields="+(slot.get()==Base.class)+":"+(slot.exact()==Child.class)+":"+(slot.entries()==storage));
  slot.put(Object.class);System.out.println("mutation="+(slot.get()==Object.class));
  Class<Integer> polluted=(Class<Integer>)(Class<?>)Child.class;
  System.out.println("argument-erasure="+((Object)polluted==Child.class)+":"+(polluted.getClass()==Class.class));
  System.out.println("nominal="+Class.class.isInstance(slot.exact())+":"+Child.class.isInstance(new Child())+":"+!Child.class.isInstance(slot.exact()));
  Object[] alias=storage;alias[0]=Base.class;
  System.out.println("array="+((Object)storage[0]==Base.class)+":"+(storage.getClass().getComponentType()==Class.class));
  try{alias[1]=wrong();System.out.println("store=unexpected");}catch(ArrayStoreException expected){System.out.println("store="+effects+":"+(storage[1]==null));}
  Class<Child>[] copy=storage.clone();System.out.println("clone="+(copy!=storage)+":"+((Object)copy[0]==Base.class));
  Class<?>[][] matrix=new Class<?>[][]{storage};System.out.println("rank2="+(matrix[0]==storage)+":"+(matrix.getClass().getComponentType()==Class[].class));
  Object reference=Child.class;Class<?> checked=(Class<?>)reference;
  System.out.println("cast="+(checked==Child.class)+":"+((Class<?>)(Object)null==null));
  try{Class<?> bad=(Class<?>)(Object)"bad";System.out.println("cast-bad=unexpected:"+bad);}catch(ClassCastException expected){System.out.println("cast-bad=ClassCastException");}
  System.out.println("method="+(slot.shadow(Integer.class)==Integer.class)+":"+(slot.shadow(String.class)==String.class));
  CapturedOuter<Child> outer=new CapturedOuter<>(Child.class);CapturedOuter<Child>.Capture<String> capture=outer.new Capture<>(String.class);
  System.out.println("capture="+(capture.outer()==Child.class)+":"+(capture.own()==String.class)+":"+(capture.shadow(Integer.class)==Integer.class));
  Child created=slot.exact().getDeclaredConstructor().newInstance();System.out.println("typed-result="+(created.getClass()==Child.class));
  caller.set("caller");synchronized(Child.class){System.out.println("callback="+(callback(Child.class)==Child.class));}
  System.out.println("literal="+(int.class==int.class)+":"+(Child[].class.getComponentType()==Child.class)+":"+!int.class.isInstance(created));
 }
}
