package probe.model;
public final class State {
 private static boolean reject;
 public static IllegalArgumentException failure;
 private int revision=1;
 private State(){if(reject){reject=false;failure=new IllegalArgumentException("constructor rejected");throw failure;}}
 public static void rejectNext(){reject=true;}
 public void apply(int delta){revision+=delta;}
 public int revision(){return revision;}
}
