package local.app;
public class Main {
 static int initialized, constructed;
 static int touch(){initialized++;return 17;}
 static class Dormant {static int seed=touch();}
 public static void main(String[] args){
  Class<?> dormant=Dormant.class;
  Class<?> first;
  {
   class Local {Local(){constructed++;}}
   first=Local.class;
   Class<?> one=Local[].class;
   Class<?> two=Local[][].class;
   System.out.println("before:"+initialized+":"+constructed+":"+(first==Local.class)+":"+first.getName().startsWith("[")+":"+one.getName().startsWith("[")+":"+two.getName().startsWith("["));
   System.out.println("arrays:"+one.getName().equals("[L"+first.getName()+";")+":"+two.getName().equals("[[L"+first.getName()+";")+":"+(one==Local[].class));
   Local value=new Local();
   System.out.println("after:"+initialized+":"+constructed+":"+(value.getClass()==first));
  }
  {
   class Local {}
   System.out.println("shadow:"+(Local.class!=first)+":"+(Local[].class!=first)+":"+(Local.class==Local.class));
  }
  new Dormant();
  System.out.println("initialized:"+initialized+":"+(dormant==Dormant.class));
  Class<?> strings=String.class, objects=Object.class, ints=int[].class, references=Object[].class;
  System.out.println("nominal:"+(strings!=objects)+":"+(ints!=references));
 }
}
