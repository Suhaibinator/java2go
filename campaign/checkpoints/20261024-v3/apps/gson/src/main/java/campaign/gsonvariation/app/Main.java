package campaign.gsonvariation.app;
import campaign.gsonvariation.workflow.Pipeline;
public final class Main {
    public static void main(String[] args) throws Exception {
        int seed = Integer.parseInt(args[0]);
        if (seed != 17 && seed != 41 && seed != 97) throw new IllegalArgumentException("unknown seed");
        System.out.println(new Pipeline(seed).run());
    }
}
