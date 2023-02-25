package run

import (
	"context"
	"sync"

	"github.com/akamensky/argparse"
	"github.com/rchapin/go-geocache-api/controller"
	"github.com/rchapin/go-geocache-api/geostore"
	"github.com/rchapin/go-geocache-api/model"
	"github.com/rchapin/go-geocache-api/service"
	"github.com/rchapin/go-geocache-api/utils"
	log "github.com/rchapin/rlog"
)

func Run(args []string, ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup) error {
	// Initially setup logging at info level.  We can update that once we parse our cli args
	utils.SetupLogging("info")
	log.Infof("Starting application; args=%+v", args)

	parser := argparse.NewParser("cache-api", "Cache API")
	port := parser.String("p", "port", &argparse.Options{
		Required: true,
		Help:     "Port on which the http server will listen",
	})
	logLevel := parser.String("l", "log-level", &argparse.Options{
		Default:  "info",
		Required: false,
		Help:     "Log level",
	})

	if err := parser.Parse(args); err != nil {
		return err
	}

	// Reconfigure logging based on configured preference
	utils.SetupLogging(*logLevel)
	log.Infof("Instantiating cache-api server; port=%s", port)

	utils.SetupSignalHandler(ctx, cancel, wg)

	// Instantiate and inject an in-memory instances of the GeoStore and CacheStore.  We could
	// eventually do some fancy dynamic-instantiation-from-config but for now we will just use the
	// only implementations that we have.  This does decouple the implementations and makes all of
	// this much easier to test and swap out whenever needed in the future.

	// Instantiate a GeoStore that covers the entire globe, with a max capacity of 4 for each
	// quadrant.
	// TODO: make the coordinates and the maxCapacity configurable
	quadrant := geostore.NewQuadrant(-180, -90, 180, 90, true)
	qt := geostore.NewQuadTree(1, quadrant, 4)
	geostore := geostore.NewGeoStoreInMem(qt)

	cacheStore := model.NewCacheStore(ctx, cancel, wg, geostore)
	service := service.NewService(ctx, cancel, wg, cacheStore)
	server := controller.NewController(ctx, cancel, wg, service, *port)
	wg.Add(1)
	server.Start()
	return nil
}
