package transpiler

import "testing"

// The factory's method binder demands one physical Adapter<T> family, while
// named nested subclasses override apply with distinct source class signatures.
// Construction must install an erased dispatch bridge before the first call.
func TestCampaignFactoryErasedApplyBridgeJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>erased-apply</artifactId><version>1</version></project>`,
		"src/main/java/api/Adapter.java": `package api;
public abstract class Adapter<T> {
    private int calls;
    protected final void record() { calls++; }
    public final int calls() { return calls; }
    public abstract T apply(T input);
}`,
		"src/main/java/api/Factory.java": `package api;
public interface Factory { <T> Adapter<T> create(boolean first); }`,
		"src/main/java/domain/First.java": `package domain;
public final class First { }`,
		"src/main/java/domain/Second.java": `package domain;
public final class Second { }`,
		"src/main/java/impl/Factories.java": `package impl;
import api.Adapter;
import api.Factory;
import domain.First;
import domain.Second;
public final class Factories implements Factory {
    @SuppressWarnings("unchecked")
    public <T> Adapter<T> create(boolean first) {
        if (first) { return (Adapter<T>) new FirstAdapter(); }
        return (Adapter<T>) new SecondAdapter();
    }
    private static final class FirstAdapter extends Adapter<First> {
        public First apply(First input) { record(); return input; }
    }
    private static final class SecondAdapter extends Adapter<Second> {
        public Second apply(Second input) { record(); return input; }
    }
}`,
		"src/main/java/app/Entry.java": `package app;
import api.Adapter;
import api.Factory;
import domain.First;
import domain.Second;
import impl.Factories;
public class Entry {
    @SuppressWarnings({"rawtypes", "unchecked"})
    public static void main(String[] args) {
        Factory factory = new Factories();
        Adapter<First> first = factory.create(true);
        Adapter<Second> second = factory.create(false);
        First a = new First();
        Second b = new Second();
        boolean firstIdentity = first.apply(a) == a;
        boolean secondIdentity = second.apply(b) == b;
        Adapter raw = first;
        boolean rawIdentity = raw.apply(a) == a;
        boolean nullResult = raw.apply(null) == null;
        boolean rejected = false;
        try { raw.apply(b); } catch (ClassCastException expected) { rejected = true; }
        System.out.println(firstIdentity + ":" + secondIdentity + ":" + rawIdentity + ":" + nullResult + ":" + rejected + ":" + first.calls() + ":" + second.calls());
    }
}`,
	}, "app.Entry", "true:true:true:true:true:3:1\n")
}

// A source class may widen to canonical java.lang.Object, but unrelated source
// declarations named Object must retain their nominal meaning.
func TestCampaignFactoryErasedApplyBridgeResultCompatibility(t *testing.T) {
	helper := setupParseHelper(t, `
public class FactoryBridgeCompatibility {
    static class Payload { }
    static class Object { }
    static class Child extends Object { }
}
`)
	owner := overrideBridgeTestScope(t, helper, "Payload")
	if !overrideBridgeResultCompatible("Payload", owner, "java.lang.Object", owner, helper.Ctx) {
		t.Fatal("source Payload must widen to canonical java.lang.Object")
	}
	if overrideBridgeResultCompatible("Payload", owner, "Object", owner, helper.Ctx) {
		t.Fatal("unrelated source class Object must not accept Payload")
	}
	if !overrideBridgeResultCompatible("Child", owner, "Object", owner, helper.Ctx) {
		t.Fatal("source Child must retain its actual source Object superclass")
	}
	if overrideBridgeResultCompatible("int", owner, "java.lang.Object", owner, helper.Ctx) {
		t.Fatal("primitive result requires boxing, not plain reference widening")
	}
}
