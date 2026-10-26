package sourceclassstoragenested;

import java.util.Date;
import java.sql.Timestamp;

final class Shell<T> {
static abstract class Kind<T extends Date> {
  private final Class<T> type;
  Deferred<T> back;
  Kind(Class<T> type) { this.type=type; }
  final Class<T> type() { return type; }
  abstract T read();
  abstract void write(T value);
}
}
final class ValueKind<T extends Date> extends Shell.Kind<T> {
  private T value;
  ValueKind(Class<T> type,T value) { super(type);this.value=value; }
  T read() { return value; }
  void write(T value) { this.value=value; }
}
abstract class Carrier<T> { abstract T read();abstract void write(T value); }
final class Deferred<T extends Date> extends Carrier<T> {
  private final Shell.Kind<T> kind;
  Deferred(Shell.Kind<T> kind) { this.kind=kind; }
  T read() { return kind.read(); }
  void write(T value) { kind.write(value); }
  Class<T> type() { return kind.type(); }
}
interface Factory { <M extends Date> Carrier<M> create(Class<M> type,M value); }
final class Maker implements Factory {
  public <M extends Date> Carrier<M> create(Class<M> type,M value) { return new Deferred<M>(new Shell.Kind<M>(type) { private M stored=value; M read(){return stored;} void write(M next){stored=next;} }); }
}
@SuppressWarnings({"unchecked","rawtypes"})
public class Main {
  public static void main(String[] args) {
    Timestamp value=new Timestamp(5678);value.setNanos(678000000);
    Factory factory=new Maker();Carrier<Timestamp> typed=factory.create(Timestamp.class,value);
    boolean identity=typed.read()==value;
    boolean metadata=((Deferred<Timestamp>)typed).type()==Timestamp.class;
    long millis=typed.read().getTime();int nanos=typed.read().getNanos();
    Carrier raw=typed;
    boolean checkedBeforeMutation=false;
    try { raw.write("wrong"); } catch(ClassCastException expected) { checkedBeforeMutation=raw.read()==value; }
    Date polluted=new Date(9012);raw.write(polluted);
    boolean stored=raw.read()==polluted;boolean delayed=false;
    try { Timestamp invalid=typed.read(); } catch(ClassCastException expected) { delayed=true; }
    raw.write(null);boolean nil=typed.read()==null;
    System.out.println(identity+"|"+metadata+"|"+millis+"|"+nanos+"|"+checkedBeforeMutation+"|"+stored+"|"+delayed+"|"+nil);
  }
}
