package stdjava

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Duration struct {
	seconds int64
	nanos   int32
}
type Instant struct {
	seconds int64
	nanos   int32
}

var durationZero = &Duration{}
var instantEpoch = &Instant{}

func normalizedJavaTime(seconds, nanoAdjustment int64) (int64, int32) {
	carry, nanos := nanoAdjustment/1000000000, nanoAdjustment%1000000000
	if nanos < 0 {
		carry--
		nanos += 1000000000
	}
	return MathAddExact(seconds, carry), int32(nanos)
}
func DurationOfSeconds(seconds, nanoAdjustment int64) *Duration {
	seconds, nanos := normalizedJavaTime(seconds, nanoAdjustment)
	if seconds == 0 && nanos == 0 {
		return durationZero
	}
	return &Duration{seconds, nanos}
}
func InstantOfEpochSecond(seconds, nanoAdjustment int64) *Instant {
	seconds, nanos := normalizedJavaTime(seconds, nanoAdjustment)
	if seconds < -31557014167219200 || seconds > 31556889864403199 {
		panic(NewDateTimeException("Instant exceeds minimum or maximum instant"))
	}
	if seconds == 0 && nanos == 0 {
		return instantEpoch
	}
	return &Instant{seconds, nanos}
}
func (d *Duration) GetSeconds() int64    { return timeRequire(d).seconds }
func (d *Duration) GetNano() int32       { return timeRequire(d).nanos }
func (i *Instant) GetEpochSecond() int64 { return timeRequire(i).seconds }
func (i *Instant) GetNano() int32        { return timeRequire(i).nanos }
func (d *Duration) Equals(value any) bool {
	timeRequire(d)
	other, ok := value.(*Duration)
	return ok && other != nil && *d == *other
}
func (i *Instant) Equals(value any) bool {
	timeRequire(i)
	other, ok := value.(*Instant)
	return ok && other != nil && *i == *other
}
func (d *Duration) HashCode() int32 {
	timeRequire(d)
	return int32(d.seconds^(d.seconds>>32)) + 51*d.nanos
}
func (i *Instant) HashCode() int32 {
	timeRequire(i)
	return int32(i.seconds^(i.seconds>>32)) + 51*i.nanos
}
func (*Duration) JavaDynamicTypeID() TypeID { return "java.time.Duration" }
func (*Instant) JavaDynamicTypeID() TypeID  { return "java.time.Instant" }
func (d *Duration) String() string {
	timeRequire(d)
	if d.seconds == 0 && d.nanos == 0 {
		return "PT0S"
	}
	effective := d.seconds
	if effective < 0 && d.nanos > 0 {
		effective++
	}
	hours, minutes, secs := effective/3600, (effective%3600)/60, effective%60
	var out strings.Builder
	out.WriteString("PT")
	if hours != 0 {
		fmt.Fprintf(&out, "%dH", hours)
	}
	if minutes != 0 {
		fmt.Fprintf(&out, "%dM", minutes)
	}
	if secs == 0 && d.nanos == 0 && out.Len() > 2 {
		return out.String()
	}
	if d.seconds < 0 && d.nanos > 0 && secs == 0 {
		out.WriteString("-0")
	} else {
		out.WriteString(strconv.FormatInt(secs, 10))
	}
	if d.nanos > 0 {
		nano := d.nanos
		if d.seconds < 0 {
			nano = 1000000000 - nano
		}
		out.WriteString(timeFraction(nano, false))
	}
	out.WriteByte('S')
	return out.String()
}
func (i *Instant) String() string {
	timeRequire(i)
	t := time.Unix(i.seconds, int64(i.nanos)).UTC()
	return timeYear(int32(t.Year())) + fmt.Sprintf("-%02d-%02dT%02d:%02d:%02d", t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second()) + timeFraction(i.nanos, true) + "Z"
}
func (d *Duration) StringJava2goExecution(*Execution) *JavaString {
	return JavaStringFromHostUTF8(d.String())
}
func (i *Instant) StringJava2goExecution(*Execution) *JavaString {
	return JavaStringFromHostUTF8(i.String())
}

var durationPattern = regexp.MustCompile(`(?i)^([-+]?)P(?:([-+]?[0-9]+)D)?(T(?:([-+]?[0-9]+)H)?(?:([-+]?[0-9]+)M)?(?:([-+]?[0-9]+)(?:[.,]([0-9]{0,9}))?S)?)?$`)

func DurationParseExecution(execution *Execution, value any) *Duration {
	text := timeText(execution, value)
	m := durationPattern.FindStringSubmatch(timeHost(text))
	if m == nil || m[3] == "T" || (m[2] == "" && m[4] == "" && m[5] == "" && m[6] == "") {
		timeParseFailure(text, 0)
	}
	seconds := new(big.Int)
	for _, part := range []struct {
		index  int
		factor int64
	}{{2, 86400}, {4, 3600}, {5, 60}, {6, 1}} {
		if m[part.index] != "" {
			v, ok := new(big.Int).SetString(m[part.index], 10)
			if !ok {
				timeParseFailure(text, 0)
			}
			v.Mul(v, big.NewInt(part.factor))
			if !v.IsInt64() {
				timeParseFailure(text, 0)
			}
			seconds.Add(seconds, v)
		}
	}
	if !seconds.IsInt64() {
		timeParseFailure(text, 0)
	}
	nano := int64(0)
	if m[7] != "" {
		nano, _ = strconv.ParseInt(m[7]+strings.Repeat("0", 9-len(m[7])), 10, 64)
	}
	if strings.HasPrefix(m[6], "-") {
		nano = -nano
	}
	// Normalization and leading negation have exact long overflow behavior.
	var result *Duration
	func() {
		defer func() {
			if failure := recover(); failure != nil {
				if _, ok := failure.(ArithmeticException); ok {
					timeParseFailure(text, 0)
				}
				panic(failure)
			}
		}()
		result = DurationOfSeconds(seconds.Int64(), nano)
		if m[1] == "-" {
			if result.nanos == 0 {
				if result.seconds == -9223372036854775808 {
					timeParseFailure(text, 0)
				}
				result = DurationOfSeconds(-result.seconds, 0)
			} else {
				result = DurationOfSeconds(-(result.seconds + 1), int64(1000000000-result.nanos))
			}
		}
	}()
	return result
}
func InstantParseExecution(execution *Execution, value any) *Instant {
	text := timeText(execution, value)
	host := timeHost(text)
	t, err := time.Parse(time.RFC3339Nano, host)
	if err != nil {
		timeParseFailure(text, 0)
	}
	return InstantOfEpochSecond(t.Unix(), int64(t.Nanosecond()))
}
