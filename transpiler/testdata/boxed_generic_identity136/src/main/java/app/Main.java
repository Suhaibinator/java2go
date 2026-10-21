package app;

import generic.Identity;
import model.Resource;
import model.State;

public class Main {
    public static void main(String[] args) {
        int seed = java.lang.Integer.parseInt(args[0]);
        System.out.println("seed:" + seed);
        java.lang.Integer first = Identity.identity(State.input(seed));
        java.lang.Integer cached = java.lang.Integer.valueOf(seed);
        java.lang.Integer same = Identity.identity(first);
        java.lang.Integer fresh = new java.lang.Integer(seed);
        java.lang.Integer freshSame = Identity.identity(fresh);
        System.out.println("identity:" + first.intValue() + ":" + (first == cached) + ":" + (same == first) + ":" + (freshSame == fresh) + ":" + (fresh != cached));
        System.out.println("native-type:" + first.getClass().getName());
        State.report("identity-state");

        java.lang.Integer absent = Identity.identity(State.missing());
        java.lang.Number boundAbsent = Identity.number((java.lang.Integer)null);
        System.out.println("null:" + (absent == null) + ":" + (boundAbsent == null));
        java.lang.Long wide = java.lang.Long.valueOf(seed + 1L);
        java.lang.Number selected = Identity.choose(false, first, wide);
        java.lang.Integer bounded = Identity.number(first);
        System.out.println("bounds:" + (selected == wide) + ":" + (bounded == first) + ":" + selected.longValue());
        State.report("bounds-state");

        try (Resource a = new Resource("A"); Resource b = new Resource("B")) {
            int missing = Identity.identity(State.missing());
            System.out.println("unbox-missed:" + missing);
        } catch (NullPointerException expected) {
            State.mark("n"); System.out.println("unbox:" + expected.getClass().getSimpleName());
        } finally { State.mark("f"); }
        State.report("unbox-state");

        try (Resource a = new Resource("C"); Resource b = new Resource("D")) {
            java.lang.Integer wrong = Identity.polluted(State.wrong(seed));
            System.out.println("consumer-cast-missed:" + wrong.intValue());
        } catch (ClassCastException expected) {
            State.mark("c"); System.out.println("consumer-cast:" + expected.getClass().getSimpleName());
        } finally { State.mark("f"); }
        State.report("consumer-cast-state");

        try (Resource a = new Resource("E"); Resource b = new Resource("F")) {
            java.lang.Number wrong = Identity.numberPolluted(State.wrong(seed));
            System.out.println("body-cast-missed:" + wrong.intValue());
        } catch (ClassCastException expected) {
            State.mark("b"); System.out.println("body-cast:" + expected.getClass().getSimpleName());
        } finally { State.mark("f"); }
        State.report("body-cast-state");

        int callsBefore = State.calls;
        try (Resource a = new Resource("G"); Resource b = new Resource("H")) {
            java.lang.Integer ignored = Identity.identity(State.failInput());
            System.out.println("argument-missed:" + ignored.intValue());
        } catch (IllegalArgumentException expected) {
            State.mark("a"); System.out.println("argument:" + expected.getMessage() + ":" + (State.calls == callsBefore));
        } finally { State.mark("f"); }
        State.report("argument-state");

        try (Resource a = new Resource("J"); Resource b = new Resource("K")) {
            java.lang.Integer ignored = Identity.fail(State.input(seed));
            System.out.println("body-missed:" + ignored.intValue());
        } catch (IllegalStateException expected) {
            State.mark("e"); System.out.println("body:" + expected.getMessage());
        } finally { State.mark("f"); }
        State.report("body-state");

        shadow.Probe.run(seed);
        State.report("shadow-state");
        System.out.println("closed:" + (State.opened == State.closed));
    }
}
