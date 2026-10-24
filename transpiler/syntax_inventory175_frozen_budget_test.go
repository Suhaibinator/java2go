package transpiler

import (
	"fmt"
	"testing"
)

func TestSyntaxInventory175FrozenWorkBudgets(t *testing.T) {
	cases := []struct {
		files, queries int
		warm           bool
		cgo            int64
		alloc          float64
		bytes          uint64
	}{
		{2, 4, false, 6837, 4697, 98214},
		{2, 4, true, 2734, 1878, 24410},
		{8, 16, false, 54690, 37568, 977673},
		{8, 16, true, 21875, 15027, 195277},
		{16, 32, false, 218756, 150273, 3908476},
		{16, 32, true, 87500, 60108, 781414},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("files%d_queries%d_warm%t", c.files, c.queries, c.warm), func(t *testing.T) {
			m := syntax175Measure(t, c.files, c.queries, c.warm)
			if m.Cgo > c.cgo {
				t.Errorf("SYNTAX_FROZEN_WORK_RED Cgo %d exceeds %d", m.Cgo, c.cgo)
			}
			if m.Allocs > c.alloc {
				t.Errorf("SYNTAX_FROZEN_WORK_RED Allocs %.0f exceeds %.0f", m.Allocs, c.alloc)
			}
			if m.Bytes > c.bytes {
				t.Errorf("SYNTAX_FROZEN_WORK_RED Bytes %d exceeds %d", m.Bytes, c.bytes)
			}
		})
	}
}
