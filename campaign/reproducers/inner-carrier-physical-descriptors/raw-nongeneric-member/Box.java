class Box<T> {
 class Member {}
 int raw(Box.Member value){return 1;} long raw(Object value){return 2L;}
 int implicit(Member value){return 3;} long implicit(Object value){return 4L;}
 int parameterized(Box<T>.Member value){return 5;} long parameterized(Object value){return 6L;}
}
class Probe {
 static String run(Box<String> owner,Box<String>.Member same,Box<Integer>.Member other){
 return owner.raw(same)+":"+owner.raw(other)+":"+owner.implicit(same)+":"+owner.implicit(other)+":"+owner.parameterized(same)+":"+owner.parameterized(other);
 }
}