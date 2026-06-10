package main

import (
	fmt "fmt"
	os "os"
)

type intOp interface {
	Apply(a int32, b int32) int32
}
type intOpFuncAdapter struct {
	fn func(a int32, b int32) int32
}

func (fa *intOpFuncAdapter) Apply(a int32, b int32) int32 {
	return fa.fn(a, b)
}
func NewintOpFuncAdapter(fn func(a int32, b int32) int32) intOp {
	return &intOpFuncAdapter{fn: fn}
}

type intPred interface {
	Test(value int32) bool
}
type intPredFuncAdapter struct {
	fn func(value int32) bool
}

func (fa *intPredFuncAdapter) Test(value int32) bool {
	return fa.fn(value)
}
func NewintPredFuncAdapter(fn func(value int32) bool) intPred {
	return &intPredFuncAdapter{fn: fn}
}

type Lambdas struct {
}

func NewLambdas() *Lambdas {
	ls := new(Lambdas)
	return ls
}
func reduce(xs []int32, seed int32, op intOp) int32 {
	acc := int32(seed)
	for _, x := range xs {
		acc = op.Apply(acc, x)
	}
	return acc
}
func countMatching(xs []int32, pred intPred) int32 {
	count := int32(0)
	for _, x := range xs {
		if pred.Test(x) {
			count++
		}
	}
	return count
}
func Main() {
	args := os.Args
	_ = args
	xs := []int32{1, 2, 3, 4, 5}
	add := NewintOpFuncAdapter(func(a int32, b int32) int32 {
		return a + b
	})
	mul := NewintOpFuncAdapter(func(a int32, b int32) int32 {
		return a * b
	})
	fmt.Println(reduce(xs, 0, add))
	fmt.Println(reduce(xs, 1, mul))
	even := NewintPredFuncAdapter(func(v int32) bool {
		return v%2 == 0
	})
	fmt.Println(countMatching(xs, even))
}
