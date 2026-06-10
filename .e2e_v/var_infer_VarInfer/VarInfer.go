package main

import (
	fmt "fmt"
	os "os"
)

type VarInfer struct {
}

func NewVarInfer() *VarInfer {
	vr := new(VarInfer)
	return vr
}
func Main() {
	args := os.Args
	_ = args
	n := int32(42)
	name := "hello"
	d := 3.5
	sum := int32(n + 8)
	fmt.Println(n)
	fmt.Println(name)
	fmt.Println(d)
	fmt.Println(sum)
	for i := int32(0); i < 3; i++ {
		fmt.Println(fmt.Sprintf("%v%v", "i=", i))
	}
	total := int32(0)
	xs := []int32{10, 20, 30}
	for _, x := range xs {
		total += x
	}
	fmt.Println(fmt.Sprintf("%v%v", "total=", total))
}
