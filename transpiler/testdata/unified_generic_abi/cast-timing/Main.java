public class Main {
 static int calls;
 static class Base {}
 static class Child extends Base {}
 static <T> T read(java.lang.Object value) { calls++; return (T)value; }
 static <T extends Base> T bounded(java.lang.Object value) { calls++; return (T)value; }
 public static void main(String[] args) {
  java.lang.Object marker=Integer.valueOf(7);
  java.lang.Object seen=Main.<String>read(marker);
  System.out.println("object:"+(seen==marker)+":"+calls);
  try {String text=Main.<String>read(marker);System.out.println("unexpected:"+text);}catch(ClassCastException e){System.out.println("cast:"+calls);}
  Main.<String>read(marker);System.out.println("discard:"+calls);
  String nil=Main.<String>read(null);System.out.println("null:"+(nil==null)+":"+calls);
  Child child=new Child();java.lang.Object base=Main.<Child>bounded(child);System.out.println("bound-object:"+(base==child)+":"+calls);
  Child exact=Main.<Child>bounded(child);System.out.println("bound-child:"+(exact==child)+":"+calls);
  java.lang.Object empty=Main.<Child>bounded(null);System.out.println("bound-null:"+(empty==null)+":"+calls);
  try{java.lang.Object wrong=Main.<Child>bounded(marker);System.out.println("unexpected-bound:"+wrong);}catch(ClassCastException e){System.out.println("bound-cast:"+calls);}
 }
}
