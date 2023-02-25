package controller

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rchapin/go-geocache-api/model"
	"github.com/rchapin/go-geocache-api/service"
	log "github.com/rchapin/rlog"
)

const apiVersion = "1"

type RequestPostCache struct {
	Name string   `json:"name"`
	Lat  float64  `json:"lat"`
	Long float64  `json:"long"`
	Tags []string `json:"tags"`
}

type RequestPutCache struct {
	Lat  float64  `json:"lat"`
	Long float64  `json:"long"`
	Tags []string `json:"tags"`
}

type ResponseCache struct {
	Id uint64 `json:"id"`
	RequestPostCache
}

type ResponseIds struct {
	Ids []uint64 `json:"ids"`
}

type RequestCacheTags struct {
	Tags []string `json:"tags"`
}

type Controller struct {
	ctx     context.Context
	cancel  context.CancelFunc
	wg      *sync.WaitGroup
	service service.Service
	port    string
	vPrefix string
}

func NewController(
	ctx context.Context,
	cancel context.CancelFunc,
	wg *sync.WaitGroup,
	service service.Service,
	port string,
) *Controller {
	return &Controller{
		ctx:     ctx,
		cancel:  cancel,
		wg:      wg,
		service: service,
		port:    port,
		vPrefix: "/v" + apiVersion,
	}
}

// parseJSON will parse the JSON payload from the gin context into the provided pointer.  If it is
// unable to parse the JSON it will set the proper response headers and error and then return the
// error to the caller.
func parseJSON[T any](c *gin.Context, ptr any) error {
	if err := c.ShouldBindJSON(&ptr); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return err
	}
	return nil
}

func cacheModelsToResponseCaches(caches []model.Cache) []ResponseCache {
	var retval []ResponseCache
	for _, cache := range caches {
		retval = append(retval, cacheModelToResponseCache(cache))
	}
	return retval
}

func cacheModelToResponseCache(cache model.Cache) ResponseCache {
	var tags []string = nil
	tagsCount := len(cache.Tags)
	if tagsCount > 0 {
		tags = make([]string, tagsCount)
		counter := 0
		for t := range cache.Tags {
			tags[counter] = t
			counter++
		}
		sort.Strings(tags)
	}
	r := RequestPostCache{
		Name: cache.Name,
		Lat:  cache.Lat,
		Long: cache.Long,
		Tags: tags,
	}
	return ResponseCache{
		Id:                cache.Id,
		RequestPostCache: r,
	}
}

func (s *Controller) createCacheHandler(c *gin.Context) {
	var rs RequestPostCache
	if err := parseJSON[RequestPostCache](c, &rs); err != nil {
		return
	}

	id, err := s.service.Create(rs.Name, rs.Lat, rs.Long, rs.Tags)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (s *Controller) getCachesHandler(c *gin.Context) {
	var caches []model.Cache
	var err error

	queryStringTags := c.DefaultQuery("tags", "")
	if queryStringTags != "" {
		tags := strings.Split(queryStringTags, ",")
		caches, err = s.service.GetByTags(tags)
		if err != nil {
			c.String(http.StatusNotFound, err.Error())
			return
		}
	} else {
		caches, err = s.service.GetAll()
		if err != nil {
			c.String(http.StatusNotFound, err.Error())
			return
		}
	}

	requestCaches := cacheModelsToResponseCaches(caches)
	c.JSON(http.StatusOK, requestCaches)
}

func (s *Controller) getNearestCachesHandler(c *gin.Context) {
	latStr := c.DefaultQuery("lat", "")
	longStr := c.DefaultQuery("long", "")
	maxDistanceStr := c.DefaultQuery("maxdistance", "")
	limitStr := c.DefaultQuery("limit", "")

	if latStr == "" || longStr == "" || maxDistanceStr == "" || limitStr == "" {
		c.String(
			http.StatusBadRequest,
			fmt.Sprintf(
				"missing required query args; "+
					"lat=%s, long=%s, maxdistance=%s, limit=%s",
				latStr, longStr, maxDistanceStr, limitStr))
		return
	}

	// TODO: Actually do error checking
	lat, _ := strconv.ParseFloat(latStr, 64)
	long, _ := strconv.ParseFloat(longStr, 64)
	maxDistance, _ := strconv.ParseFloat(maxDistanceStr, 64)
	limit, _ := strconv.Atoi(limitStr)

	caches, err := s.service.FindNearest(lat, long, maxDistance, limit)
	if err != nil {
		// FIXME: need to sort out this error checking/handling/reporting better
		c.String(http.StatusNotFound, err.Error())
		return
	}

	requestCaches := cacheModelsToResponseCaches(caches)
	c.JSON(http.StatusOK, requestCaches)
}

func (s *Controller) getCacheByNameHandler(c *gin.Context) {
	name := c.Params.ByName("name")
	if name == "" {
		c.String(http.StatusBadRequest, "Missing valid 'name' parameter")
		return
	}

	cache, err := s.service.GetByName(name)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	c.JSON(http.StatusOK, cacheModelToResponseCache(cache))
}

func (s *Controller) putCacheByNameHandler(c *gin.Context) {
	name := c.Params.ByName("name")
	if name == "" {
		c.String(http.StatusBadRequest, "Missing valid 'name' parameter")
		return
	}
	var rs RequestPutCache
	if err := parseJSON[RequestPutCache](c, &rs); err != nil {
		return
	}

	newTags := make(map[string]bool, len(rs.Tags))
	for _, t := range rs.Tags {
		newTags[t] = true
	}
	updatedCache := model.Cache{
		Name: name,
		Lat:  rs.Lat,
		Long: rs.Long,
		Tags: newTags,
	}
	cache, err := s.service.Update(name, updatedCache)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, cacheModelToResponseCache(cache))
}

func (s *Controller) ruok(c *gin.Context) {
	// TODO: implement some sort of heath check for monitoring.  This is a naive approach and would
	// be better served by the server emitting some sort of regular heartbeat stat that could be
	// monitored separately.
	c.String(200, "ack")
}

func (s *Controller) Start() {
	defer s.wg.Done()

	// Set up middleware for logging and panic recovery explicitly.  Other middleware can be added
	// to compose in Authn and Authz and other features.
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Define the routes for our http server
	router.POST(s.vPrefix+"/geocaches", s.createCacheHandler)
	router.GET(s.vPrefix+"/geocaches", s.getCachesHandler)
	router.GET(s.vPrefix+"/geocaches/:name", s.getCacheByNameHandler)
	router.PUT(s.vPrefix+"/geocaches/:name", s.putCacheByNameHandler)
	router.GET(s.vPrefix+"/geocaches/nearest", s.getNearestCachesHandler)
	router.GET(s.vPrefix+"/ruok", s.ruok)

	// Instantiate an http server then initialize it in a go routine so that it will not block and
	// that we can then listen to the close event on the context and execute a graceful shutdown
	// routine.
	server := &http.Server{
		Addr:    ":" + s.port,
		Handler: router,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("listen error; err=%s\n", err)
		}
	}()
	// Wait for the done event.
	<-s.ctx.Done()

	// Instantiate a secondary context granting the http server n number of seconds to finish
	// serving the existing requests that it is currently processing.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("http server forced to shutdown; err=%s\n", err)
	}
	log.Info("Server finished shutting down")
}
