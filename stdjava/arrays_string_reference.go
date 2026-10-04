package stdjava

// Java arrays inherit Object.toString. These nominal runtime adapters preserve
// their descriptor, identity hash and fresh canonical String result.
func (array *PrimitiveArray[T]) StringJava2goExecution(execution *Execution) *JavaString {
	return ObjectDefaultJavaStringExecution(execution, array)
}

func (array *ReferenceArray) StringJava2goExecution(execution *Execution) *JavaString {
	return ObjectDefaultJavaStringExecution(execution, array)
}

// javaPrimitiveArrayText is implemented only by the runtime's descriptor-bearing
// primitive arrays. Formatting keeps char's UTF16 ABI distinct from int32.
type javaPrimitiveArrayText interface {
	appendJavaArrayText([]uint16) []uint16
	javaArrayTextLength() int
}

func (array *PrimitiveArray[T]) javaArrayTextLength() int { return len(array.Elements) }

func (array *PrimitiveArray[T]) appendJavaArrayText(units []uint16) []uint16 {
	units = append(units, '[')
	for index, element := range array.Elements {
		if index != 0 {
			units = append(units, ',', ' ')
		}
		var text *JavaString
		switch value := any(element).(type) {
		case bool:
			text = JavaStringValueOfBoolean(value)
		case int8:
			text = JavaStringValueOfInt(int32(value))
		case int16:
			text = JavaStringValueOfInt(int32(value))
		case int32:
			if array.componentType == PrimitiveTypeID("char") {
				units = append(units, uint16(value))
				continue
			}
			text = JavaStringValueOfInt(value)
		case int64:
			text = JavaStringValueOfLong(value)
		case float32:
			text = JavaStringValueOfFloat(value)
		case float64:
			text = JavaStringValueOfDouble(value)
		default:
			panic(NewUnsupportedOperationException("unsupported primitive array String conversion"))
		}
		units = append(units, text.units...)
	}
	return append(units, ']')
}

// JavaArrayToStringExecution implements every Arrays.toString overload at the
// canonical String boundary. Reference elements are read live, left to right,
// and invoke their declared Java toString once with the caller's execution.
func JavaArrayToStringExecution(execution *Execution, value any) *JavaString {
	requireExecution(execution)
	if javaReferenceIsNull(value) {
		return JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'})
	}
	if primitive, ok := value.(javaPrimitiveArrayText); ok {
		if primitive.javaArrayTextLength() == 0 {
			return JavaStringLiteralUTF16([]uint16{'[', ']'})
		}
		return &JavaString{units: primitive.appendJavaArrayText(nil)}
	}
	array, ok := value.(*ReferenceArray)
	if !ok {
		panic(NewUnsupportedOperationException("Arrays.toString requires a Java array"))
	}
	if len(array.elements) == 0 {
		return JavaStringLiteralUTF16([]uint16{'[', ']'})
	}
	units := []uint16{'['}
	for index, element := range array.elements {
		if index != 0 {
			units = append(units, ',', ' ')
		}
		units = append(units, JavaStringTextOperandExecution(execution, element).units...)
	}
	return &JavaString{units: append(units, ']')}
}

// JavaArrayDeepToStringExecution implements Arrays.deepToString(Object[]).
// The active path belongs to this invocation, so repeated sibling references
// render fully and a throwing callback leaves no persistent recursion state.
func JavaArrayDeepToStringExecution(execution *Execution, value any) *JavaString {
	requireExecution(execution)
	if javaReferenceIsNull(value) {
		return JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'})
	}
	array, ok := value.(*ReferenceArray)
	if !ok {
		panic(NewUnsupportedOperationException("Arrays.deepToString requires a Java reference array"))
	}
	return &JavaString{units: appendJavaDeepArrayText(execution, nil, array, make(map[*ReferenceArray]bool))}
}

func appendJavaDeepArrayText(execution *Execution, units []uint16, array *ReferenceArray, active map[*ReferenceArray]bool) []uint16 {
	if array == nil {
		return append(units, 'n', 'u', 'l', 'l')
	}
	if active[array] {
		return append(units, '[', '.', '.', '.', ']')
	}
	active[array] = true
	defer delete(active, array)
	units = append(units, '[')
	for index, element := range array.elements {
		if index != 0 {
			units = append(units, ',', ' ')
		}
		if javaReferenceIsNull(element) {
			units = append(units, 'n', 'u', 'l', 'l')
		} else if nested, ok := element.(*ReferenceArray); ok {
			units = appendJavaDeepArrayText(execution, units, nested, active)
		} else if primitive, ok := element.(javaPrimitiveArrayText); ok {
			units = primitive.appendJavaArrayText(units)
		} else {
			units = append(units, JavaStringTextOperandExecution(execution, element).units...)
		}
	}
	return append(units, ']')
}
