package app

import (
	common "com/acme/common"
	domain "com/acme/domain"
	fmt "fmt"
	stdjava "github.com/NickyBoy89/java2go/stdjava"
	os "os"
)

type MainApp struct {
}

var __java2goReferenceTypeRegistrationMainApp = func() bool {
	stdjava.RegisterJavaType(stdjava.TypeID("com.acme.app.MainApp"), stdjava.ObjectTypeID)
	return true
}()

func (mp *MainApp) JavaDynamicTypeID() stdjava.TypeID {
	return stdjava.TypeID("com.acme.app.MainApp")
}
func NewMainApp() *MainApp {
	return NewMainAppJava2goExecution(stdjava.NewExecution())
}
func NewMainAppJava2goExecution(__java2goExecution *stdjava.Execution) *MainApp {
	mp := new(MainApp)
	return mp
}
func Main() {
	MainJava2goExecution(stdjava.NewExecution())
}
func MainJava2goExecution(__java2goExecution *stdjava.Execution) {
	args := os.Args
	_ = args
	task := domain.NewParseTaskJava2goExecution(__java2goExecution, "alpha")
	identity := common.NewMapperFuncAdapterJava2goExecution[string, string](func(__java2goExecution *stdjava.Execution, v string) string {
		return v
	})
	count := int32(ExecuteJava2goExecution(__java2goExecution, task, identity))
	mode := common.ModeValueOf("FAST")
	for _, each := range common.ModeValues() {
		common.LogJava2goExecution(__java2goExecution, each.Name())
	}
	common.LogJava2goExecution(__java2goExecution, func() string {
		__java2goInvocationReceiver := task
		__java2goExecutionReceiver, __java2goHasExecutionReceiver := interface {
		}(__java2goInvocationReceiver).(domain.TaskJava2goExecution)
		if __java2goHasExecutionReceiver {
			return __java2goExecutionReceiver.NameJava2goExecution(__java2goExecution)
		}
		return __java2goInvocationReceiver.Name()
	}())
	common.LogJava2goExecution(__java2goExecution, fmt.Sprintf("%v%v%v", "", count, mode.Ordinal()))
	common.LogJava2goExecution(__java2goExecution, fmt.Sprintf("%v%v", "", GuardedValueJava2goExecution(__java2goExecution)))
	common.LogJava2goExecution(__java2goExecution, fmt.Sprintf("%v%v", "", GuardedFinallyOverrideJava2goExecution(__java2goExecution)))
	common.LogJava2goExecution(__java2goExecution, fmt.Sprintf("%v%v", "", GuardedCatchFinallyOverrideJava2goExecution(__java2goExecution)))
	var (
		__java2goRecovered_792 interface {
		}
		__java2goDidPanic_792     bool
		__java2goCatchHandled_792 bool
		__java2goShouldReturn_792 bool
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				__java2goRecovered_792 = stdjava.NormalizePanic(r)
				__java2goDidPanic_792 = true
				__java2goShouldReturn_792 = false
			}
		}()
		common.LogJava2goExecution(__java2goExecution, fmt.Sprintf("%v%v", "", GuardedFinallyPanicOverrideJava2goExecution(__java2goExecution)))
	}()
	if __java2goDidPanic_792 && stdjava.CaughtAs(__java2goRecovered_792, "Exception") {
		__java2goCatchHandled_792 = true
		e := __java2goRecovered_792
		_ = e
		func() {
			common.LogJava2goExecution(__java2goExecution, "PANIC_FINALLY")
		}()
	}
	if __java2goShouldReturn_792 {
		return
	}
	if __java2goDidPanic_792 && !__java2goCatchHandled_792 {
		panic(__java2goRecovered_792)
	}
	common.LogJava2goExecution(__java2goExecution, GuardedResourceOrderJava2goExecution(__java2goExecution))
}
