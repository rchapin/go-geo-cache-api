package main

import (
	"context"
	"os"
	"sync"

	"github.com/rchapin/go-geocache-api/run"
	log "github.com/rchapin/rlog"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	if err := run.Run(os.Args, ctx, cancel, wg); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
