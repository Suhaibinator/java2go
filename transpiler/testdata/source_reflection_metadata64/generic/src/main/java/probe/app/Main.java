package probe.app;
import probe.model.*;
import java.lang.reflect.*;
import java.util.*;
public final class Main {
 public static void main(String[] args)throws Exception{
  int seed=Integer.parseInt(args[0]);Order state=new Order();state.state="seed-"+seed;
  Base<List<Order>> anonymous=new Base<List<Order>>(){};anonymous.state=new ArrayList<>();anonymous.state.add(state);
  ParameterizedType inherited=(ParameterizedType)Order.class.getGenericSuperclass();
  ParameterizedType captured=(ParameterizedType)anonymous.getClass().getGenericSuperclass();
  ParameterizedType list=(ParameterizedType)captured.getActualTypeArguments()[0];
  System.out.println("inherited|"+inherited.getTypeName()+"|"+(inherited.getRawType()==Base.class)+"|"+(inherited.getActualTypeArguments()[0]==String.class)+"|"+(inherited.getOwnerType()==null));
  System.out.println("anonymous|"+captured.getTypeName()+"|"+(captured.getRawType()==Base.class)+"|"+(list.getRawType()==List.class)+"|"+(list.getActualTypeArguments()[0]==Order.class));
  for(Field field:Order.class.getDeclaredFields())System.out.println("field|"+field.getName()+"|"+field.getGenericType().getTypeName());
  TypeVariable<?> variable=Base.class.getTypeParameters()[0];
  System.out.println("variable|"+variable.getName()+"|"+(variable.getGenericDeclaration()==Base.class)+"|"+(variable.getBounds()[0]==Object.class));
  System.out.println("ordinary|"+(Object.class.getGenericSuperclass()==null)+"|"+(String.class.getGenericSuperclass()==Object.class));
  anonymous.state.get(0).state+="-mutated";
  System.out.println("state|"+state.state+"|"+anonymous.state.size());
 }
}
