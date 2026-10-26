package stdjava

import (
	_ "embed"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

//go:embed testdata/stream_list_interference_jdk21.json
var streamListInterferenceJDK []byte

type streamListInterferenceExpected struct {
	Events []string `json:"events"`
	Root   string   `json:"root"`
	Fails  bool     `json:"fails"`
}

func streamListInterferenceText(values []*Integer) string {
	items := make([]string, len(values))
	for index, value := range values {
		if value == nil {
			items[index] = "null"
		} else {
			items[index] = strconv.FormatInt(int64(value.IntValue()), 10)
		}
	}
	return "[" + strings.Join(items, ", ") + "]"
}

func streamListInterferenceCase(t *testing.T, name string) {
	t.Helper()
	var fixture struct {
		Oracle string                                               `json:"oracle_sha256"`
		Seeds  map[string]map[string]streamListInterferenceExpected `json:"seeds"`
	}
	if err := json.Unmarshal(streamListInterferenceJDK, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Oracle != "820038f0c82da0039f884ee55b86955c90faae3c1b4ed2d59cfbf1d902a4c559" {
		t.Fatal("unexpected pinned JDK oracle")
	}
	for _, seed := range []int32{0, 7, 28} {
		t.Run(strconv.FormatInt(int64(seed), 10), func(t *testing.T) {
			e := NewExecution()
			root := NewListFrom(BoxInteger(seed), BoxInteger(seed+2), BoxInteger(seed+1), BoxInteger(seed+9))
			source := root
			if strings.HasPrefix(name, "sub-") {
				source = root.SubList(1, 3)
			}
			var events []string
			var result []*Integer
			allocated := false
			pipeline := streamPeek(streamFromIterable[*Integer](source), func(got *Execution, value *Integer) {
				if got != e {
					t.Fatal("wrong peek Execution")
				}
				text := "null"
				if value != nil {
					text = strconv.FormatInt(int64(value.IntValue()), 10)
				}
				events = append(events, "peek:"+text)
			})
			if strings.HasSuffix(name, "-sorted") {
				pipeline = streamSorted(pipeline, func(got *Execution, a, b *Integer) int32 {
					if got != e || allocated {
						t.Fatal("comparison after actual-A or wrong Execution")
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
			if strings.HasSuffix(name, "-limit") {
				pipeline = streamLimit(pipeline, 1)
			}
			failure := streamPhase1Recover(func() {
				streamClaimAndEvaluate(e, pipeline, func() streamSink[*Integer] {
					return &streamSinkFuncs[*Integer]{
						begin: func(int64) {
							allocated = true
							events = append(events, "actual-A")
							switch name {
							case "root-remove", "root-remove-limit", "sub-root-remove":
								root.RemoveAt(0)
							case "root-clear", "root-clear-sorted":
								root.Clear()
							case "sub-root-add":
								root.Add(BoxInteger(seed + 7))
							case "sub-clear":
								source.Clear()
							case "sub-remove":
								source.RemoveAt(0)
							case "sub-set":
								source.Set(0, BoxInteger(seed+8))
							default:
								t.Fatal("unknown case", name)
							}
						},
						accept: func(value *Integer) {
							text := "null"
							if value != nil {
								text = strconv.FormatInt(int64(value.IntValue()), 10)
							}
							events = append(events, "accumulate:"+text)
							result = append(result, value)
						},
					}
				})
				events = append(events, "result:"+streamListInterferenceText(result))
			})
			want, ok := fixture.Seeds[strconv.FormatInt(int64(seed), 10)][name]
			if !ok {
				t.Fatal("missing exact frozen case")
			}
			if !reflect.DeepEqual(events, want.Events) {
				t.Errorf("source events\ngot  %v\nwant %v", events, want.Events)
			}
			if got := streamListInterferenceText(root.Slice()); got != want.Root {
				t.Errorf("root state got %s want %s", got, want.Root)
			}
			if want.Fails {
				if _, ok := failure.(ConcurrentModificationException); !ok {
					t.Errorf("failure %T want ConcurrentModificationException", failure)
				}
			} else if failure != nil {
				t.Errorf("unexpected failure %T", failure)
			}
			if reuse := streamPhase1Recover(func() { streamCountExecution(e, pipeline) }); reuse == nil {
				t.Error("failed/completed terminal reusable")
			} else if _, ok := reuse.(IllegalStateException); !ok {
				t.Errorf("reuse %T want IllegalStateException", reuse)
			}
		})
	}
}

func TestStreamListInterferenceRootRemove(t *testing.T) { streamListInterferenceCase(t, "root-remove") }
func TestStreamListInterferenceRootClear(t *testing.T)  { streamListInterferenceCase(t, "root-clear") }
func TestStreamListInterferenceRootRemoveLimit(t *testing.T) {
	streamListInterferenceCase(t, "root-remove-limit")
}
func TestStreamListInterferenceRootClearSorted(t *testing.T) {
	streamListInterferenceCase(t, "root-clear-sorted")
}
func TestStreamListInterferenceSubRootAdd(t *testing.T) {
	streamListInterferenceCase(t, "sub-root-add")
	t.Run("root-change-after-construction-runs-A-and-old-fence", func(t *testing.T) {
		e := NewExecution()
		root := NewListFrom(BoxInteger(0), BoxInteger(2), BoxInteger(1), BoxInteger(9))
		view := root.SubList(1, 3)
		source := streamFromIterable[*Integer](view)
		root.Add(BoxInteger(7))
		allocated := 0
		var values []*Integer
		failure := streamPhase1Recover(func() {
			streamClaimAndEvaluate(e, source, func() streamSink[*Integer] {
				return &streamSinkFuncs[*Integer]{begin: func(int64) { allocated++ }, accept: func(v *Integer) { values = append(values, v) }}
			})
		})
		if _, ok := failure.(ConcurrentModificationException); !ok || allocated != 1 || streamListInterferenceText(values) != "[2, 1]" {
			t.Fatal("JDK post-construction timing", failure, allocated, values)
		}
	})
	t.Run("root-change-before-construction-fails-before-A", func(t *testing.T) {
		root := NewListFrom(BoxInteger(0), BoxInteger(2), BoxInteger(1), BoxInteger(9))
		view := root.SubList(1, 3)
		root.Add(BoxInteger(7))
		failure := streamPhase1Recover(func() { streamFromIterable[*Integer](view) })
		if _, ok := failure.(ConcurrentModificationException); !ok {
			t.Fatal("JDK construction timing", failure)
		}
	})
	t.Run("nested-view-absolute-offset", func(t *testing.T) {
		e := NewExecution()
		root := NewListFrom(BoxInteger(0), BoxInteger(1), BoxInteger(2), BoxInteger(3), BoxInteger(4))
		view := root.SubList(1, 5).SubList(1, 3)
		if got := streamListInterferenceText(streamToSliceExecution(e, streamFromIterable[*Integer](view))); got != "[2, 3]" {
			t.Fatal(got)
		}
	})
}
func TestStreamListInterferenceSubRootRemove(t *testing.T) {
	streamListInterferenceCase(t, "sub-root-remove")
}
func TestStreamListInterferenceSubClear(t *testing.T)  { streamListInterferenceCase(t, "sub-clear") }
func TestStreamListInterferenceSubRemove(t *testing.T) { streamListInterferenceCase(t, "sub-remove") }
func TestStreamListInterferenceSubSet(t *testing.T)    { streamListInterferenceCase(t, "sub-set") }
