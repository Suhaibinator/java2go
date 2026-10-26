package stdjava

import "testing"

// Expected predicates are exact imported+qualified observations from the frozen
// Java60-row calibration, repeated three times on pinned JDK21. This focused
// runtime probe does not replace that whole app, whose Go build is still held.
func TestMathExtrema181JDKNumericRuntimeOracle(t *testing.T) {
 fp := float32(0); fm := -fp; fn := fp/fp
 dp := float64(0); dm := -dp; dn := dp/dp
 cases := []struct {name string; actual func() bool; want bool}{
  {"float-nan-left",func() bool {return func() bool { v := MathMax(fn, float32(1)); return v != v }()},true},
  {"float-nan-right",func() bool {return func() bool { v := MathMax(float32(1), fn); return v != v }()},true},
  {"float-min-nan-left",func() bool {return func() bool { v := MathMin(fn, float32(1)); return v != v }()},true},
  {"float-min-nan-right",func() bool {return func() bool { v := MathMin(float32(1), fn); return v != v }()},true},
  {"double-nan-left",func() bool {return func() bool { v := MathMax(dn, float64(1)); return v != v }()},true},
  {"double-nan-right",func() bool {return func() bool { v := MathMax(float64(1), dn); return v != v }()},true},
  {"double-min-nan-left",func() bool {return func() bool { v := MathMin(dn, float64(1)); return v != v }()},true},
  {"double-min-nan-right",func() bool {return func() bool { v := MathMin(float64(1), dn); return v != v }()},true},
  {"float-max-zero-pn",func() bool {return float32(1)/MathMax(fp,fm)>float32(0)},true},
  {"float-max-zero-np",func() bool {return float32(1)/MathMax(fm,fp)>float32(0)},true},
  {"float-min-zero-pn",func() bool {return float32(1)/MathMin(fp,fm)<float32(0)},true},
  {"float-min-zero-np",func() bool {return float32(1)/MathMin(fm,fp)<float32(0)},true},
  {"double-max-zero-pn",func() bool {return float64(1)/MathMax(dp,dm)>float64(0)},true},
  {"double-max-zero-np",func() bool {return float64(1)/MathMax(dm,dp)>float64(0)},true},
  {"double-min-zero-pn",func() bool {return float64(1)/MathMin(dp,dm)<float64(0)},true},
  {"double-min-zero-np",func() bool {return float64(1)/MathMin(dm,dp)<float64(0)},true},
  {"long-exact-max",func() bool {return MathMax(int64(9007199254740993),int64(9007199254740992))==int64(9007199254740993)},true},
  {"long-exact-min",func() bool {return MathMin(int64(9007199254740993),int64(9007199254740992))==int64(9007199254740992)},true},
 }
 for _, tc := range cases {
  t.Run(tc.name,func(t *testing.T){
   got := tc.actual()
   t.Logf("MATH181_RUNTIME_ORACLE name=%s actual=%t JDK=%t",tc.name,got,tc.want)
   if got != tc.want {t.Errorf("existing runtime %s predicate=%t want frozen JDK %t",tc.name,got,tc.want)}
  })
 }
}
