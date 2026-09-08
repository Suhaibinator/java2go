package common

import (
	fmt "fmt"
	stdjava "github.com/NickyBoy89/java2go/stdjava"
)

type Logger struct {
}

var __java2goReferenceTypeRegistrationLogger = func() bool {
	stdjava.RegisterJavaType(stdjava.TypeID("com.acme.common.Logger"), stdjava.ObjectTypeID)
	return true
}()

func (lr *Logger) JavaDynamicTypeID() stdjava.TypeID {
	return stdjava.TypeID("com.acme.common.Logger")
}
func NewLogger() *Logger {
	return NewLoggerJava2goExecution(stdjava.NewExecution())
}
func NewLoggerJava2goExecution(__java2goExecution *stdjava.Execution) *Logger {
	lr := new(Logger)
	return lr
}
func Log(msg string) {
	LogJava2goExecution(stdjava.NewExecution(), msg)
}
func LogJava2goExecution(__java2goExecution *stdjava.Execution, msg string) {
	fmt.Println(stdjava.StringValueOf(msg))
}
