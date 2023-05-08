package main

import (
	log "github.com/sirupsen/logrus"
)

func init() {
	Formatter := new(log.TextFormatter)
	Formatter.TimestampFormat = "02-01-2006 15:04:05"
	Formatter.FullTimestamp = true
	log.SetFormatter(Formatter)
}

func main() {
	Execute()
}
