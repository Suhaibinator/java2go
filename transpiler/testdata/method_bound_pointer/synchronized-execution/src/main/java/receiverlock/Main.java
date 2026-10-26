package receiverlock;
class State {static String events="";static Object lock;static boolean fail;}
class Base {String voice(Base argument){State.events+="B"+Thread.holdsLock(State.lock);return "base";}}
class Child extends Base {@Override String voice(Base argument){State.events+="V"+Thread.holdsLock(State.lock);return "child";}}
class Host<T extends Base> {
 <U extends Base> U once(U value){State.events+="R"+Thread.holdsLock(this);return value;}
 Base effect(){State.events+="A"+Thread.holdsLock(this);if(State.fail)throw new IllegalStateException("argument");return null;}
 synchronized <U extends Base> String run(U value){try{return once(value).voice(effect());}finally{State.events+="F"+Thread.holdsLock(this);}}
}
public class Main {
 public static void main(String[] args){Host<Base> host=new Host<>();Child child=new Child();State.lock=host;
 synchronized(host){System.out.println(host.run(child)+":"+State.events+":"+Thread.holdsLock(host));}
 System.out.println("outside:"+Thread.holdsLock(host));
 State.events="";synchronized(host){try{host.run((Base)null);}catch(NullPointerException expected){System.out.println("nil:"+State.events+":"+Thread.holdsLock(host));}}
 System.out.println("after-nil:"+Thread.holdsLock(host));
 State.events="";State.fail=true;synchronized(host){try{host.run((Base)null);}catch(IllegalStateException expected){System.out.println(expected.getMessage()+":"+State.events+":"+Thread.holdsLock(host));}}
 System.out.println("after-abrupt:"+Thread.holdsLock(host));
 }
}
