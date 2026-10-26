// Retention diagnostic only. It is not a Java/Go performance comparison.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/NickyBoy89/java2go/stdjava"
)

type payload struct{ Bytes []byte }

var retainedList *stdjava.List[*payload]

func heap() runtime.MemStats {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

//go:noinline
func allocate(mode string, count, size int) int {
	execution := stdjava.NewExecution()
	list := stdjava.NewList[*payload]()
	for i := 0; i < count; i++ {
		value := &payload{Bytes: make([]byte, size)}
		value.Bytes[0] = byte(i)
		switch mode {
		case "monitor":
			guard := stdjava.MonitorEnterExecution(execution, value)
			stdjava.MonitorExitExecution(guard)
		case "list-remove", "list-clear":
			list.Add(value)
		case "baseline":
			runtime.KeepAlive(value)
		default:
			panic("unknown mode")
		}
	}
	if mode == "list-remove" {
		for !list.IsEmpty() {
			list.RemoveAt(list.Size() - 1)
		}
		retainedList = list
		nonnil := 0
		for _, value := range list.Slice()[:cap(list.Slice())] {
			if value != nil {
				nonnil++
			}
		}
		return nonnil
	}
	if mode == "list-clear" {
		list.Clear()
		retainedList = list
	}
	return 0
}

func main() {
	mode := flag.String("mode", "baseline", "baseline, monitor, list-remove, list-clear")
	count := flag.Int("count", 32, "payload count")
	size := flag.Int("bytes", 131072, "bytes per payload")
	profile := flag.String("heap-profile", "", "optional post-GC heap profile path")
	flag.Parse()
	if *count < 1 || *size < 1 {
		panic("positive count and bytes required")
	}
	before := heap()
	stale := allocate(*mode, *count, *size)
	after := heap()
	if *profile != "" {
		f, err := os.Create(*profile)
		if err != nil {
			panic(err)
		}
		if err := pprof.WriteHeapProfile(f); err != nil {
			panic(err)
		}
		if err := f.Close(); err != nil {
			panic(err)
		}
	}
	encoded, err := json.Marshal(map[string]any{
		"mode": *mode, "go_version": runtime.Version(), "count": *count, "payload_bytes": *size,
		"allocated_payload_bytes": int64(*count) * int64(*size), "heap_alloc_delta": int64(after.HeapAlloc) - int64(before.HeapAlloc),
		"heap_objects_delta": int64(after.HeapObjects) - int64(before.HeapObjects), "stale_list_references": stale,
		"scope": "Go runtime retention diagnostic; no Java parity or speed claim",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(encoded))
	runtime.KeepAlive(retainedList)
}
