package stdjava_test

import (
	"fmt"
	j "github.com/NickyBoy89/java2go/stdjava"
	"strings"
	"testing"
)

type campaignReentrantCell struct{ name string }

var campaignReentrantAction func()
var campaignReentrantDone bool

func (c *campaignReentrantCell) String() string { return c.name }
func (c *campaignReentrantCell) CompareTo(other *campaignReentrantCell) int32 {
	if !campaignReentrantDone {
		campaignReentrantDone = true
		campaignReentrantAction()
	}
	return 0
}
func campaignReentrantFailure(f func()) (result string) {
	result = "none"
	defer func() {
		if v := recover(); v != nil {
			v = j.NormalizePanic(v)
			result = "unknown"
			for _, n := range []string{"NoSuchElementException", "ConcurrentModificationException", "UnsupportedOperationException", "IndexOutOfBoundsException"} {
				if j.CaughtAs(v, n) {
					result = n
					break
				}
			}
		}
	}()
	f()
	return
}
func campaignReentrantSize(v *j.List[*campaignReentrantCell]) (result string) {
	fail := campaignReentrantFailure(func() { result = fmt.Sprintf("size=%d", v.Size()) })
	if fail != "none" {
		return fail
	}
	return
}
func campaignReentrantProbe(mode string, natural bool) string {
	base := j.NewListFrom(&campaignReentrantCell{"p"}, &campaignReentrantCell{"b"}, &campaignReentrantCell{"a"}, &campaignReentrantCell{"q"})
	if mode == "own-set-fixed" {
		base = j.AsList(&campaignReentrantCell{"p"}, &campaignReentrantCell{"b"}, &campaignReentrantCell{"a"}, &campaignReentrantCell{"q"})
	}
	if mode == "own-set-erased" {
		j.CollectionListSetExecution(j.NewExecution(), base, 0, base.Get(0))
	}
	view := base.SubList(1, 3)
	campaignReentrantDone = false
	campaignReentrantAction = func() {
		if strings.HasPrefix(mode, "own-set") {
			view.Set(0, &campaignReentrantCell{"X"})
		} else if mode == "own-clear" {
			view.Clear()
		} else if mode == "own-clear-grow" {
			view.Clear()
			view.Add(&campaignReentrantCell{"z"})
		} else if mode == "own-grow" {
			view.Add(&campaignReentrantCell{"z"})
		} else if mode == "external-grow" {
			base.Add(&campaignReentrantCell{"z"})
		} else if mode == "external-clear" {
			base.Clear()
		}
	}
	failed := campaignReentrantFailure(func() {
		if natural {
			j.CollectionSortOrderedExecution(j.NewExecution(), view)
		} else {
			j.SortWith(view, func(a, b *campaignReentrantCell) int32 { return a.CompareTo(b) })
		}
	})
	return fmt.Sprintf("%s|%s|%s|%s", mode, failed, base.String(), campaignReentrantSize(view))
}

func TestCampaignSubListReentrantSortJVMObservations(t *testing.T) {
	expected := []string{
		"own-set-mutable|none|[p, b, a, q]|size=2",
		"own-set-fixed|none|[p, b, a, q]|size=2",
		"own-set-erased|none|[p, b, a, q]|size=2",
		"own-clear|NoSuchElementException|[p, q]|size=0",
		"own-clear-grow|NoSuchElementException|[p, b, q]|size=1",
		"own-grow|none|[p, b, a, z, q]|size=3",
		"external-grow|ConcurrentModificationException|[p, b, a, q, z]|ConcurrentModificationException",
		"external-clear|ConcurrentModificationException|[]|ConcurrentModificationException",
	}
	for _, natural := range []bool{false, true} {
		for _, want := range expected {
			mode := strings.SplitN(want, "|", 2)[0]
			t.Run(fmt.Sprintf("natural=%v/%s", natural, mode), func(t *testing.T) {
				if got := campaignReentrantProbe(mode, natural); got != want {
					t.Fatalf("JDK21 observation: got %q; want %q", got, want)
				}
			})
		}
	}
}
