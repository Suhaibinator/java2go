import java.lang.reflect.*;
import java.util.*;
public final class GenericParents {
  private static String quote(String value) { return "\""+value.replace("\\","\\\\").replace("\"","\\\"")+"\""; }
  private static String type(Type t) {
    if (t instanceof TypeVariable<?> v) {
      if (!(v.getGenericDeclaration() instanceof Class<?> owner)) throw new IllegalStateException("non-class parameter declaration");
      TypeVariable<?>[] parameters=owner.getTypeParameters();
      for(int i=0;i<parameters.length;i++) if(parameters[i].equals(v)) return "{\"kind\":\"parameter\",\"owner\":"+quote(owner.getName())+",\"index\":"+i+",\"name\":"+quote(v.getName())+"}";
      throw new IllegalStateException("unowned parameter");
    }
    if(t instanceof Class<?> c) {if(c.isArray()) return "{\"kind\":\"array\",\"component\":"+type(c.getComponentType())+"}";return "{\"kind\":\"nominal\",\"owner\":"+quote(c.getName())+",\"arguments\":[]}";}
    if(t instanceof ParameterizedType p) {
      if(!(p.getRawType() instanceof Class<?> raw) || p.getOwnerType()!=null) throw new IllegalStateException("unsupported member parameterized type");
      List<String> args=new ArrayList<>();for(Type a:p.getActualTypeArguments())args.add(type(a));
      return "{\"kind\":\"nominal\",\"owner\":"+quote(raw.getName())+",\"arguments\":["+String.join(",",args)+"]}";
    }
    throw new IllegalStateException("unsupported generic fact "+t.getTypeName());
  }
  public static void main(String[] args) throws Exception {
    String[] names={"java.lang.Iterable","java.util.AbstractCollection","java.util.ArrayDeque","java.util.Collection","java.util.Deque","java.util.List","java.util.Queue","java.util.SequencedCollection"};
    List<String> records=new ArrayList<>();
    for(String name:names) {
      Class<?> c=Class.forName(name,false,ClassLoader.getPlatformClassLoader());
      if(!c.getModule().getName().equals("java.base") || c.getClassLoader()!=null)throw new IllegalStateException("unexpected class provenance");
      List<String> params=new ArrayList<>(),parents=new ArrayList<>();
      for(TypeVariable<?> p:c.getTypeParameters())params.add(type(p));
      Type superclass=c.getGenericSuperclass();if(superclass!=null)parents.add(type(superclass));
      for(Type parent:c.getGenericInterfaces())parents.add(type(parent));
      records.add("{\"owner\":"+quote(name)+",\"parameters\":["+String.join(",",params)+"],\"parents\":["+String.join(",",parents)+"]}");
    }
    System.out.println("["+String.join(",",records)+"]");
  }
}
