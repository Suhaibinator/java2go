package app

import (
	common "com/acme/common"
	domain "com/acme/domain"
	fmt "fmt"
	os "os"
)

type MainApp struct {
}

func Main() {
	args := os.Args
	_ = args
	task := domain.NewParseTask("alpha")
	identity := common.NewMapperFuncAdapter(func(v string) string {
		return v
	})
	count := Execute(task, identity)
	mode := common.ModeValueOf("FAST")
	for _, each := range common.ModeValues() {
		common.Log(each.Name())
	}
	common.Log(task.Name())
	common.Log(fmt.Sprintf("%v%v%v", "", count, mode.Ordinal()))
	common.Log(fmt.Sprintf("%v%v", "", GuardedValue()))
	common.Log(fmt.Sprintf("%v%v", "", GuardedFinallyOverride()))
	common.Log(fmt.Sprintf("%v%v", "", GuardedCatchFinallyOverride()))
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
				__java2goRecovered_792 = r
				__java2goDidPanic_792 = true
			}
		}()
		common.Log(fmt.Sprintf("%v%v", "", GuardedFinallyPanicOverride()))
	}()
	if __java2goDidPanic_792 && true {
		__java2goCatchHandled_792 = true
		e := __java2goRecovered_792
		_ = e
		func() {
			common.Log("PANIC_FINALLY")
		}()
	}
	if __java2goShouldReturn_792 {
		return
	}
	if __java2goDidPanic_792 && !__java2goCatchHandled_792 {
		panic(__java2goRecovered_792)
	}
	common.Log(GuardedResourceOrder())
}
