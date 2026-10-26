package metadatadefaults;
class java {static class lang {static class Object {}}}
public class Defaults<T> {
 static class Explicit<T extends java.lang.Object> extends java.lang.Object {}
 public static void main(String[] args) {
  System.out.println("implicit="+(Defaults.class.getTypeParameters()[0].getBounds()[0]==Object.class)+":"+(Defaults.class.getGenericSuperclass()==Object.class));
  System.out.println("explicit="+(Explicit.class.getTypeParameters()[0].getBounds()[0]==java.lang.Object.class)+":"+(Explicit.class.getGenericSuperclass()==java.lang.Object.class)+":"+(((Class<?>)java.lang.Object.class)!=Object.class));
 }
}
