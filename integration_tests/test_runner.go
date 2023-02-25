package inttest

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rchapin/go-geocache-api/run"
	log "github.com/rchapin/rlog"
)

type TestRunner struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

func NewTestRunner(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup) *TestRunner {
	return &TestRunner{
		ctx:    ctx,
		cancel: cancel,
		wg:     wg,
	}
}

func (t *TestRunner) runServer() error {
	// Create a set of "mock" cli args and execute the Run entrypoint for the application.
	args := []string{"placeholder-token", "-p", testPort}
	go run.Run(args, t.ctx, t.cancel, t.wg)

	// Now we need to block and wait until the server is up and running before we can return to the
	// test method and start hitting the server with test REST calls.
	timer := time.NewTimer(time.Duration(startServerPollTimeout) * time.Millisecond)
	ticker := time.NewTicker(time.Duration(startServerPollWait) * time.Millisecond)

	// Context and CancelFunc to be used while waiting for the http server to be ready to serve
	// requests. We will base it off the test context so that we can cancel if the entire test is
	// cancelled.
	ctx, cancel := context.WithCancel(t.ctx)
	for {
		select {
		case <-ticker.C:
			go func() {
				log.Info("Checking to see if server is accepting connections")
				resp, err := http.Get("http://localhost:" + testPort + "/v1/ruok")
				if err != nil {
					log.Infof(
						"GET returned error. Waiting for http server to start listening; err=%s\n",
						err,
					)
					return
				}
				defer resp.Body.Close()
				log.Infof("%+v", resp)
				if resp.StatusCode == 200 {
					body, err := io.ReadAll((resp.Body))
					if err != nil {
						panic(err)
					}
					log.Infof(
						"Success!  http server is accepting requests, ruok returned 200, body=%s\n",
						string(body))
					cancel()
				}
			}()
		case <-timer.C:
			cancel()
			return errors.New("timed out waiting for http server to start listening")
		case <-ctx.Done():
			log.Info("Exiting loop waiting for http server to start")
			cancel()
			return nil
		}
	}
}

func (t *TestRunner) shutdownServer() {
	// This is the same cancel func passed into the Server.  Cancelling it will cause the server to
	// gracefully shutdown.
	t.cancel()
	t.wg.Wait()
}
