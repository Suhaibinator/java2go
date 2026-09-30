package main

import (
	"flag"
	"os"

	java2go "github.com/NickyBoy89/java2go"
	log "github.com/sirupsen/logrus"
)

func main() {
	if err := java2go.Run(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			return
		}
		log.WithField("error", err).Fatal("Conversion failed")
	}
}
