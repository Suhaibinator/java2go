package transpiler

import "testing"

func TestRawInnerBoundArrayInheritedShadowAdversarial(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `public class BoundArrayInheritedShadow {
    interface Tagged {}
    static class Value {
        final String text;
        Value(String text) { this.text = text; }
    }
    static class Full extends Value implements Tagged {
        Full(String text) { super(text); }
    }
    static class Note {}
    static class Base<T extends Value & Tagged,A> {
        T outer;
        A region;
        Base(T outer,A region) { this.outer = outer; this.region = region; }
        class Inner<U extends T> {
            U inner;
            <T extends Note> Inner(T note,U[] initial) { inner = initial[0]; }
            String render() { return outer.text + ":" + inner.text + ":" + region; }
        }
    }
    static class Sub<X,Y extends Value & Tagged> extends Base<Y,X> {
        Sub(X region,Y outer) { super(outer,region); }
    }
    @SuppressWarnings({"rawtypes","unchecked"})
    public static String run() {
        Sub<Long,Full> typed = new Sub<Long,Full>(17L,new Full("initial"));
        Sub raw = typed;
        Base.Inner child = raw.new Inner(new Note(),new Value[] {new Value("array")});
        raw.outer = new Value("raw");
        child.inner = new Value("after");
        String before = child.render();
        typed.outer = new Full("restored");
        raw.region = true;
        return before + "|" + child.render() + ":" + (raw == typed);
    }
    public static void main(String[] args) { System.out.println(run()); }
}
`, "raw:after:17|restored:after:true:true")
}
