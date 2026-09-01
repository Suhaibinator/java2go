package app

import (
	common "com/acme/common"
	domain "com/acme/domain"
)

type Pipeline struct {
}

func Execute(task domain.TaskI, mapper common.Mapper[string, string]) int32 {
	out := mapper.Map(task.Name())
	if func() bool {
		_, ok := any(task).(*domain.ParseTask)
		return ok
	}() {
		return int32(len(out))
	}
	return 0
}
func GuardedValue() int32 {
	var total int32
	total = 0
	var (
		__java2goRecovered_456	interface {
		}
		__java2goDidPanic_456		bool
		__java2goCatchHandled_456	bool
		__java2goShouldReturn_456	bool
		__java2goReturnValue_456	int32
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				__java2goRecovered_456 = r
				__java2goDidPanic_456 = true
			}
		}()
		var denom int32
		denom = 0
		total += 10 / denom
	}()
	if __java2goDidPanic_456 && true {
		__java2goCatchHandled_456 = true
		e := __java2goRecovered_456
		_ = e
		func() {
			total += 50
		}()
	}
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
	var (
		__java2goRecovered_738	interface {
		}
		__java2goDidPanic_738		bool
		__java2goCatchHandled_738	bool
		__java2goShouldReturn_738	bool
		__java2goReturnValue_738	int32
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				__java2goRecovered_738 = r
				__java2goDidPanic_738 = true
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
	var (
		__java2goRecovered_889	interface {
		}
		__java2goDidPanic_889		bool
		__java2goCatchHandled_889	bool
		__java2goShouldReturn_889	bool
		__java2goReturnValue_889	int32
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				__java2goRecovered_889 = r
				__java2goDidPanic_889 = true
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
	if __java2goDidPanic_889 && true {
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
	var (
		__java2goRecovered_1147	interface {
		}
		__java2goDidPanic_1147		bool
		__java2goCatchHandled_1147	bool
		__java2goShouldReturn_1147	bool
		__java2goReturnValue_1147	int32
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				__java2goRecovered_1147 = r
				__java2goDidPanic_1147 = true
			}
		}()
		{
			__java2goReturnValue_1147 = 1
			__java2goShouldReturn_1147 = true
			return
		}
	}()
	if __java2goDidPanic_1147 && true {
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
	trace := NewCloseTrace()
	var (
		__java2goRecovered_1445	interface {
		}
		__java2goDidPanic_1445		bool
		__java2goCatchHandled_1445	bool
		__java2goShouldReturn_1445	bool
		__java2goReturnValue_1445	string
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				__java2goRecovered_1445 = r
				__java2goDidPanic_1445 = true
			}
		}()
		first := NewResourceToken(trace, "A")
		defer func() {
			if first != nil {
				first.Close()
			}
		}()
		second := NewResourceToken(trace, "B")
		defer func() {
			if second != nil {
				second.Close()
			}
		}()
		trace.Append("X")
	}()
	if __java2goShouldReturn_1445 {
		return __java2goReturnValue_1445
	}
	if __java2goDidPanic_1445 && !__java2goCatchHandled_1445 {
		panic(__java2goRecovered_1445)
	}
	return trace.Get()
}
