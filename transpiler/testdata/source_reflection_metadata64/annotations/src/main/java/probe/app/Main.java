package probe.app;
import probe.model.Payload;
import com.google.gson.annotations.SerializedName;
import java.lang.reflect.Field;
public final class Main{
 public static void main(String[] args)throws Exception{
  int seed=Integer.parseInt(args[0]);Payload data=new Payload("order-"+seed);
  StringBuilder export=new StringBuilder();
  for(Field field:Payload.class.getDeclaredFields()){
   SerializedName name=field.getAnnotation(SerializedName.class);
   String[] aliases=name.alternate();StringBuilder list=new StringBuilder();
   for(String alias:aliases){if(list.length()>0)list.append(',');list.append(alias);}
   if(aliases.length>0)aliases[0]="changed";
   String[] again=name.alternate();
   field.setAccessible(true);
   export.append(name.value()).append('=').append(field.get(data)).append(';');
   System.out.println("annotation|"+field.getName()+"|"+name.annotationType().getName()+"|"+name.value()+"|"+list+"|"+(again.length==0?"empty":again[0]));
   if(again.length>0)field.set(data,"alias-"+seed);
  }
  System.out.println("export|"+export+"|"+data.identifier());
 }
}
