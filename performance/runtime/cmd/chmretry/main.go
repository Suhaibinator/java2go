// A deterministic unrelated-bucket retry diagnostic, not a throughput benchmark.
package main

import (
	"encoding/json"
	"github.com/NickyBoy89/java2go/stdjava"
	"os"
	"sync/atomic"
)

type key struct {
	id      int32
	onEqual func()
}

func (k *key) HashCode() int32 { return k.id }
func (k *key) Equals(other any) bool {
	if k.onEqual != nil {
		k.onEqual()
	}
	right, ok := other.(*key)
	return ok && k.id == right.id
}
func main() {
	m := stdjava.NewConcurrentHashMap[*key, int32]()
	m.Put(&key{id: 1}, 10)
	reached, proceed, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	query := &key{id: 1, onEqual: func() {
		if calls.Add(1) == 1 {
			close(reached)
			<-proceed
		}
	}}
	go func() { m.Put(query, 11); close(done) }()
	<-reached
	m.Put(&key{id: 2}, 20) // Different hash; cannot change the queried collision bucket.
	close(proceed)
	<-done
	err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"equal_calls_for_one_update": calls.Load(), "final_key1": m.Get(&key{id: 1}), "final_key2": m.Get(&key{id: 2}), "size": m.Size(),
		"scope": "Go deterministic retry diagnostic; no Java parity or speed claim",
	})
	if err != nil {
		panic(err)
	}
}
