package stdjava

import (
	"encoding/json"
	"os"
	"slices"
	"strconv"
	"sync"
	"testing"
)

func TestURI126FrozenJDKGrammar(t *testing.T) {
	var rows []struct {
		Input, String, ASCII, Message, Reason []uint16
		Exception                             string
		Index                                 int32
		CauseNull                             bool `json:"cause_null"`
	}
	raw, err := os.ReadFile("testdata/uri126_jdk_grammar.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 41 {
		t.Fatalf("must retain all original JDK grammar observations: %d", len(rows))
	}
	execution := NewExecution()
	for index, row := range rows {
		t.Run(strconv.Itoa(index), func(t *testing.T) {
			input := NewJavaStringUTF16(row.Input)
			defer func() {
				observed := recover()
				if row.Exception == "" {
					if observed != nil {
						t.Fatalf("unexpected failure: %v", observed)
					}
					return
				}
				failure, ok := observed.(URISyntaxException)
				if !ok {
					t.Fatalf("expected syntax failure %s, got %T %v", row.Exception, observed, observed)
				}
				if failure.JavaDynamicTypeID() != TypeID(row.Exception) || failure.GetIndex() != row.Index || (GetCause(failure) == nil) != row.CauseNull || !slices.Equal(failure.GetInput().UTF16Copy(), row.Input) || !slices.Equal(failure.GetReason().UTF16Copy(), row.Reason) || !slices.Equal(JavaThrowableMessageExecution(execution, failure).UTF16Copy(), row.Message) {
					t.Fatalf("syntax observation differs from pinned JDK: %+v", failure)
				}
			}()
			value := NewJavaURIStringExecution(execution, input)
			if row.Exception != "" {
				t.Fatal("invalid syntax accepted")
			}
			text := value.StringJava2goExecution(execution)
			ascii := value.ToASCIIStringJava2goExecution(execution)
			if text == nil || ascii == nil || !slices.Equal(text.UTF16Copy(), row.String) || !slices.Equal(ascii.UTF16Copy(), row.ASCII) {
				t.Fatalf("UTF16/ASCII differs from pinned JDK: %v %v", text, ascii)
			}
		})
	}
}

func TestURI126ConcurrentASCIIAndIdentity(t *testing.T) {
	input := NewJavaStringUTF16([]uint16{'u', 'r', 'n', ':', 'e', 0x301, ':', 0xd83d, 0xde00})
	value := NewJavaURIStringExecution(NewExecution(), input)
	want := uriRuntimeText("urn:%C3%A9:%F0%9F%98%80").UTF16Copy()
	var workers sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			execution := NewExecution()
			for iteration := 0; iteration < 50; iteration++ {
				if value.StringJava2goExecution(execution) != input {
					t.Error("String constructor identity lost")
				}
				got := value.ToASCIIStringJava2goExecution(execution)
				if got == nil || !slices.Equal(got.UTF16Copy(), want) {
					t.Error("concurrent ASCII differs")
				}
				if value.JavaDynamicTypeID() != "java.net.URI" || !ClassLiteral("java.io.Serializable").IsInstance(value) || !ClassLiteral("java.lang.Comparable").IsInstance(value) {
					t.Error("nominal identity lost")
				}
			}
		}()
	}
	workers.Wait()
}
