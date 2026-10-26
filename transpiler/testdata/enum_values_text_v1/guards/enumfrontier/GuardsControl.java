package enumfrontier;
public class GuardsControl {
 static class Object {public java.lang.String toString(){return "source-object";}}
 static class Enum {public java.lang.String toString(){return "source-enum";}static int[] values(){return new int[]{7};}}
 static class String {public java.lang.String toString(){return "source-string";}}
 static class Plain {public java.lang.String toString(){return "plain";}}
 enum Real {A;java.lang.String toString(int n){return "real-overload";}static int values(int n){return n+1;} }
 static <Enum extends Plain>java.lang.String bound(Enum value){return value.toString();}
 static <Object extends Plain>java.lang.String objectBound(Object value){return value.toString();}
 static java.lang.String read(java.lang.Enum<?> value){return value.toString();}
 public static void main(java.lang.String[] args){
  System.out.println("source="+new Object().toString()+":"+new Enum().toString()+":"+new String().toString()+":"+Enum.values()[0]);
  System.out.println("binders="+bound(new Plain())+":"+objectBound(new Plain()));
  System.out.println("real="+Real.A.toString()+":"+Real.A.toString(4)+":"+Real.values(9)+":"+read(Real.A));
 }
}
