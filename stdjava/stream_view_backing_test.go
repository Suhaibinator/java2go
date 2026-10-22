package stdjava

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

//go:embed testdata/stream_view_backing_jdk21.json
var streamVBJDK []byte

type streamVBRecorder struct {
	t      *testing.T
	want   map[string]string
	events []string
}

func (r *streamVBRecorder) check(label string, action func()) {
	r.t.Helper()
	r.events = nil
	failure := streamPhase1Recover(action)
	outcome := "ok"
	if failure != nil {
		switch failure.(type) {
		case ConcurrentModificationException:
			outcome = "java.util.ConcurrentModificationException"
		case IllegalStateException:
			outcome = "java.lang.IllegalStateException"
		default:
			r.t.Fatalf("%s unexpected failure %T", label, failure)
		}
	}
	raw, ok := r.want[label]
	if !ok {
		r.t.Fatal("missing JDK label", label)
	}
	marker := "|events="
	i := strings.Index(raw, marker)
	if i < 0 {
		r.t.Fatal("malformed oracle", raw)
	}
	expectedOutcome := raw[:i]
	if strings.HasPrefix(expectedOutcome, "java.") {
		expectedOutcome = strings.SplitN(expectedOutcome, ":", 2)[0]
	}
	// The full actual JDK comparator trace remains frozen. Core stable-sort trace
	// equivalence is an independent original-Supplier28 gate, not this source gate.
	expectedEvents := raw[i+len(marker):]
	if strings.Contains(expectedEvents, "compare:") {
		parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(expectedEvents, "["), "]"), ", ")
		kept := parts[:0]
		for _, v := range parts {
			if !strings.HasPrefix(v, "compare:") {
				kept = append(kept, v)
			}
		}
		expectedEvents = "[" + strings.Join(kept, ", ") + "]"
	}
	gotEvents := "[" + strings.Join(r.events, ", ") + "]"
	if outcome != expectedOutcome || gotEvents != expectedEvents {
		r.t.Errorf("%s\ngot  %s|events=%s\nwant %s|events=%s", label, outcome, gotEvents, expectedOutcome, expectedEvents)
	}
}
func streamVBValue(value *Integer) string {
	if value == nil {
		return "null"
	}
	return strconv.FormatInt(int64(value.IntValue()), 10)
}
func streamVBRoot(seed int32, values ...int32) *List[*Integer] {
	r := NewList[*Integer]()
	for _, v := range values {
		r.Add(BoxInteger(seed + v))
	}
	return r
}
func streamVBPromote(root *List[*Integer], erased bool) {
	if erased {
		root.promoteCollectionStorage()
	}
}
func streamVBView(t *testing.T, r *streamVBRecorder, name string, seed int32, erased bool) {
	e := NewExecution()
	root := streamVBRoot(seed, 0, 4, 2, 1, 9, 8)
	streamVBPromote(root, erased)
	view := root.SubList(2, 4)
	if strings.HasPrefix(name, "view-nested-") {
		view = root.SubList(1, 5).SubList(1, 3)
	}
	before := strings.Contains(name, "-before-build-")
	mutation := ""
	// The name's initial view- prefix is not a mutation suffix.
	for _, v := range []string{"root-add", "root-remove", "root-clear", "view-add", "view-remove", "view-clear"} {
		if strings.HasSuffix(name, "-"+v) {
			mutation = v
		}
	}
	mutate := func() {
		r.events = append(r.events, "mutation:"+mutation)
		switch mutation {
		case "root-add":
			root.Add(BoxInteger(seed + 7))
		case "root-remove":
			root.RemoveAt(0)
		case "root-clear":
			root.Clear()
		case "view-add":
			view.Add(BoxInteger(seed + 7))
		case "view-remove":
			view.RemoveAt(0)
		case "view-clear":
			view.Clear()
		default:
			t.Fatal("unknown mutation", mutation)
		}
	}
	var source Stream[*Integer]
	created := false
	allocated := 0
	if before {
		r.check(name+"-mutation", mutate)
	}
	r.check(name+"-build", func() {
		source = streamPeek(streamFromIterable[*Integer](view), func(got *Execution, v *Integer) {
			if got != e {
				t.Fatal("wrong Execution")
			}
			r.events = append(r.events, "peek:"+streamVBValue(v))
		})
		created = true
		r.events = append(r.events, "stream-created")
	})
	if !created {
		r.check(name+"-A-count", func() { r.events = append(r.events, strconv.Itoa(allocated)) })
		r.check(name+"-root", func() { r.events = append(r.events, streamListInterferenceText(root.Slice())) })
		return
	}
	if !before {
		r.check(name+"-mutation", mutate)
	}
	r.check(name+"-collect", func() {
		var values []*Integer
		streamClaimAndEvaluate(e, source, func() streamSink[*Integer] {
			return &streamSinkFuncs[*Integer]{begin: func(int64) { allocated++; r.events = append(r.events, fmt.Sprintf("actual-A:%d", allocated)) }, accept: func(v *Integer) {
				r.events = append(r.events, "accumulate:"+streamVBValue(v))
				values = append(values, v)
			}}
		})
		r.events = append(r.events, "result:"+streamListInterferenceText(values))
	})
	r.check(name+"-A-count", func() { r.events = append(r.events, strconv.Itoa(allocated)) })
	r.check(name+"-reuse", func() { streamCountExecution(e, source) })
	streamCloseExecution(e, source)
	r.check(name+"-root", func() { r.events = append(r.events, streamListInterferenceText(root.Slice())) })
}
func streamVBBulk(t *testing.T, r *streamVBRecorder, name string, seed int32, erased bool) {
	e := NewExecution()
	root := NewListFrom(BoxInteger(seed+2), BoxInteger(seed+1), BoxInteger(seed+3))
	streamVBPromote(root, erased)
	aPhase := strings.HasPrefix(name, "bulk-A-")
	mutation := ""
	for _, v := range []string{"grow-set", "set", "clear", "remove-next"} {
		if strings.Contains(name, "-"+v) {
			mutation = v
			break
		}
	}
	mutate := func() {
		r.events = append(r.events, "mutation:"+mutation)
		switch mutation {
		case "grow-set":
			for i := int32(0); i < 64; i++ {
				root.Add(BoxInteger(seed + 100 + i))
			}
			root.Set(1, BoxInteger(99))
		case "set":
			root.Set(1, BoxInteger(99))
		case "clear":
			root.Clear()
		case "remove-next":
			root.RemoveAt(1)
		default:
			t.Fatal(mutation)
		}
	}
	var source Stream[*Integer]
	peeked := false
	allocated := 0
	r.check(name+"-build", func() {
		source = streamPeek(streamFromIterable[*Integer](root), func(got *Execution, v *Integer) {
			if got != e {
				t.Fatal("wrong peek Execution")
			}
			r.events = append(r.events, "peek:"+streamVBValue(v))
			if !aPhase && !peeked {
				peeked = true
				mutate()
			}
		})
		if strings.Contains(name, "-sorted") {
			source = streamSorted(source, func(got *Execution, a, b *Integer) int32 {
				if got != e || allocated != 0 {
					t.Fatal("comparison phase")
				}
				if a == nil {
					if b == nil {
						return 0
					}
					return -1
				}
				if b == nil {
					return 1
				}
				return a.IntValue() - b.IntValue()
			})
		}
		if strings.HasSuffix(name, "-limit3") {
			source = streamLimit(source, 3)
		}
	})
	r.check(name+"-collect", func() {
		var values []*Integer
		streamClaimAndEvaluate(e, source, func() streamSink[*Integer] {
			return &streamSinkFuncs[*Integer]{begin: func(int64) {
				allocated++
				r.events = append(r.events, fmt.Sprintf("actual-A:%d", allocated))
				if aPhase {
					mutate()
				}
			}, accept: func(v *Integer) {
				r.events = append(r.events, "accumulate:"+streamVBValue(v))
				values = append(values, v)
			}}
		})
		r.events = append(r.events, "result:"+streamListInterferenceText(values))
	})
	r.check(name+"-A-count", func() { r.events = append(r.events, strconv.Itoa(allocated)) })
	r.check(name+"-reuse", func() { streamCountExecution(e, source) })
	streamCloseExecution(e, source)
	r.check(name+"-root", func() { r.events = append(r.events, streamListInterferenceText(root.Slice())) })
}
func TestStreamViewBackingJDK36(t *testing.T) {
	var fixture struct {
		Oracle string              `json:"oracle_sha256"`
		Seeds  map[string][]string `json:"seeds"`
	}
	if err := json.Unmarshal(streamVBJDK, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Oracle != "60ee003d9596130918d75ed99d277d1870cb086bc38f268022009ce941451a26" {
		t.Fatal("wrong oracle")
	}
	for _, seed := range []int32{0, 7, 28} {
		want := map[string]string{}
		var names []string
		for _, line := range fixture.Seeds[strconv.Itoa(int(seed))] {
			i := strings.Index(line, "=")
			if i < 0 {
				t.Fatal(line)
			}
			label, value := line[:i], line[i+1:]
			want[label] = value
			if strings.HasSuffix(label, "-build") {
				names = append(names, strings.TrimSuffix(label, "-build"))
			}
		}
		if len(names) != 36 {
			t.Fatal("case count", len(names))
		}
		// Count/root lines encode their value before |events, rather than as events.
		for label, raw := range want {
			if strings.HasSuffix(label, "-root") || strings.HasSuffix(label, "-A-count") {
				value := strings.SplitN(raw, "|events=", 2)[0]
				want[label] = "ok|events=[" + value + "]"
			}
		}
		for _, erased := range []bool{false, true} {
			for _, name := range names {
				t.Run(fmt.Sprintf("seed%d/erased%t/%s", seed, erased, name), func(t *testing.T) {
					r := &streamVBRecorder{t: t, want: want}
					if strings.HasPrefix(name, "view-") {
						streamVBView(t, r, name, seed, erased)
					} else {
						streamVBBulk(t, r, name, seed, erased)
					}
				})
			}
		}
	}
}

func TestStreamBackingClearPrerequisite(t *testing.T) {
	for _, erased := range []bool{false, true} {
		t.Run(fmt.Sprintf("erased%t", erased), func(t *testing.T) {
			input := []*Integer{BoxInteger(1), BoxInteger(2)}
			root := NewListFrom(input...)
			streamVBPromote(root, erased)
			typed := root.elements
			raw := root.erasedElements
			alias := root.Slice()
			before := root.modCount
			root.Clear()
			if root.Size() != 0 || root.modCount != before+1 {
				t.Fatal("clear state")
			}
			if input[0] == nil {
				t.Fatal("owned constructor changed input")
			}
			for _, v := range typed {
				if v != nil {
					t.Error("typed backing retained reference")
				}
			}
			for _, v := range raw {
				if v != nil {
					t.Error("erased backing retained reference")
				}
			}
			if !erased && alias[0] != nil {
				t.Error("Slice alias retained reference")
			}
			if erased && alias[0] == nil {
				t.Error("converted Slice snapshot changed")
			}
		})
	}
	t.Run("fixed-slice", func(t *testing.T) {
		input := []*Integer{BoxInteger(1)}
		root := AsList(input...)
		before := root.modCount
		failure := streamPhase1Recover(root.Clear)
		if _, ok := failure.(UnsupportedOperationException); !ok || input[0] == nil || root.modCount != before {
			t.Fatal("fixed slice mutated", failure)
		}
	})
	t.Run("fixed-array", func(t *testing.T) {
		array := NewReferenceArray(1, ObjectTypeID)
		ReferenceArraySet(array, 0, BoxInteger(1))
		root := AsListArray[*Integer](array, ObjectTypeID)
		before := root.modCount
		failure := streamPhase1Recover(root.Clear)
		if _, ok := failure.(UnsupportedOperationException); !ok || ReferenceArrayGet[*Integer](array, 0, ObjectTypeID) == nil || root.modCount != before {
			t.Fatal("fixed array mutated", failure)
		}
	})
}
