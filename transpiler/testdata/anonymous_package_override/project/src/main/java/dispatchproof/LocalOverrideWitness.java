package dispatchproof;
public class LocalOverrideWitness {
 static abstract class Gate {abstract int choose(Object value);int route(){return choose(null);}}
 static class Named extends Gate {int choose(Object value){return 12;}}
 public static void main(String[] args){
  class Local extends Gate {public int choose(Object value){return 11;}}
  Gate local=new Local();Gate named=new Named();
  System.out.println(local.route()+":"+named.route());
 }
}
