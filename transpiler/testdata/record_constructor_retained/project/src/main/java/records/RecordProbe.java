package records;
public class RecordProbe {
 public record Value(int number){public Value(){this(0);}}
 public static void main(String[] args)throws Exception {
  try {System.out.println("record="+Value.class.getDeclaredConstructor().getParameterCount());}
  catch(UnsupportedOperationException expected){System.out.println("record=unsupported");}
 }
}
