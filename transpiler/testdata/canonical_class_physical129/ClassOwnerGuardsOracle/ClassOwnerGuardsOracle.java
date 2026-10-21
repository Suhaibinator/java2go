package guards;
public class ClassOwnerGuardsOracle {
 static class Class<T>{T value;Class(T value){this.value=value;}T read(){return value;}}
 static class Base{String tag(){return "base";}}
 static <Class extends Base>java.lang.Class<Class> qualified(java.lang.Class<Class> token){return token;}
 static <Class>Class binder(Class value){return value;}
 public static void main(String[] args){
  Class<String> source=new Class<>("source");foreign.Class<String> other=new foreign.Class<>("foreign");
  System.out.println("sources="+source.read()+":"+other.read());
  System.out.println("identities="+((Object)source!=(Object)other)+":"+(source.getClass()==Class.class)+":"+(other.getClass()==foreign.Class.class));
  System.out.println("canonical="+(qualified(Base.class)==Base.class)+":"+(java.lang.Class.class.isInstance(Base.class)));
  Base base=new Base();System.out.println("binder="+(binder(base)==base)+":"+binder(base).tag());
  class Class<T>{T value;Class(T value){this.value=value;}T read(){return value;}}
  Class<Integer> local=new Class<>(7);System.out.println("local="+local.read());
 }
}
