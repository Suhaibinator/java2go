package stdjava

func (field *Field) GetLong(receiver any) int64 {
	return field.GetLongExecution(NewExecution(), receiver)
}

func (field *Field) GetLongExecution(execution *Execution, receiver any) int64 {
	value := field.GetExecution(execution, receiver)
	switch field.descriptor.Type {
	case PrimitiveByteTypeID, PrimitiveShortTypeID, PrimitiveCharTypeID, PrimitiveIntTypeID, PrimitiveLongTypeID:
		return reflectionUnbox(value, PrimitiveLongTypeID).(int64)
	default:
		panic(NewIllegalArgumentException("field cannot be widened to long"))
	}
}

func (field *Field) SetLong(receiver any, value int64) {
	field.SetLongExecution(NewExecution(), receiver, value)
}

func (field *Field) SetLongExecution(execution *Execution, receiver any, value int64) {
	field.setExecution(execution, receiver, BoxLong(value), func(boxed any, id TypeID) any {
		if !isPrimitiveTypeID(id) {
			panic(NewIllegalArgumentException("typed long setter requires a primitive field"))
		}
		return reflectionFieldValue(boxed, id)
	})
}
