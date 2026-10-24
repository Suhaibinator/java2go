package flow.app;
import flow.api.Action;
import flow.api.Factories;
import workflow.state.Ledger;
public final class Main {
    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        Ledger ledger = new Ledger(seed);
        ledger.install(Integer.valueOf(seed + 1000));
        try {
            for (int kind = 0; kind < 3; kind++) {
                ledger.reset();
                Action action;
                synchronized (ledger.lock) {
                    if (kind == 0) action = Factories.sam(ledger, seed);
                    else if (kind == 1) action = Factories.structured(ledger, seed);
                    else action = Factories.local(ledger, seed);
                }
                System.out.println("capture:" + seed + ":" + kind + ":" + ledger.report());
                ledger.reset();
                int value = -1;
                try { synchronized (ledger.lock) { value = action.read(false); } }
                finally { ledger.finish("OF"); }
                System.out.println("return:" + seed + ":" + kind + ":" + value + ":" + ledger.report());
                ledger.reset();
                boolean same = false;
                try { synchronized (ledger.lock) { action.read(true); } }
                catch (Ledger.Fault failure) { same = failure == ledger.fault; ledger.touch(); ledger.record("C"); }
                finally { ledger.finish("OF"); }
                synchronized (ledger.lock) { ledger.touch(); ledger.record("AF"); }
                System.out.println("throw:" + seed + ":" + kind + ":" + same + ":" + ledger.fault.code + ":" + ledger.report());
            }
        } finally { ledger.finish("ALLF"); ledger.remove(); }
        System.out.println("cleanup:" + seed + ":" + ledger.absent() + ":" + ledger.report());
    }
}
