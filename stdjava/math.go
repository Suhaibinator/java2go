package stdjava

import (
	"math"
)

// This file implements the java.lang.Math operations whose Go equivalents
// either are not type-preserving (math.Abs/Max/Min are float64-only) or have a
// different return contract (Math.round returns long via round-half-up).

// The `number` constraint used by these helpers is declared in common.go.

// MathAbs returns the absolute value of x, preserving its numeric type, matching
// Java's overloaded Math.abs (which returns the same type it is given rather than
// always a float64).
func MathAbs[T number](x T) T {
	if x < 0 {
		return -x
	}
	return x
}

// MathMax returns the larger of a and b, preserving the numeric type, matching
// Java's overloaded Math.max.
func MathMax[T number](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// MathMin returns the smaller of a and b, preserving the numeric type, matching
// Java's overloaded Math.min.
func MathMin[T number](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// MathRound rounds x to the nearest long using round-half-up (ties round toward
// positive infinity), matching Java's Math.round(double). Go's math.Round rounds
// half away from zero, which differs for negative .5 values, so this is computed
// explicitly.
func MathRound(x float64) int64 {
	return int64(math.Floor(x + 0.5))
}

// The kernels and medium-size argument reduction below are a Go port of the
// OpenJDK FdLibm implementation used by StrictMath. Java Math delegates to
// StrictMath by default, so Go's Cephes-based math.Sin and math.Cos can
// accumulate observable last-bit differences in otherwise identical programs.
//
// OpenJDK source: src/java.base/share/classes/java/lang/FdLibm.java

const (
	javaTrigExpSignifBits = uint32(0x7fffffff)
	javaTrigExpBits       = uint32(0x7ff00000)
)

func javaHighWord(value float64) uint32 {
	return uint32(math.Float64bits(value) >> 32)
}

func javaFloatFromWords(high, low uint32) float64 {
	return math.Float64frombits(uint64(high)<<32 | uint64(low))
}

const (
	javaSinS1 = -0x1.5555555555549p-3
	javaSinS2 = 0x1.111111110f8a6p-7
	javaSinS3 = -0x1.a01a019c161d5p-13
	javaSinS4 = 0x1.71de357b1fe7dp-19
	javaSinS5 = -0x1.ae5e68a2b9cebp-26
	javaSinS6 = 0x1.5d93a5acfd57cp-33

	javaCosC1 = 0x1.555555555554cp-5
	javaCosC2 = -0x1.6c16c16c15177p-10
	javaCosC3 = 0x1.a01a019cb159p-16
	javaCosC4 = -0x1.27e4f809c52adp-22
	javaCosC5 = 0x1.1ee9ebdb4b1c4p-29
	javaCosC6 = -0x1.8fae9be8838d4p-37
)

func javaKernelSin(x, y float64, tail int) float64 {
	ix := javaHighWord(x) & javaTrigExpSignifBits
	if ix < 0x3e400000 && int32(x) == 0 {
		return x
	}
	z := x * x
	v := z * x
	r := javaSinS2 + z*(javaSinS3+z*(javaSinS4+z*(javaSinS5+z*javaSinS6)))
	if tail == 0 {
		return x + v*(javaSinS1+z*r)
	}
	return x - ((z*(0.5*y-v*r) - y) - v*javaSinS1)
}

func javaKernelCos(x, y float64) float64 {
	ix := javaHighWord(x) & javaTrigExpSignifBits
	if ix < 0x3e400000 && int32(x) == 0 {
		return 1
	}
	z := x * x
	r := z * (javaCosC1 + z*(javaCosC2+z*(javaCosC3+z*(javaCosC4+z*(javaCosC5+z*javaCosC6)))))
	if ix < 0x3fd33333 {
		return 1 - (0.5*z - (z*r - x*y))
	}
	var qx float64
	if ix > 0x3fe90000 {
		qx = 0.28125
	} else {
		qx = javaFloatFromWords(ix-0x00200000, 0)
	}
	hz := 0.5*z - qx
	a := 1 - qx
	return a - (hz - (z*r - x*y))
}

var javaNPio2HighWords = [...]uint32{
	0x3ff921fb, 0x400921fb, 0x4012d97c, 0x401921fb, 0x401f6a7a, 0x4022d97c,
	0x4025fdbb, 0x402921fb, 0x402c463a, 0x402f6a7a, 0x4031475c, 0x4032d97c,
	0x40346b9c, 0x4035fdbb, 0x40378fdb, 0x403921fb, 0x403ab41b, 0x403c463a,
	0x403dd85a, 0x403f6a7a, 0x40407e4c, 0x4041475c, 0x4042106c, 0x4042d97c,
	0x4043a28c, 0x40446b9c, 0x404534ac, 0x4045fdbb, 0x4046c6cb, 0x40478fdb,
	0x404858eb, 0x404921fb,
}

const (
	javaInvPio2 = 0x1.45f306dc9c883p-1
	javaPio2_1  = 0x1.921fb544p0
	javaPio2_1t = 0x1.0b4611a626331p-34
	javaPio2_2  = 0x1.0b4611a6p-34
	javaPio2_2t = 0x1.3198a2e037073p-69
	javaPio2_3  = 0x1.3198a2ep-69
	javaPio2_3t = 0x1.b839a252049c1p-104
)

// javaRemPio2 returns the fdlibm two-part remainder for arguments up to
// 2^19*(pi/2). Larger finite arguments retain the Go implementation until the
// full Payne-Hanek table is needed.
func javaRemPio2(x float64) (n int, y0, y1 float64, exact bool) {
	hx := javaHighWord(x)
	ix := hx & javaTrigExpSignifBits
	if ix <= 0x3fe921fb {
		return 0, x, 0, true
	}
	if ix < 0x4002d97c {
		if int32(hx) > 0 {
			z := x - javaPio2_1
			if ix != 0x3ff921fb {
				y0 = z - javaPio2_1t
				y1 = (z - y0) - javaPio2_1t
			} else {
				z -= javaPio2_2
				y0 = z - javaPio2_2t
				y1 = (z - y0) - javaPio2_2t
			}
			return 1, y0, y1, true
		}
		z := x + javaPio2_1
		if ix != 0x3ff921fb {
			y0 = z + javaPio2_1t
			y1 = (z - y0) + javaPio2_1t
		} else {
			z += javaPio2_2
			y0 = z + javaPio2_2t
			y1 = (z - y0) + javaPio2_2t
		}
		return -1, y0, y1, true
	}
	if ix > 0x413921fb {
		return 0, 0, 0, false
	}

	t := math.Abs(x)
	n = int(t*javaInvPio2 + 0.5)
	fn := float64(n)
	r := t - fn*javaPio2_1
	w := fn * javaPio2_1t
	if n < 32 && ix != javaNPio2HighWords[n-1] {
		y0 = r - w
	} else {
		j := int(ix >> 20)
		y0 = r - w
		i := j - int((javaHighWord(y0)>>20)&0x7ff)
		if i > 16 {
			t = r
			w = fn * javaPio2_2
			r = t - w
			w = fn*javaPio2_2t - ((t - r) - w)
			y0 = r - w
			i = j - int((javaHighWord(y0)>>20)&0x7ff)
			if i > 49 {
				t = r
				w = fn * javaPio2_3
				r = t - w
				w = fn*javaPio2_3t - ((t - r) - w)
				y0 = r - w
			}
		}
	}
	y1 = (r - y0) - w
	if int32(hx) < 0 {
		return -n, -y0, -y1, true
	}
	return n, y0, y1, true
}

// JavaMathSin implements Math.sin with StrictMath's fdlibm rounding for all
// ordinary-sized arguments.
func JavaMathSin(x float64) float64 {
	ix := javaHighWord(x) & javaTrigExpSignifBits
	if ix <= 0x3fe921fb {
		return javaKernelSin(x, 0, 0)
	}
	if ix >= javaTrigExpBits {
		return x - x
	}
	n, y0, y1, exact := javaRemPio2(x)
	if !exact {
		return math.Sin(x)
	}
	switch n & 3 {
	case 0:
		return javaKernelSin(y0, y1, 1)
	case 1:
		return javaKernelCos(y0, y1)
	case 2:
		return -javaKernelSin(y0, y1, 1)
	default:
		return -javaKernelCos(y0, y1)
	}
}

// JavaMathCos implements Math.cos with StrictMath's fdlibm rounding for all
// ordinary-sized arguments.
func JavaMathCos(x float64) float64 {
	ix := javaHighWord(x) & javaTrigExpSignifBits
	if ix <= 0x3fe921fb {
		return javaKernelCos(x, 0)
	}
	if ix >= javaTrigExpBits {
		return x - x
	}
	n, y0, y1, exact := javaRemPio2(x)
	if !exact {
		return math.Cos(x)
	}
	switch n & 3 {
	case 0:
		return javaKernelCos(y0, y1)
	case 1:
		return -javaKernelSin(y0, y1, 1)
	case 2:
		return -javaKernelCos(y0, y1)
	default:
		return javaKernelSin(y0, y1, 1)
	}
}
