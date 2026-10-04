package clonefamily;
class Value { final int code; Value(int code){this.code=code;} }
abstract class Adapter<T> implements java.lang.Cloneable {
 final T value;Adapter(T value){Main.constructions++;this.value=value;}
 abstract T read();
 Object copy() throws CloneNotSupportedException{return super.clone();}
}
interface Factory {<S> Adapter<S> create();}
class Concrete extends Adapter<Value> {
 Concrete(Value value){super(value);}
 Value read(){return value;}
}
public class Main {
 static int constructions;
 public static void main(String[] args) throws CloneNotSupportedException {
  Value value=new Value(9);Concrete source=new Concrete(value);
  Adapter<Value> copy=(Adapter<Value>)source.copy();
  System.out.println((source!=copy)+":"+(copy instanceof Concrete)+":"+(source.getClass()==copy.getClass())+":"+(source.read()==copy.read())+":"+copy.read().code+":"+constructions);
 }
}