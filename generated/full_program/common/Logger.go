package common

import fmt "fmt"

type Logger struct {
}

func Log(msg string) {
	fmt.Println(msg)
}
