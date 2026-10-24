package stdjava

import "testing"

// This control uses only pre-existing APIs, so the baseline RED is an executed
// dispatch failure rather than a missing-new-API compiler error.
func TestThreadSourceExistingNominalRunFrontier(t *testing.T) {
	const id TypeID = "threadsource.ExistingNominalRun"
	RegisterJavaType(id, ThreadTypeID)
	RegisterJavaSourceType(id)
	caller := NewExecution()
	calls := 0
	native := NewThread(func(execution *Execution) {
		if execution != caller {
			t.Error("native target lost caller execution")
		}
		calls++
	})
	source := &threadSourceNominalFrontier{native: native}
	source.info = NewObjectInfo(id, func(requested TypeID) any {
		switch requested {
		case ThreadTypeID, RunnableTypeID:
			return native
		case id, ObjectTypeID:
			return source
		}
		return nil
	})
	RunRunnableExecution(caller, source)
	if source.publicCalls != 0 || calls != 1 {
		t.Fatalf("nominal source Thread selected public Go shape: public=%d native-target=%d", source.publicCalls, calls)
	}
}

type threadSourceNominalFrontier struct {
	native      *Thread
	info        *ObjectInfo
	publicCalls int
}

func (source *threadSourceNominalFrontier) JavaObjectInfo() *ObjectInfo { return source.info }
func (source *threadSourceNominalFrontier) Run()                        { source.publicCalls++ }
