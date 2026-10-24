package enumfrontier;
public class MessagesControl {
 static String trace="";
 enum Normal {A;Normal(){try{Normal.values();trace+="unexpected;";}catch(NullPointerException expected){trace+=expected.getMessage()+";";}}}
 enum BackingCollision {A;static int $VALUES=1;BackingCollision(){try{BackingCollision.values();trace+="unexpected;";}catch(NullPointerException expected){trace+=expected.getMessage()+";";}}}
 public static void main(String[] args){
  Normal.values();BackingCollision.values();System.out.println("messages="+trace);
  System.out.println("cleanup="+Normal.values().length+":"+BackingCollision.values().length+":"+BackingCollision.$VALUES);
 }
}
