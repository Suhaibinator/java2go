package staticabi;
class Object{int code=99;}
class Child extends Object{}
class T2{int code=2;}
class T3{int code=3;}
class Owner<T>{
 static <T> T universal(java.lang.Object value){return (T)value;}
 static <T extends Object> T source(Object value){return (T)value;}
 static <T extends java.lang.Object> T canonical(java.lang.Object value){return (T)value;}
 static <U,T extends U> T dependent(java.lang.Object value){return (T)value;}
 static <T> staticabi.T2 second(T value){return new staticabi.T2();}
 static <T> staticabi.T3 third(T value){return new staticabi.T3();}
}
public class Main{
 public static void main(String[]args){
  java.lang.Object marker=Integer.valueOf(7);
  java.lang.Object universal=Owner.<String>universal(marker);
  Child child=new Child();Object source=Owner.<Child>source(child);
  java.lang.Object canonical=Owner.<String>canonical(marker);
  java.lang.Object dependent=Owner.<String,String>dependent(marker);
  Child nil=Owner.<Child>source(null);
  staticabi.T2 second=Owner.second("x");staticabi.T3 third=Owner.third("x");
  System.out.println((universal==marker)+":"+(source==child)+":"+(canonical==marker)+":"+(dependent==marker)+":"+(nil==null)+":"+second.code+":"+third.code);
 }
}
