package transpiler

import "testing"

func TestSourceDowncastRecoversCanonicalObjectView(t *testing.T) {
	assertGenericReferenceEqualityResult(t, `
public class SourceDowncastViews {
    static int evaluations;
    static class Base { int weight() { return 1; } }
    static class Child extends Base { int weight() { return 7; } }
    static class Sibling extends Base {}
    static Base select(Base value) { evaluations++; return value; }
    public static int run() {
        Child original = new Child();
        Base base = original;
        Child recovered = (Child) select(base);
        Base missing = null;
        Child nullChild = (Child) missing;
        int score = recovered == original ? 1 : 0;
        if (nullChild == null) score += 2;
        try { Sibling wrong = (Sibling) select(base); }
        catch (ClassCastException expected) { score += 4; }
        return score + recovered.weight() + evaluations;
    }
}
`, 16)
}
