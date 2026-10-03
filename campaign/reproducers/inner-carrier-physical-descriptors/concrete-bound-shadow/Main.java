public class Main {
 public static void main(String[] args) {
  Carrier<String> owner = new Carrier<String>();
  Carrier<Integer> otherOwner = new Carrier<Integer>();
  Carrier<String>.Cell<Integer> same = owner.new Cell<Integer>();
  Carrier<Integer>.Cell<Integer> other = otherOwner.new Cell<Integer>();
  Carrier<String>.Cell<Child> sameBound = owner.new Cell<Child>();
  Carrier<Integer>.Cell<Child> otherBound = otherOwner.new Cell<Child>();
  System.out.print(Probe.run(owner,same,other,sameBound,otherBound));
 }
}