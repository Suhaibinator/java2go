package transpiler

import "testing"

func TestCampaignAssignabilityStrictJVMControls(t *testing.T) {
	previousStrict := diagnostics.strict
	setStrictMode(true)
	t.Cleanup(func() { setStrictMode(previousStrict) })
	const source = `public class AssignabilityOracle {
 interface Flag { int value(); }
 static class Base { int value; Base(int value) { this.value = value; } }
 static class Child extends Base implements Flag {
  Child(int value) { super(value); }
  public int value() { return value; }
 }
 public static String run() {
  Child child = new Child(7);
  Child[] actual = new Child[]{child};
  Base[] bases = actual;
  Flag[] flags = actual;
  Object[] erased = bases;
  String out = (bases[0] == child) + ":" + (flags[0] == child) + ":" + flags[0].value() + ":";
  try { erased[0] = new Base(9); out += "accepted:"; }
  catch (ArrayStoreException expected) { out += "store:" + (actual[0] == child) + ":"; }
  Object wide = actual;
  out += (wide instanceof Base[]) + ":" + (wide instanceof Flag[]) + ":";
  try { Child[] invalid = (Child[])new Base[1]; out += "cast-accepted:"; }
  catch (ClassCastException expected) { out += "cast:"; }
  Object primitive = new int[]{3};
  Object rows = new int[1][1];
  out += (primitive instanceof int[]) + ":" + (primitive instanceof long[]) + ":" + (rows instanceof Object[]) + ":";
  Object text = "A\uD800";
  out += (text instanceof String) + ":" + (text instanceof CharSequence) + ":" + (text instanceof java.io.Serializable);
  return out;
 }
}`
	verifyCanonicalStringStreamOracle(t, "AssignabilityOracle", source)
	t.Run("CovarianceStoresAndDescriptors", TestGeneratedReferenceArrayCovarianceChecksStoresAndReifiesAllArrayDescriptors)
	t.Run("ObjectArrayView", TestReferenceArrayAdversarial_ObjectArrayViewRetainsDynamicComponent)
	t.Run("CastsAndInstanceof", TestReferenceArrayAdversarial_CastsAndInstanceofUseJavaDescriptors)
	t.Run("InterfaceArrays", TestReferenceArrayAdversarial_InterfaceArraysAcceptSyntheticImplementors)
}
