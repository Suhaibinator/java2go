package enumfrontier;
public class ConstructorControl {
 static String literal="Cannot invoke \"[Lenumfrontier.ConstructorControl$$VALUES;.clone()\" because \"enumfrontier.ConstructorControl$$VALUES.$VALUES\" is null";
 static String observed;static boolean sameMessage;static boolean freshMessage;static boolean interned;
 enum $VALUES { A;
  $VALUES(){try{values();}catch(NullPointerException expected){observed=expected.getMessage();sameMessage=observed==expected.getMessage();freshMessage=observed!=literal;interned=observed.intern()==literal;}}
 }
 public static void main(String[] args){$VALUES.values();System.out.println("message="+observed);System.out.println("references="+sameMessage+":"+freshMessage+":"+interned);System.out.println("cleanup="+$VALUES.values().length);}
}
