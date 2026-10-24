package origin.base;
import origin.real.Token;
public class Base<T> {
 public int calls;
 public Token echo(T value){calls++;return new Token(String.valueOf(value));}
}
