package main

import (
	"flag"
	"os"

	"github.com/NickyBoy89/java2go/internal/cliownership"
	"github.com/NickyBoy89/java2go/transpiler"
	log "github.com/sirupsen/logrus"
)

func main() {
	if err := transpiler.RunStandalone(os.Args[1:], cliownership.New()); err != nil {
		if err == flag.ErrHelp {
			return
		}
		log.WithField("error", err).Fatal("Conversion failed")
	}
}
