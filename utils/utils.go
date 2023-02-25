package utils

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	log "github.com/rchapin/rlog"
)

func SetupLogging(level string) {
	os.Setenv("RLOG_LOG_LEVEL", strings.ToUpper(level))
	os.Setenv("RLOG_CALLER_INFO", "1")
	os.Setenv("RLOG_TIME_FORMAT", "2006-01-02 15:04:05.000")
	os.Setenv("RLOG_LOG_STREAM", "STDOUT")
	log.UpdateEnv()
}

func SetupSignalHandler(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup) {
	notify := make(chan os.Signal, 1)
	signal.Notify(notify, syscall.SIGINT, syscall.SIGTERM)
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Block here on the configured signals and cancel context on signal event
		select {
		case s := <-notify:
			log.Infof("Signal handler - Canceling context on signal, '%s'", s)
			// Cancel the app context which will trigger all child contexts
			cancel()
			return
		case <-ctx.Done():
			log.Info("Signal handler - Exiting on context done")
			return
		}
	}()
}
