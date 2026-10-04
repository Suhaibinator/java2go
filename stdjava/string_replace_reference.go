package stdjava

// JavaStringReplaceChar replaces UTF16 code units, preserving the receiver
// reference when the characters are equal or the old character is absent.
func JavaStringReplaceChar(text *JavaString, oldChar, newChar rune) *JavaString {
	RequireJavaString(text)
	oldUnit, newUnit := uint16(oldChar), uint16(newChar)
	if oldUnit == newUnit {
		return text
	}
	first := -1
	for index, unit := range text.units {
		if unit == oldUnit {
			first = index
			break
		}
	}
	if first < 0 {
		return text
	}
	result := text.UTF16Copy()
	for index := first; index < len(result); index++ {
		if result[index] == oldUnit {
			result[index] = newUnit
		}
	}
	return &JavaString{units: result}
}

// JavaStringReplaceExecution implements the CharSequence overload. JDK21
// invokes target.toString and then replacement.toString, even when the first
// returns null. Neither length nor charAt on the supplied sequences is called.
// The caller's Execution and any abrupt callback value are retained directly.
func JavaStringReplaceExecution(execution *Execution, text *JavaString, target, replacement any) *JavaString {
	RequireJavaString(text)
	ReferenceRequireNonNull(target)
	oldText := JavaStringValueOfExecution(execution, target)
	ReferenceRequireNonNull(replacement)
	newText := JavaStringValueOfExecution(execution, replacement)
	RequireJavaString(oldText)
	RequireJavaString(newText)
	oldUnits, newUnits := oldText.units, newText.units
	if len(oldUnits) == 1 && len(newUnits) == 1 {
		return JavaStringReplaceChar(text, rune(oldUnits[0]), rune(newUnits[0]))
	}
	if len(oldUnits) == 0 {
		result := append([]uint16{}, newUnits...)
		for _, unit := range text.units {
			result = append(result, unit)
			result = append(result, newUnits...)
		}
		return &JavaString{units: result}
	}
	var result []uint16
	matched := false
	for index := 0; index < len(text.units); {
		if javaStringUnitsMatchAt(text.units, oldUnits, index) {
			if !matched {
				result = append(result, text.units[:index]...)
				matched = true
			}
			result = append(result, newUnits...)
			index += len(oldUnits)
		} else {
			if matched {
				result = append(result, text.units[index])
			}
			index++
		}
	}
	if !matched {
		return text
	}
	if len(result) == 0 {
		return JavaStringLiteralUTF16(nil)
	}
	return &JavaString{units: result}
}
