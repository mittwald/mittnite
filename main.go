package main

import (
	"github.com/mittwald/mittnite/cmd"
	log "github.com/sirupsen/logrus"
	"os"
)

func init() {
	Formatter := new(log.TextFormatter)
	Formatter.TimestampFormat = "02-01-2006 15:04:05"
	Formatter.FullTimestamp = true
	log.SetFormatter(Formatter)
	if os.Getenv("MITTNITE_LOG_LEVEL") == "debug" {
		log.SetLevel(log.DebugLevel)
	}
}

func main() {
	cmd.Execute()
}
