package main

import (
	fmt "fmt"
	stdjava "github.com/NickyBoy89/java2go/stdjava"
	os "os"
)

type SyncCounter struct {
	count int32
}

func (sr *SyncCounter) __java2goInitFields() {
	sr.count = 0
}
func NewSyncCounter() *SyncCounter {
	sr := new(SyncCounter)
	sr.__java2goInitFields()
	return sr
}
func (sr *SyncCounter) increment() {
	__java2goMethodMonitor := stdjava.MonitorEnter(sr)
	defer stdjava.MonitorExit(__java2goMethodMonitor)
	sr.count++
}
func (sr *SyncCounter) get() int32 {
	return sr.count
}
// throws InterruptedException
func Main() {
	args := os.Args
	_ = args
	counter := NewSyncCounter()
	threadCount := int32(4)
	perThread := int32(1000)
	threads := make([]*stdjava.Thread, threadCount)
	for t := int32(0); t < threadCount; t++ {
		threads[int(t)] = stdjava.NewThread(&SyncCounterAnon1{perThread: perThread, counter: counter})
	}
	for t := int32(0); t < threadCount; t++ {
		threads[int(t)].Start()
	}
	for t := int32(0); t < threadCount; t++ {
		threads[int(t)].Join()
	}
	fmt.Println(counter.get())
}

type SyncCounterAnon1 struct {
	perThread	int32
	counter		*SyncCounter
}

func (s1 *SyncCounterAnon1) Run() {
	for i := int32(0); i < s1.perThread; i++ {
		s1.counter.increment()
	}
}
