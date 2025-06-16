package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"

	"github.com/mittwald/mittnite/cmd"
	log "github.com/sirupsen/logrus"
)

func init() {
	Formatter := new(log.TextFormatter)
	Formatter.TimestampFormat = "02-01-2006 15:04:05"
	Formatter.FullTimestamp = true
	log.SetFormatter(Formatter)
}

func main() {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Error(err)
	} else {
		defer listener.Close()
		fmt.Printf("pprof listening on %s\n", listener.Addr().String())
		go http.Serve(listener, nil)
	}
	cmd.Execute()
}
