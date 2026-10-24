package receiverorder;
class State {static String events="";static boolean receiverFails,argumentFails;}
class Base {String voice(Base argument){State.events+="B";return "base";}}
class Child extends Base {@Override String voice(Base argument){State.events+="V";return "child";}}
class Host<T extends Base> {
 <T extends Base> T once(T value){State.events+="R";if(State.receiverFails)throw new IllegalStateException("receiver");return value;}
 Base effect(){State.events+="A";if(State.argumentFails)throw new IllegalStateException("argument");return null;}
 <T extends Base> String run(T value){try{return once(value).voice(effect());}finally{State.events+="F";}}
}
public class Main {
 public static void main(String[] args){Host<Base> host=new Host<>();Child child=new Child();
 System.out.println(host.run(child)+":"+State.events);
 State.events="";try{host.run((Base)null);}catch(NullPointerException expected){System.out.println("nil:"+State.events);}
 State.events="";State.argumentFails=true;try{host.run((Base)null);}catch(IllegalStateException expected){System.out.println(expected.getMessage()+":"+State.events);}
 State.events="";State.argumentFails=false;State.receiverFails=true;try{host.run(child);}catch(IllegalStateException expected){System.out.println(expected.getMessage()+":"+State.events);}
 }
}
