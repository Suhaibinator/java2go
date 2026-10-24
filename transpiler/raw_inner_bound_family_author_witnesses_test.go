package transpiler

import "testing"

func TestRawInnerBoundFamilyAuthorClassFirstBoundTransitive(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `public class ClassFirstBoundTransitive {
    interface Tagged {}
    static class Value {
        final String text;
        Value(String text) { this.text = text; }
        String text() { return text; }
    }
    static class FullValue extends Value implements Tagged {
        FullValue(String text) { super(text); }
    }
    static class Outer<T extends Value & Tagged> {
        T outer;
        Outer(T value) { outer = value; }
        class Inner<U extends T> {
            U inner;
            Inner(U value) { inner = value; }
            String render() { return outer.text() + ":" + inner.text(); }
        }
    }
    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Outer raw = new Outer(new FullValue("start"));
        Outer.Inner child = raw.new Inner(new Value("initial-inner"));
        raw.outer = new Value("outer-erased");
        child.inner = new Value("inner-erased");
        return child.render();
    }
}
`, "outer-erased:inner-erased")
}

func TestRawInnerBoundFamilyAuthorInterfaceFirstBoundTransitive(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `public class InterfaceFirstBoundTransitive {
    interface Tagged { String text(); }
    interface Marker {}
    static class TaggedOnly implements Tagged {
        final String value;
        TaggedOnly(String value) { this.value = value; }
        public String text() { return value; }
    }
    static class FullValue extends TaggedOnly implements Marker {
        FullValue(String value) { super(value); }
    }
    static class Outer<T extends Tagged & Marker> {
        T outer;
        Outer(T value) { outer = value; }
        class Inner<U extends T> {
            U inner;
            Inner(U value) { inner = value; }
            String render() { return outer.text() + ":" + inner.text(); }
        }
    }
    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Outer raw = new Outer(new FullValue("start"));
        Outer.Inner child = raw.new Inner(new TaggedOnly("initial-inner"));
        raw.outer = new TaggedOnly("outer-interface");
        child.inner = new TaggedOnly("inner-interface");
        return child.render();
    }
}
`, "outer-interface:inner-interface")
}

func TestRawInnerBoundFamilyAuthorShadowedFirstBounds(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `public class ShadowedFirstBounds {
    static class OuterValue {
        final String text;
        OuterValue(String text) { this.text = text; }
        String text() { return text; }
    }
    static class InnerValue {
        final String text;
        InnerValue(String text) { this.text = text; }
        String text() { return text; }
    }
    static class Outer<T extends OuterValue> {
        T outer;
        Outer(T value) { outer = value; }
        class Inner<T extends InnerValue> {
            T inner;
            Inner(T value) { inner = value; }
            String render() { return Outer.this.outer.text() + ":" + inner.text(); }
        }
    }
    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Outer raw = new Outer(new OuterValue("initial"));
        Outer.Inner child = raw.new Inner(new InnerValue("initial-inner"));
        raw.outer = new OuterValue("outer-T");
        child.inner = new InnerValue("inner-T");
        return child.render();
    }
}
`, "outer-T:inner-T")
}
