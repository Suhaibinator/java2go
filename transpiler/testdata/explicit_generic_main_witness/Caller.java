public class Caller {
  static int argumentCalls;
  static String trace = "";
  static String[] retainedArguments;
  static String[] arguments(String value) {
    argumentCalls++;
    trace += "arguments,";
    String[] result = new String[]{new String(value)};
    retainedArguments = result;
    return result;
  }
  public static void main(String[] processArgs) {
    String seed = processArgs[0];
    try {
      trace += "before,";
      Main.<Root, Root>main(arguments(seed));
      trace += "between,";
      System.out.println("first=" + argumentCalls + ":" + Main.reads + ":" + (Main.retained == null) + ":" + (retainedArguments[0].equals(seed)) + ":" + trace);
      Main.<Root, Root>main(arguments(seed));
      trace += "after,";
      System.out.println("second=" + argumentCalls + ":" + Main.reads + ":" + (Main.retained == null) + ":" + (retainedArguments.length == 1) + ":" + trace);
    } finally {
      retainedArguments = null;
      trace += "cleanup,";
    }
    System.out.println("caller.final=" + (retainedArguments == null) + ":" + argumentCalls + ":" + Main.reads + ":" + trace);
  }
}
