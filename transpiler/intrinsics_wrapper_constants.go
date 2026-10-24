package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
)

// Every dimension field is a JDK constant variable of Java type int, even
// when its owner represents a narrower or floating-point primitive.
var wrapperDimensionConstants = []struct {
	owner, field string
	value        int32
}{
	{"Byte", "SIZE", 8}, {"Byte", "BYTES", 1},
	{"Short", "SIZE", 16}, {"Short", "BYTES", 2},
	{"Integer", "SIZE", 32}, {"Integer", "BYTES", 4},
	{"Long", "SIZE", 64}, {"Long", "BYTES", 8},
	{"Float", "SIZE", 32}, {"Float", "BYTES", 4}, {"Float", "PRECISION", 24}, {"Float", "MAX_EXPONENT", 127}, {"Float", "MIN_EXPONENT", -126},
	{"Double", "SIZE", 64}, {"Double", "BYTES", 8}, {"Double", "PRECISION", 53}, {"Double", "MAX_EXPONENT", 1023}, {"Double", "MIN_EXPONENT", -1022},
}

func registerWrapperDimensionConstants() {
	for _, fact := range wrapperDimensionConstants {
		registerStaticFieldIntrinsic(fact.owner, fact.field, func(ctx Ctx) ast.Expr {
			return &ast.CallExpr{
				Fun:  javaTypeStringToGoTypeExpr("int", nil, ctx),
				Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(int64(fact.value), 10)}},
			}
		})
		registerStaticFieldIntrinsicResultType(fact.owner, fact.field, "int")
	}
}

// The caller must already have selected an admitted canonical type owner or
// unique static import. A value qualifier never denotes a constant expression.
func wrapperDimensionConstantVariable(key intrinsicKey) bool {
	for _, fact := range wrapperDimensionConstants {
		if key == (intrinsicKey{fact.owner, fact.field}) {
			return true
		}
	}
	return false
}
