package stdjava

// IsVarArgs reads the Java method declaration flag. The method descriptor
// retains the distinction between an array parameter and a varargs parameter.
func (method *Method) IsVarArgs() bool {
	method.requireMethod()
	return method.descriptor.Modifiers&0x80 != 0
}
