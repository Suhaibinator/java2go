class T {}
class Base {} class Child extends Base {}
class Carrier<T> {
 class Cell<U> {}
 int implicit(Cell<Integer> value){return 1;} long implicit(Object value){return 2L;}
 int explicit(Carrier<T>.Cell<Integer> value){return 3;} long explicit(Object value){return 4L;}
 <T extends Base> int shadow(Cell<T> value){return 5;} long shadow(Object value){return 6L;}
}
class Probe {
 static String run(Carrier<String> owner,Carrier<String>.Cell<Integer> same,Carrier<Integer>.Cell<Integer> other,Carrier<String>.Cell<Child> sameBound,Carrier<Integer>.Cell<Child> otherBound){
  return owner.implicit(same)+":"+owner.implicit(other)+":"+owner.explicit(same)+":"+owner.explicit(other)+":"+owner.shadow(sameBound)+":"+owner.shadow(otherBound);
 }
}