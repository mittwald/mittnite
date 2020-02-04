package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mittwald/mittnite/pkg/proc"
)

func main() {

	//bla := make(map[string]string)
	//
	//fmt.Println("hallo:", bla["hallo"])
	//matches, err := filepath.Glob("/tmp/test2/")
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println("matches:", matches)

	sigChan := make(chan os.Signal, 2)

	signal.Notify(sigChan,
		syscall.SIGTERM,
		syscall.SIGINT,
	)

	runner := proc.NewRunner(nil)

	// start runner
	go runner.Run()

	// stop runner on SIGINT/SIGTERM
	go func() {
		runner.Stop(<-sigChan)
	}()

	time.Sleep(5 * time.Second)

	runner.Stop(nil)
}
