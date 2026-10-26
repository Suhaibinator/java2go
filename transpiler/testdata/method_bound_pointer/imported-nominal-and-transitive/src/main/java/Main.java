import foreign.Base;
import foreign.Child;
class T {String mark(){return "nominal-T";}}
class T2 {String mark(){return "nominal-T2";}}
class Object {String mark(){return "nominal-Object";}}
class Host<T extends Base> {
 <T extends foreign.Base> String direct(T value){return value.voice()+":"+Main.nominal().mark()+":"+Main.nominal2().mark();}
 <Object extends foreign.Base> String objectShadow(Object value){return value.voice()+":"+Main.sourceObject().mark();}
 <B extends foreign.Base,U extends B> String chain(U value){return value.voice();}
}
public class Main {
 static T nominal(){return new T();}
 static T2 nominal2(){return new T2();}
 static Object sourceObject(){return new Object();}
 public static void main(String[] args){Host<Base> host=new Host<>();Child child=new Child();
 System.out.println(host.direct(child));
 System.out.println(host.objectShadow(child));
 System.out.println(host.chain(child));
 try{host.direct((Base)null);}catch(NullPointerException expected){System.out.println("nil");}
 }
}
