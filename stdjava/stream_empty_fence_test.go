package stdjava

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

//go:embed testdata/stream_empty_fence_jdk21.json
var streamEmptyFenceJDK []byte

func streamEmptyFenceCase(t *testing.T, r *streamVBRecorder, name string, seed int32, erased bool) {
	e := NewExecution()
	shape := strings.SplitN(name, "-", 2)[0]
	root := NewList[*Integer]()
	if shape != "root" {
		root = NewListFrom(BoxInteger(seed+5), BoxInteger(seed+6))
	}
	streamVBPromote(root, erased)
	selected := root
	switch shape {
	case "root":
	case "direct":
		selected = root.SubList(1, 1)
	case "nested":
		selected = root.SubList(0, 2).SubList(1, 1)
	default:
		t.Fatal(shape)
	}
	source := streamPeek(streamFromIterable[*Integer](selected), func(got *Execution, v *Integer) {
		if got != e {
			t.Fatal("wrong Execution")
		}
		r.events = append(r.events, "peek:"+streamVBValue(v))
	})
	rest := strings.TrimPrefix(name, shape+"-")
	mutation := ""
	for _, v := range []string{"root-add", "root-clear", "own-add"} {
		if strings.HasSuffix(rest, "-"+v) {
			mutation = v
		}
	}
	mode := strings.TrimSuffix(rest, "-"+mutation)
	if strings.HasPrefix(mode, "sorted") {
		source = streamSorted(source, func(*Execution, *Integer, *Integer) int32 { t.Fatal("empty comparator called"); return 0 })
	}
	if strings.HasSuffix(mode, "limit0") {
		source = streamLimit(source, 0)
	}
	if strings.HasSuffix(mode, "limit1") {
		source = streamLimit(source, 1)
	}
	allocated := 0
	r.check(name+"-collect", func() {
		var result []*Integer
		streamClaimAndEvaluate(e, source, func() streamSink[*Integer] {
			return &streamSinkFuncs[*Integer]{begin: func(int64) {
				allocated++
				r.events = append(r.events, fmt.Sprintf("actual-A:%d", allocated), "mutation:"+mutation)
				switch mutation {
				case "root-add":
					root.Add(BoxInteger(seed + 7))
				case "root-clear":
					root.Clear()
				case "own-add":
					selected.Add(BoxInteger(seed + 7))
				default:
					t.Fatal(mutation)
				}
			}, accept: func(v *Integer) {
				r.events = append(r.events, "accumulate:"+streamVBValue(v))
				result = append(result, v)
			}}
		})
		r.events = append(r.events, "result:"+streamListInterferenceText(result))
	})
	r.check(name+"-A-count", func() { r.events = append(r.events, strconv.Itoa(allocated)) })
	r.check(name+"-reuse", func() { r.events = append(r.events, "count:"+strconv.FormatInt(streamCountExecution(e, source), 10)) })
	r.check(name+"-close", func() { streamCloseExecution(e, source) })
	r.check(name+"-root", func() { r.events = append(r.events, streamListInterferenceText(root.Slice())) })
}

func TestStreamEmptyFenceJDK54(t *testing.T) {
	var fixture struct {
		Oracle string              `json:"oracle_sha256"`
		Seeds  map[string][]string `json:"seeds"`
	}
	if err := json.Unmarshal(streamEmptyFenceJDK, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Oracle != "da1381a0472ae1c4b91c8a7d78ee34621a14bef52c92e3ddd5ab82aff28f32f7" {
		t.Fatal("wrong actualJDK oracle")
	}
	for _, seed := range []int32{0, 7, 28} {
		want := map[string]string{}
		var names []string
		for _, line := range fixture.Seeds[strconv.Itoa(int(seed))] {
			i := strings.Index(line, "=")
			if i < 0 {
				t.Fatal("malformed oracle", line)
			}
			label, value := line[:i], line[i+1:]
			want[label] = value
			if strings.HasSuffix(label, "-collect") {
				names = append(names, strings.TrimSuffix(label, "-collect"))
			}
			if strings.HasSuffix(label, "-A-count") || strings.HasSuffix(label, "-root") {
				want[label] = "ok|events=[" + value + "]"
			}
		}
		if len(names) != 54 {
			t.Fatal("case count", len(names))
		}
		for _, erased := range []bool{false, true} {
			for _, name := range names {
				t.Run(fmt.Sprintf("seed%d/erased%t/%s", seed, erased, name), func(t *testing.T) { streamEmptyFenceCase(t, &streamVBRecorder{t: t, want: want}, name, seed, erased) })
			}
		}
	}
}
