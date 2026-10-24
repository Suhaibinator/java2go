// Allocation and identity checks only; deliberately no sustained timing.
package main

import (
	"encoding/json"
	"os"
	"runtime"
	"testing"

	"github.com/NickyBoy89/java2go/stdjava"
)

func main() {
	ready := stdjava.NewCountDownLatch(0)
	readyWaitAllocs := testing.AllocsPerRun(32, func() {
		if !ready.AwaitTimed(1, stdjava.SECONDS) {
			panic("ready latch blocked")
		}
	})
	closedDownAllocs := testing.AllocsPerRun(32, func() {
		ready.CountDown()
		if ready.GetCount() != 0 {
			panic("latch underflow")
		}
	})
	pool := stdjava.NewFixedThreadPool(1)
	done := make(chan *stdjava.Thread, 1)
	var worker *stdjava.Thread
	identityStable := true
	task := stdjava.NewRunnableFuncAdapter(func(execution *stdjava.Execution) { done <- stdjava.ThreadCurrentThread(execution) })
	executeAllocs := testing.AllocsPerRun(32, func() {
		pool.Execute(task)
		current := <-done
		if worker == nil {
			worker = current
		} else if worker != current {
			identityStable = false
		}
	})
	workerAlive := worker.IsAlive()
	pool.Shutdown()
	pool.AwaitTermination()
	workerStopped := !worker.IsAlive()
	err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"go_version": runtime.Version(), "ready_timed_latch_allocs": readyWaitAllocs, "closed_countdown_allocs": closedDownAllocs,
		"sequential_execute_allocs_per_task": executeAllocs, "worker_identity_stable": identityStable,
		"worker_alive_before_shutdown": workerAlive, "worker_stopped_after_termination": workerStopped,
		"scope": "33 calls per allocation sample; Go-only diagnostic, no Java parity or speed claim",
	})
	if err != nil {
		panic(err)
	}
}
