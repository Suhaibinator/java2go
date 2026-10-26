package stdjava

// GetParameterCount observes the Java descriptor formals. Runtime-only Go
// execution and generic ABI parameters never enter this descriptor.
func (constructor *Constructor) GetParameterCount() int32 {
	if constructor == nil {
		panic(NewNullPointerException("constructor is null"))
	}
	return int32(len(constructor.descriptor.Parameters))
}
