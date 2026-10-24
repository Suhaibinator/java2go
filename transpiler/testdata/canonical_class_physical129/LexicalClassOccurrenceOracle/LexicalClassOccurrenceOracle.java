package lexical;
public class LexicalClassOccurrenceOracle<T> {
 Class<T> memberBeforeLocal;
 Class<T> header(Class<T> token) { class Class<U>{U value;} Class<String> local=new Class<>(); local.value="header-local"; System.out.println(local.value); return token; }
 void ordinary(T value, Class<T> token) {
  Class<T> before=token; System.out.println("before="+(before==token));
  { class Class<U>{U item;Class(U item){this.item=item;}} Class<T> inside=new Class<>(value); System.out.println("inside="+(inside.item==value)); }
  Class<T> afterBlock=token; System.out.println("after-block="+(afterBlock==token));
  { class Class<U>{U item;Class(U item){this.item=item;}} Class<T> sibling=new Class<>(value); System.out.println("sibling="+(sibling.item==value)); }
  class Class<U>{U item;Class(U item){this.item=item;}Class<U> self(){return this;}}
  Class<T> local=new Class<>(value); System.out.println("local="+(local.item==value)+":"+(local.self()==local));
  { Class<T> nested=local; System.out.println("nested="+(nested==local)); }
  class Capture { Class<T> captured=local; Object get(){return captured;} }
  Capture capture=new Capture(); System.out.println("capture="+(capture.get()==local));
  Runnable anonymous=new Runnable(){Class<T> captured=local; public void run(){System.out.println("anonymous="+(captured==local));}};anonymous.run();
  java.lang.Class<T> qualified=token;System.out.println("qualified="+(qualified==token));
 }
 void otherMethod(Class<T> token){System.out.println("other-method="+(token==memberBeforeLocal));}
 static void switches(){
  switch(1){case 1: class Class<U>{U value;} Class<String> local=new Class<>();local.value="switch-local";System.out.println(local.value);break;case 2: Class<String> canonical=String.class;System.out.println(canonical==String.class);break;default:break;}
 }
 public static void main(String[] args){LexicalClassOccurrenceOracle<String> app=new LexicalClassOccurrenceOracle<>();app.memberBeforeLocal=String.class;app.ordinary("value",String.class);System.out.println("header="+(app.header(String.class)==String.class));app.otherMethod(String.class);switches();}
}
