package prereq.reflection.error;

/** Deliberately has no public (String, Throwable) constructor. */
public final class NoMatchingConstructor extends Exception {
  public NoMatchingConstructor() {
    super("only-default");
  }
}
