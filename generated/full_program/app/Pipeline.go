package app

import (
	common "com/acme/common"
	domain "com/acme/domain"
	stdjava "github.com/NickyBoy89/java2go/stdjava"
)

type Pipeline struct {
}

var __java2goReferenceTypeRegistrationPipeline = func() bool {
	stdjava.RegisterJavaType(stdjava.TypeID("com.acme.app.Pipeline"), stdjava.ObjectTypeID)
	return true
}()

func (pe *Pipeline) JavaDynamicTypeID() stdjava.TypeID {
	return stdjava.TypeID("com.acme.app.Pipeline")
}
func NewPipeline() *Pipeline {
	return NewPipelineJava2goExecution(stdjava.NewExecution())
}
func NewPipelineJava2goExecution(__java2goExecution *stdjava.Execution) *Pipeline {
	pe := new(Pipeline)
	return pe
}
func Execute(task domain.TaskI, mapper common.Mapper[string, string]) int32 {
	return ExecuteJava2goExecution(stdjava.NewExecution(), task, mapper)
}
func ExecuteJava2goExecution(__java2goExecution *stdjava.Execution, task domain.TaskI, mapper common.Mapper[string, string]) int32 {
	out := func() string {
		__java2goInvocationReceiver := stdjava.EvaluationValue(mapper)
		__java2goInvocationArg0 := func() string {
			__java2goInvocationReceiver := task
			__java2goExecutionReceiver, __java2goHasExecutionReceiver := interface {
			}(__java2goInvocationReceiver).(domain.TaskJava2goExecution)
			if __java2goHasExecutionReceiver {
				return __java2goExecutionReceiver.NameJava2goExecution(__java2goExecution)
			}
			return __java2goInvocationReceiver.Name()
		}()
		__java2goExecutionReceiver, __java2goHasExecutionReceiver := interface {
		}(__java2goInvocationReceiver).(common.MapperJava2goExecution[string, string])
		if __java2goHasExecutionReceiver {
			return __java2goExecutionReceiver.MapJava2goExecution(__java2goExecution, __java2goInvocationArg0)
		}
		return __java2goInvocationReceiver.Map(__java2goInvocationArg0)
	}()
	if stdjava.ObjectInstanceOf(task, stdjava.TypeID("com.acme.domain.ParseTask")) {
		return stdjava.StringLength(stdjava.StringRequireNonNull(out))
	}
	return 0
}
func GuardedValue() int32 {
	return GuardedValueJava2goExecution(stdjava.NewExecution())
}
func GuardedValueJava2goExecution(__java2goExecution *stdjava.Execution) int32 {
	var total int32
	total = 0
	var (
		__java2goRecovered_456 interface {
		}
		__java2goDidPanic_456     bool
		__java2goCatchHandled_456 bool
		__java2goShouldReturn_456 bool
		__java2goReturnValue_456  int32
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				__java2goRecovered_456 = stdjava.NormalizePanic(r)
				__java2goDidPanic_456 = true
				__java2goShouldReturn_456 = false
			}
		}()
		var denom int32
		denom = 0
		total += 10 / denom
	}()
	func() {
		defer func() {
			if catchPanic := recover(); catchPanic != nil {
				__java2goRecovered_456 = stdjava.NormalizePanic(catchPanic)
				__java2goDidPanic_456 = true
				__java2goShouldReturn_456 = false
				__java2goCatchHandled_456 = false
			}
		}()
		if __java2goDidPanic_456 && stdjava.CaughtAs(__java2goRecovered_456, "Exception") {
			__java2goCatchHandled_456 = true
			e := __java2goRecovered_456
			_ = e
			func() {
				total += 50
			}()
		}
	}()
	func() {
		total += 3
	}()
	if __java2goShouldReturn_456 {
		return __java2goReturnValue_456
	}
	if __java2goDidPanic_456 && !__java2goCatchHandled_456 {
		panic(__java2goRecovered_456)
	}
	return total
}
func GuardedFinallyOverride() int32 {
	return GuardedFinallyOverrideJava2goExecution(stdjava.NewExecution())
}
func GuardedFinallyOverrideJava2goExecution(__java2goExecution *stdjava.Execution) int32 {
	var (
		__java2goRecovered_738 interface {
		}
		__java2goDidPanic_738     bool
		__java2goCatchHandled_738 bool
		__java2goShouldReturn_738 bool
		__java2goReturnValue_738  int32
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				__java2goRecovered_738 = stdjava.NormalizePanic(r)
				__java2goDidPanic_738 = true
				__java2goShouldReturn_738 = false
			}
		}()
		{
			__java2goReturnValue_738 = 10
			__java2goShouldReturn_738 = true
			return
		}
	}()
	func() {
		{
			__java2goReturnValue_738 = 20
			__java2goShouldReturn_738 = true
			return
		}
	}()
	if __java2goShouldReturn_738 {
		return __java2goReturnValue_738
	}
	if __java2goDidPanic_738 && !__java2goCatchHandled_738 {
		panic(__java2goRecovered_738)
	}
	return 0
}
func GuardedCatchFinallyOverride() int32 {
	return GuardedCatchFinallyOverrideJava2goExecution(stdjava.NewExecution())
}
func GuardedCatchFinallyOverrideJava2goExecution(__java2goExecution *stdjava.Execution) int32 {
	var (
		__java2goRecovered_889 interface {
		}
		__java2goDidPanic_889     bool
		__java2goCatchHandled_889 bool
		__java2goShouldReturn_889 bool
		__java2goReturnValue_889  int32
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				__java2goRecovered_889 = stdjava.NormalizePanic(r)
				__java2goDidPanic_889 = true
				__java2goShouldReturn_889 = false
			}
		}()
		var denom int32
		denom = 0
		{
			__java2goReturnValue_889 = 10 / denom
			__java2goShouldReturn_889 = true
			return
		}
	}()
	func() {
		defer func() {
			if catchPanic := recover(); catchPanic != nil {
				__java2goRecovered_889 = stdjava.NormalizePanic(catchPanic)
				__java2goDidPanic_889 = true
				__java2goShouldReturn_889 = false
				__java2goCatchHandled_889 = false
			}
		}()
		if __java2goDidPanic_889 && stdjava.CaughtAs(__java2goRecovered_889, "Exception") {
			__java2goCatchHandled_889 = true
			e := __java2goRecovered_889
			_ = e
			func() {
				{
					__java2goReturnValue_889 = 7
					__java2goShouldReturn_889 = true
					return
				}
			}()
		}
	}()
	func() {
		{
			__java2goReturnValue_889 = 9
			__java2goShouldReturn_889 = true
			return
		}
	}()
	if __java2goShouldReturn_889 {
		return __java2goReturnValue_889
	}
	if __java2goDidPanic_889 && !__java2goCatchHandled_889 {
		panic(__java2goRecovered_889)
	}
	return 0
}
func GuardedFinallyPanicOverride() int32 {
	return GuardedFinallyPanicOverrideJava2goExecution(stdjava.NewExecution())
}
func GuardedFinallyPanicOverrideJava2goExecution(__java2goExecution *stdjava.Execution) int32 {
	var (
		__java2goRecovered_1147 interface {
		}
		__java2goDidPanic_1147     bool
		__java2goCatchHandled_1147 bool
		__java2goShouldReturn_1147 bool
		__java2goReturnValue_1147  int32
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				__java2goRecovered_1147 = stdjava.NormalizePanic(r)
				__java2goDidPanic_1147 = true
				__java2goShouldReturn_1147 = false
			}
		}()
		{
			__java2goReturnValue_1147 = 1
			__java2goShouldReturn_1147 = true
			return
		}
	}()
	func() {
		defer func() {
			if catchPanic := recover(); catchPanic != nil {
				__java2goRecovered_1147 = stdjava.NormalizePanic(catchPanic)
				__java2goDidPanic_1147 = true
				__java2goShouldReturn_1147 = false
				__java2goCatchHandled_1147 = false
			}
		}()
		if __java2goDidPanic_1147 && stdjava.CaughtAs(__java2goRecovered_1147, "Exception") {
			__java2goCatchHandled_1147 = true
			e := __java2goRecovered_1147
			_ = e
			func() {
				{
					__java2goReturnValue_1147 = 2
					__java2goShouldReturn_1147 = true
					return
				}
			}()
		}
	}()
	func() {
		var denom int32
		denom = 0
		{
			__java2goReturnValue_1147 = 3 / denom
			__java2goShouldReturn_1147 = true
			return
		}
	}()
	if __java2goShouldReturn_1147 {
		return __java2goReturnValue_1147
	}
	if __java2goDidPanic_1147 && !__java2goCatchHandled_1147 {
		panic(__java2goRecovered_1147)
	}
	return 0
}
func GuardedResourceOrder() string {
	return GuardedResourceOrderJava2goExecution(stdjava.NewExecution())
}
func GuardedResourceOrderJava2goExecution(__java2goExecution *stdjava.Execution) string {
	trace := NewCloseTraceJava2goExecution(__java2goExecution)
	var (
		__java2goRecovered_1445 interface {
		}
		__java2goDidPanic_1445     bool
		__java2goCatchHandled_1445 bool
		__java2goShouldReturn_1445 bool
		__java2goReturnValue_1445  string
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				__java2goRecovered_1445 = stdjava.NormalizePanic(r)
				__java2goDidPanic_1445 = true
				__java2goShouldReturn_1445 = false
			}
		}()
		first := NewResourceTokenJava2goExecution(__java2goExecution, trace, "A")
		defer stdjava.CloseResource(func() {
			if first != nil {
				__java2goCloseExecutionReceiver, __java2goHasCloseExecutionReceiver := interface {
				}(first).(interface {
					CloseJava2goExecution(*stdjava.Execution)
				})
				if __java2goHasCloseExecutionReceiver {
					__java2goCloseExecutionReceiver.CloseJava2goExecution(__java2goExecution)
					return
				}
				first.Close()
			}
		})
		second := NewResourceTokenJava2goExecution(__java2goExecution, trace, "B")
		defer stdjava.CloseResource(func() {
			if second != nil {
				__java2goCloseExecutionReceiver, __java2goHasCloseExecutionReceiver := interface {
				}(second).(interface {
					CloseJava2goExecution(*stdjava.Execution)
				})
				if __java2goHasCloseExecutionReceiver {
					__java2goCloseExecutionReceiver.CloseJava2goExecution(__java2goExecution)
					return
				}
				second.Close()
			}
		})
		trace.AppendJava2goExecution(__java2goExecution, "X")
	}()
	if __java2goShouldReturn_1445 {
		return __java2goReturnValue_1445
	}
	if __java2goDidPanic_1445 && !__java2goCatchHandled_1445 {
		panic(__java2goRecovered_1445)
	}
	return trace.GetJava2goExecution(__java2goExecution)
}
