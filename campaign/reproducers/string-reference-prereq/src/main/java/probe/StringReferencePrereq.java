package probe;

import java.util.HashMap;
import java.util.Map;

/** Small prerequisite for String allocation identity and value-based map keys. */
public final class StringReferencePrereq {
  private static final String LITERAL = "alpha";
  private static int creations;
  private static String lastCreated;

  private StringReferencePrereq() {}

  private static String literalFromMethod() {
    return "alpha";
  }

  private static String fresh(String text) {
    creations++;
    lastCreated = new String(text);
    return lastCreated;
  }

  private static void require(boolean condition, String description) {
    if (!condition) throw new AssertionError(description);
  }

  public static void main(String[] args) {
    String literal = LITERAL;
    String methodLiteral = literalFromMethod();
    String first = fresh(literal);
    String second = fresh(methodLiteral);
    String alias = first;
    Object erasedAlias = alias;
    String chars = new String(new char[] {'a', 'l', 'p', 'h', 'a'});
    String absent = null;

    boolean literalIdentity = literal == methodLiteral;
    boolean copiesDistinct = first != second && first != literal && second != literal;
    boolean copiesEqual = first.equals(second) && first.equals(literal);
    boolean aliasIdentity = alias == first && erasedAlias == first && erasedAlias != second;
    boolean internIdentity = first.intern() == literal && second.intern() == literal
        && first != first.intern();
    boolean charsIdentity = chars != literal && chars.equals(literal)
        && chars.intern() == literal;
    boolean nullIdentity = absent == null && first != absent && null == (String) null;
    require(literalIdentity && copiesDistinct && copiesEqual && aliasIdentity
        && internIdentity && charsIdentity && nullIdentity, "String reference contract");

    Map<String, String> byValue = new HashMap<>();
    String firstPut = byValue.put(first, "one");
    String secondPut = byValue.put(second, "two");
    boolean mapContract = firstPut == null && "one".equals(secondPut)
        && byValue.size() == 1 && "two".equals(byValue.get(literal))
        && byValue.containsKey(chars);
    String nullPut = byValue.put(null, "null-value");
    boolean nullMapContract = nullPut == null && byValue.size() == 2
        && "null-value".equals(byValue.get(null));
    require(mapContract && nullMapContract, "value-based map key contract");

    int before = creations;
    boolean freshComparison = first == fresh("alpha");
    boolean callOnce = creations == before + 1 && lastCreated != first
        && lastCreated.equals(first);
    require(!freshComparison && callOnce, "allocation side effect timing");

    System.out.println("literal=" + literalIdentity);
    System.out.println("new=distinct:" + copiesDistinct + ":equal:" + copiesEqual);
    System.out.println("alias=" + aliasIdentity + ":intern=" + internIdentity
        + ":chars=" + charsIdentity + ":null=" + nullIdentity);
    System.out.println("map=" + mapContract + ":null-key=" + nullMapContract
        + ":size=" + byValue.size());
    System.out.println("effect=created:" + creations + ":once:" + callOnce
        + ":fresh-compare:" + freshComparison);
  }
}
