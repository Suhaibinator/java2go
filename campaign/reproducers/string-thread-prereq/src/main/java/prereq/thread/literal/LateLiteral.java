package prereq.thread.literal;

/** Called only after a worker interns equal char-array content. */
public final class LateLiteral {
  private LateLiteral() {}

  public static String value() {
    return "late-canonical";
  }
}
