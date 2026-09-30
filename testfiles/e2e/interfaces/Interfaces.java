interface Greeter {
    String greet(String who);

    default String shout(String who) {
        return greet(who) + "!";
    }
}

class Formal implements Greeter {
    public String greet(String who) {
        return "Good day, " + who;
    }
}

class Casual implements Greeter {
    public String greet(String who) {
        return "Hey " + who;
    }
}

public class Interfaces {
    public static void main(String[] args) {
        Greeter[] greeters = new Greeter[] { new Formal(), new Casual() };
        for (Greeter g : greeters) {
            System.out.println(g.greet("Sam"));
            System.out.println(g.shout("Sam"));
        }
    }
}
