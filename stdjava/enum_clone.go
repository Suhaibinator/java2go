package stdjava

// EnumCloneExecution is the final java.lang.Enum.clone body. Enum constants
// cannot be cloned, including constants whose enum implements Cloneable.
func EnumCloneExecution(execution *Execution) any {
	requireExecution(execution)
	panic(CloneNotSupportedException{newJavaThrowableBase("CloneNotSupportedException", nil)})
}
