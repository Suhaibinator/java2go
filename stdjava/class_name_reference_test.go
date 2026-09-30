package stdjava

import (
	"sync"
	"testing"
	"unicode/utf16"
)

func TestClassJavaNameConcurrentIdentity(t *testing.T) {
	class := ClassLiteral(StringTypeID)
	results := make([]*JavaString, 32)
	var tasks sync.WaitGroup
	for index := range results {
		tasks.Add(1)
		go func(index int) {
			defer tasks.Done()
			results[index] = ClassJavaName(class)
		}(index)
	}
	tasks.Wait()
	for _, result := range results {
		if result != results[0] {
			t.Fatal("concurrent name reads lost cached reference identity")
		}
	}
	if got := results[0].UTF16Copy(); string(utf16.Decode(got)) != "java.lang.String" {
		t.Fatal("class name content changed")
	}
}
