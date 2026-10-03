package stdjava

// JDBC parses Java String code units with Integer's decimal CharSequence-range
// contract, including range-relative error indexes and the original payload.
func sqlParseIntegerUnits(units []uint16, begin, end int) int32 {
	if begin == end {
		panic(javaIntegerParseInputException(&JavaString{}, 10))
	}
	fail := func(index int) {
		message := []uint16{'E', 'r', 'r', 'o', 'r', ' ', 'a', 't', ' ', 'i', 'n', 'd', 'e', 'x', ' '}
		message = append(message, JavaStringValueOfInt(int32(index-begin)).units...)
		message = append(message, ' ', 'i', 'n', ':', ' ', '"')
		message = append(message, units[begin:end]...)
		message = append(message, '"')
		panic(NewJavaNumberFormatException(&JavaString{units: message}))
	}
	i, negative, limit := begin, false, int32(-2147483647)
	if first := units[i]; first < '0' {
		if first == '-' {
			negative, limit = true, -2147483648
		} else if first != '+' {
			fail(i)
		}
		i++
		if i == end {
			fail(i)
		}
	}
	var result int32
	for ; i < end; i++ {
		digit := javaIntegerDigit21(units[i])
		if digit < 0 || digit > 9 || result < limit/10 {
			fail(i)
		}
		result *= 10
		if result < limit+digit {
			fail(i)
		}
		result -= digit
	}
	if negative {
		return result
	}
	return -result
}

func sqlIndexUnit(units []uint16, unit uint16, from int) int {
	if from < 0 {
		from = 0
	}
	for i := from; i < len(units); i++ {
		if units[i] == unit {
			return i
		}
	}
	return -1
}

func sqlDateUnitParts(units []uint16, end int) (year, month, day int32, valid bool) {
	first := sqlIndexUnit(units, '-', 0)
	second := sqlIndexUnit(units, '-', first+1)
	if first != 4 || second <= first+1 || second > first+3 || end <= second+1 || end > second+3 {
		return
	}
	year = sqlParseIntegerUnits(units, 0, first)
	month = sqlParseIntegerUnits(units, first+1, second)
	day = sqlParseIntegerUnits(units, second+1, end)
	valid = month >= 1 && month <= 12 && day >= 1 && day <= 31
	return
}

func SQLDateValueOfJavaString(text *JavaString) *SQLDate {
	if text == nil {
		panic(NewJavaIllegalArgumentExceptionMessage(nil))
	}
	year, month, day, valid := sqlDateUnitParts(text.units, len(text.units))
	if !valid {
		panic(NewJavaIllegalArgumentExceptionMessage(nil))
	}
	return NewSQLDate(sqlLocalMillis(year, month, day, 0, 0, 0))
}

func SQLTimeValueOfJavaString(text *JavaString) *SQLTime {
	if text == nil {
		panic(NewJavaIllegalArgumentExceptionMessage(nil))
	}
	units := text.units
	first := sqlIndexUnit(units, ':', 0)
	second := sqlIndexUnit(units, ':', first+1)
	if first <= 0 || second <= 0 || second >= len(units)-1 {
		panic(NewJavaIllegalArgumentExceptionMessage(nil))
	}
	hour := sqlParseIntegerUnits(units, 0, first)
	minute := sqlParseIntegerUnits(units, first+1, second)
	seconds := sqlParseIntegerUnits(units, second+1, len(units))
	return NewSQLTime(sqlLocalMillis(1970, 1, 1, hour, minute, seconds))
}

func SQLTimestampValueOfJavaString(text *JavaString) *SQLTimestamp {
	invalid := func() {
		panic(NewJavaIllegalArgumentExceptionMessage(JavaStringLiteralUTF16([]uint16{
			'T', 'i', 'm', 'e', 's', 't', 'a', 'm', 'p', ' ', 'f', 'o', 'r', 'm', 'a', 't', ' ', 'm', 'u', 's', 't', ' ', 'b', 'e', ' ',
			'y', 'y', 'y', 'y', '-', 'm', 'm', '-', 'd', 'd', ' ', 'h', 'h', ':', 'm', 'm', ':', 's', 's', '[', '.', 'f', 'f', 'f', 'f', 'f', 'f', 'f', 'f', 'f', ']',
		})))
	}
	if text == nil {
		panic(NewJavaIllegalArgumentExceptionMessage(JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l', ' ', 's', 't', 'r', 'i', 'n', 'g'})))
	}
	units := JavaStringTrim(text).units
	space := sqlIndexUnit(units, ' ', 0)
	if space < 0 {
		invalid()
	}
	year, month, day, valid := sqlDateUnitParts(units, space)
	if !valid {
		invalid()
	}
	first := sqlIndexUnit(units, ':', space+1)
	second := sqlIndexUnit(units, ':', first+1)
	period := sqlIndexUnit(units, '.', second+1)
	if first <= 0 || second <= 0 || second >= len(units)-1 {
		invalid()
	}
	hour := sqlParseIntegerUnits(units, space+1, first)
	minute := sqlParseIntegerUnits(units, first+1, second)
	var seconds, nanos int32
	if period > 0 && period < len(units)-1 {
		seconds = sqlParseIntegerUnits(units, second+1, period)
		precision := len(units) - period - 1
		if digit := javaIntegerDigit21(units[period+1]); precision > 9 || digit < 0 || digit > 9 {
			invalid()
		}
		nanos = sqlParseIntegerUnits(units, period+1, len(units))
		for ; precision < 9; precision++ {
			nanos *= 10
		}
	} else if period > 0 {
		invalid()
	} else {
		seconds = sqlParseIntegerUnits(units, second+1, len(units))
	}
	stamp := NewSQLTimestamp(sqlLocalMillis(year, month, day, hour, minute, seconds))
	stamp.SetNanos(nanos)
	return stamp
}

// SQL formatting is a trusted ASCII numeric kernel. Each Java toString call
// creates a fresh wrapper, including when reached through Object conversion.
func (date *SQLDate) StringJava2goExecution(_ *Execution) *JavaString {
	return javaStringFromNumericASCII(date.String(), false)
}
func (value *SQLTime) StringJava2goExecution(_ *Execution) *JavaString {
	return javaStringFromNumericASCII(value.String(), false)
}
func (stamp *SQLTimestamp) StringJava2goExecution(_ *Execution) *JavaString {
	return javaStringFromNumericASCII(stamp.String(), false)
}
