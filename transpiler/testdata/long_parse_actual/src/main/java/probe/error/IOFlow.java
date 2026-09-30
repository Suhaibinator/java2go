package probe.error;
import probe.state.Journal;
public final class IOFlow {
    private static void observe(String label, java.io.IOException problem, String message, Throwable cause, boolean assignable, Signal replacement) {
        boolean before = problem.getCause() == cause;
        String result;
        try { result = "ok:" + (problem.initCause(replacement) == problem); }
        catch (RuntimeException failure) { result = failure.getClass().getName() + ":" + Journal.units(failure.getMessage()); }
        System.out.println("io:" + label + ":" + (problem.getMessage() == message) + ":" + Journal.units(problem.getMessage()) + ":" + before + ":" + result + ":" + (problem.getCause() == (assignable ? replacement : cause)));
    }
    public static void run(int seed) {
        String message = new String(new char[]{(char) ('a' + seed % 26), 0, '\ud800', 'x', '\udc00'});
        Signal signal = new Signal(seed);
        observe("child-empty", new IOChild(), null, null, true, signal);
        observe("child-string", new IOChild(message), message, null, true, signal);
        observe("child-both", new IOChild(message, signal), message, signal, false, signal);
        observe("child-null-cause", new IOChild((Throwable) null), null, null, false, signal);
        observe("empty", new java.io.IOException(), null, null, true, signal);
        observe("string", new java.io.IOException(message), message, null, true, signal);
        observe("null-string", new java.io.IOException((String) null), null, null, true, signal);
        observe("both", new java.io.IOException(message, signal), message, signal, false, signal);
        observe("null-message-cause", new java.io.IOException((String) null, signal), null, signal, false, signal);
        observe("message-null-cause", new java.io.IOException(message, (Throwable) null), message, null, false, signal);
        observe("both-null", new java.io.IOException((String) null, (Throwable) null), null, null, false, signal);
        observe("null-cause", new java.io.IOException((Throwable) null), null, null, false, signal);
        Journal journal = new Journal(); CallbackCause cause = new CallbackCause(journal, message, null);
        java.io.IOException wrapped;
        synchronized (journal.lock()) { journal.mark("enter"); wrapped = new java.io.IOException(cause); journal.mark("built"); }
        boolean released = !Thread.holdsLock(journal.lock());
        synchronized (journal.lock()) { journal.mark("reacquire"); }
        observe("cause", wrapped, message, cause, false, signal);
        System.out.println("callback:cause:" + cause.calls() + ":" + cause.held() + ":" + cause.sameThread() + ":" + released + ":" + journal.count() + ":" + journal.trace());
        Journal lazyJournal = new Journal(); CallbackCause lazy = new CallbackCause(lazyJournal, message, signal);
        java.io.IOException direct = new java.io.IOException(message, lazy);
        System.out.println("callback:lazy:" + lazy.calls() + ":" + (direct.getMessage() == message) + ":" + (direct.getCause() == lazy) + ":" + lazyJournal.count());
        Journal abrupt = new Journal(); CallbackCause throwing = new CallbackCause(abrupt, message, signal); boolean identity = false;
        try { synchronized (abrupt.lock()) { abrupt.mark("enter"); new java.io.IOException(throwing); abrupt.mark("unreachable"); } }
        catch (Signal failure) { identity = failure == signal; abrupt.mark("catch"); }
        boolean cleanup = !Thread.holdsLock(abrupt.lock());
        synchronized (abrupt.lock()) { abrupt.mark("reacquire"); }
        System.out.println("callback:throw:" + identity + ":" + throwing.calls() + ":" + throwing.held() + ":" + throwing.sameThread() + ":" + cleanup + ":" + abrupt.count() + ":" + abrupt.trace());
    }
}
