class String {String(){} }
class Integer {java.lang.String value;Integer(java.lang.String value){this.value=value;}static Integer valueOf(java.lang.String value){return new Integer(value);}public java.lang.String toString(){return "source:"+value;}}
public class BoxedStringSourceBinders {
 static <Integer,String> java.lang.Integer binder(Integer ignored,String ignoredText){return new java.lang.Integer("７");}
 public static java.lang.String run(){Integer source=Integer.valueOf("owner");return source+":"+java.lang.Integer.valueOf("７")+":"+binder(new Integer("binder"),new String());}
 public static void main(java.lang.String[] args){System.out.print(run());}
}
