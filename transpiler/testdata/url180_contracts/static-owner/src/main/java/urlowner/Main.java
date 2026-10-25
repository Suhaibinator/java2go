package urlowner;
class Endpoint {
 static int calls;
 void setURLStreamHandlerFactory(Object ignored){calls++;}
 Object of(Object first,Object second){calls+=10;return this;}
}
class Net {Endpoint URL=new Endpoint();}
class java {static Net net=new Net();}
public class Main {
 public static void main(String[] args){
  java.net.URL.setURLStreamHandlerFactory(null);
  Object result=java.net.URL.of(null,null);
  System.out.println(Endpoint.calls+":"+(result==java.net.URL));
 }
}
