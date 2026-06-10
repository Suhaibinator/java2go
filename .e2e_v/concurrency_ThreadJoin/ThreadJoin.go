package main

import (
	fmt "fmt"
	stdjava "github.com/NickyBoy89/java2go/stdjava"
	os "os"
)

type worker struct {
	*stdjava.Thread
	id	int32
	results	[]int32
}

func newWorker(id int32, results []int32) *worker {
	wr := new(worker)
	wr.Thread = stdjava.NewThreadBase(wr)
	wr.id = id
	wr.results = results
	return wr
}
func (wr *worker) Run() {
	total := int32(0)
	for i := int32(1); i <= wr.id; i++ {
		total += i
	}
	wr.results[int(wr.id)] = total
}

type ThreadJoin struct {
}

func NewThreadJoin() *ThreadJoin {
	tn := new(ThreadJoin)
	return tn
}
// throws InterruptedException
func Main() {
	args := os.Args
	_ = args
	n := int32(5)
	results := make([]int32, n)
	workers := make([]*worker, n)
	for i := int32(0); i < n; i++ {
		workers[int(i)] = newWorker(i, results)
	}
	for i := int32(0); i < n; i++ {
		workers[int(i)].Start()
	}
	for i := int32(0); i < n; i++ {
		workers[int(i)].Join()
	}
	grand := int32(0)
	for i := int32(0); i < n; i++ {
		fmt.Println(fmt.Sprintf("%v%v%v%v", "worker ", i, " = ", results[int(i)]))
		grand += results[int(i)]
	}
	fmt.Println(fmt.Sprintf("%v%v", "grand ", grand))
}
