package prereq.generic.app;

import prereq.generic.domain.Entry;
import prereq.generic.ledger.Ledger;
import prereq.generic.workflow.Agent;

public final class Main {
    private Main() {}

    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        Agent<StringBuilder> agent = new Agent<>(new StringBuilder("agent-" + seed));
        Ledger<Integer> retail = new Ledger<Integer>(Integer.valueOf(seed), Integer.valueOf(seed));
        Ledger<Integer> reserve = agent.openReserve(Integer.valueOf(seed * 2));

        int primitive = retail.post(agent.primitive(3), agent.entry("primitive"));
        int boxed = retail.post(agent.boxed(4), agent.entry("boxed"));
        int wide = retail.post(agent.wide(5), agent.entry("wide"));
        int widened = retail.post((Number) agent.boxed(6), agent.entry("widened"));
        int forwarded = agent.forward(retail, agent.boxed(7), agent.entry("forwarded"));
        int reservePosted = reserve.post(agent.boxed(1), agent.entry("reserve"));

        Entry<StringBuilder> rejected = agent.entry("rejected");
        String beforeReject = retail.snapshot();
        boolean rejectedAsExpected = false;
        try {
            retail.post(agent.primitive(-2), rejected);
        } catch (IllegalArgumentException expected) {
            rejectedAsExpected = true;
        }

        System.out.println("seed=" + seed + ",routes=" + primitive + "," + boxed
                + "," + wide + "," + widened + "," + forwarded + "," + reservePosted);
        System.out.println("retail=" + retail.snapshot());
        System.out.println("reserve=" + reserve.snapshot());
        System.out.println("bound=" + agent.boundCheck(retail) + "," + agent.boundCheck(reserve));
        System.out.println("rejection=" + rejectedAsExpected + ",touches=" + rejected.touches()
                + ",before=" + beforeReject);
        System.out.println("evaluations=" + agent.evaluations());
    }
}
