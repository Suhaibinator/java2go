package enumconsumer;
import enumprovider.Owner;
import static enumprovider.Owner.values;
public class NamespaceControl {
 static int calls;static Owner qualifier(){calls++;return null;}
 public static void main(String[] args){
  System.out.println("owner="+values().length+":"+Owner.values()[1].name()+":"+enumprovider.Owner.values()[0].name()+":"+Owner.implicit());
  System.out.println("names="+Owner.OwnerValues()+":"+Owner.OwnerValuesJava2goExecution());
  System.out.println("qualified="+qualifier().values().length+":"+calls+":"+enumprovider.Outer.Member.values().length);
  System.out.println("fresh="+(values()!=Owner.values()));
 }
}
