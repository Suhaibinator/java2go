package flow.api;
import workflow.state.Ledger;
public final class Factories {
public static Action sam(final Ledger ledger, final int seed) {
        int pass = 0;
        route: do {
            pass++;
            if (pass < 2) continue route;
            try {
                synchronized (ledger.lock) {
                    return new Action() {
public int read(boolean fail) {
                    try {
                        synchronized (ledger.lock) {
                            int total = seed;
                            route: do {
                                try {
                                    ledger.touch(); ledger.record("R"); total++;
                                    if (total == seed + 1) continue route;
                                    if (total == seed + 2) break route;
                                } finally { ledger.finish("LF"); }
                            } while (total < seed + 3);
                            if (fail) throw ledger.fault;
                            return total;
                        }
                    } finally { ledger.finish("MF"); }
                }
};
                }
            } finally { ledger.finish("CF"); }
        } while (pass < 3);
        return null;
    }
public static Action structured(final Ledger ledger, final int seed) {
        int pass = 0;
        route: do {
            pass++;
            if (pass < 2) continue route;
            try {
                synchronized (ledger.lock) {
                    return new Action() {
private final int marker = seed;
public int read(boolean fail) {
                    try {
                        synchronized (ledger.lock) {
                            int total = marker;
                            route: do {
                                try {
                                    ledger.touch(); ledger.record("R"); total++;
                                    if (total == seed + 1) continue route;
                                    if (total == seed + 2) break route;
                                } finally { ledger.finish("LF"); }
                            } while (total < seed + 3);
                            if (fail) throw ledger.fault;
                            return total;
                        }
                    } finally { ledger.finish("MF"); }
                }
};
                }
            } finally { ledger.finish("CF"); }
        } while (pass < 3);
        return null;
    }
public static Action local(final Ledger ledger, final int seed) {
        int pass = 0;
        route: do {
            pass++;
            if (pass < 2) continue route;
            try {
                synchronized (ledger.lock) {
                    class Work implements Action {
private final int marker = seed;
public int read(boolean fail) {
                    try {
                        synchronized (ledger.lock) {
                            int total = marker;
                            route: do {
                                try {
                                    ledger.touch(); ledger.record("R"); total++;
                                    if (total == seed + 1) continue route;
                                    if (total == seed + 2) break route;
                                } finally { ledger.finish("LF"); }
                            } while (total < seed + 3);
                            if (fail) throw ledger.fault;
                            return total;
                        }
                    } finally { ledger.finish("MF"); }
                }
}
return new Work();
                }
            } finally { ledger.finish("CF"); }
        } while (pass < 3);
        return null;
    }
}
