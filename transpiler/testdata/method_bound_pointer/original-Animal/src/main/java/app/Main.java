package app;
import foreign.Animal;
class Child extends Animal {@Override public String voice(){return "child";} }
class Bound<Animal extends foreign.Animal> {
 int calls;
 Animal once(Animal value){calls++;return value;}
 String read(Animal value){return once(value).voice();}
 <Animal extends foreign.Animal> String shadow(Animal value){return value.voice();}
}
public class Main {
 static <T extends foreign.Animal> String witness(T value){return value.voice();}
 public static void main(String[] args){Bound<Animal> bound=new Bound<>();Child child=new Child();
 System.out.println(bound.read(child)+":"+bound.calls+":"+bound.shadow(child)+":"+witness(child));
 try{bound.read(null);}catch(NullPointerException expected){System.out.println("nil:"+bound.calls);}
 System.out.println(bound.read(child)+":"+bound.calls+":"+(child.getClass().getSuperclass()==foreign.Animal.class));
 }
}
