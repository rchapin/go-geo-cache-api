package model

//go:generate mockgen -destination=../mocks/mock_cache.go -package=mocks  github.com/rchapin/go-geocache-api/model CacheStore

import (
	"context"
	"fmt"
	"sync"

	"github.com/rchapin/go-geocache-api/geostore"
)

type Cache struct {
	Id   uint64          `json:"id"`
	Name string          `json:"name"`
	Lat  float64         `json:"lat"`
	Long float64         `json:"long"`
	Tags map[string]bool `json:"tags"`
}

type CacheNotFoundErr struct {
	id   uint64
	name string
}

func (e *CacheNotFoundErr) Error() string {
	return fmt.Sprintf("Cache not found; id=%d, name=%s", e.id, e.name)
}

type CacheStore interface {
	Create(name string, lat float64, long float64, tags []string) (uint64, error)
	FindNearest(lat, long, maxDistance float64, limit int) ([]Cache, error)
	GetAll() ([]Cache, error)
	GetById(id uint64) (Cache, error)
	GetByName(name string) (Cache, error)
	GetByTags(tags []string) ([]Cache, error)
	Delete(id uint64) error
	DeleteAll() error
	Update(name string, cache Cache) (Cache, error)
	Shutdown() error
}

type InMemCacheStore struct {
	caches       map[uint64]*Cache
	sCounter     uint64
	cachesByName map[string]*Cache
	cachesByTag  map[string]map[*Cache]bool
	geostore     geostore.GeoStore
	sMux         *sync.RWMutex
}

func NewCacheStore(
	ctx context.Context,
	cancel context.CancelFunc,
	wg *sync.WaitGroup,
	geoStore geostore.GeoStore,
) CacheStore {
	return &InMemCacheStore{
		caches:       make(map[uint64]*Cache),
		sCounter:     1,
		cachesByName: make(map[string]*Cache),
		cachesByTag:  make(map[string]map[*Cache]bool),
		geostore:     geoStore,
		sMux:         &sync.RWMutex{},
	}
}

func (s *InMemCacheStore) Create(
	name string,
	lat float64,
	long float64,
	tags []string,
) (uint64, error) {
	// Convert the slice of tags provided for the new element into a map that we will store.
	t := make(map[string]bool, len(tags))
	for _, tag := range tags {
		t[tag] = true
	}

	s.sMux.Lock()
	defer s.sMux.Unlock()

	cache := &Cache{
		Id:   s.sCounter,
		Name: name,
		Lat:  lat,
		Long: long,
		Tags: t,
	}
	s.caches[cache.Id] = cache
	s.cachesByName[cache.Name] = cache
	id := cache.Id
	// Bump our 'auto-incrementing int id value.
	s.sCounter++

	var ok bool
	var tMap map[*Cache]bool
	for _, t := range tags {
		tMap, ok = s.cachesByTag[t]
		if !ok {
			tMap = make(map[*Cache]bool)
			s.cachesByTag[t] = tMap
		}
		tMap[cache] = true
	}

	node := geostore.NewNode(cache.Long, cache.Lat, cache.Id)
	s.geostore.Insert(node)

	return id, nil
}

func copyCache(cache *Cache) Cache {
	return Cache{
		Id:   cache.Id,
		Name: cache.Name,
		Lat:  cache.Lat,
		Long: cache.Long,
		Tags: cache.Tags,
	}
}

func (s *InMemCacheStore) FindNearest(
	lat, long, maxDistance float64,
	limit int,
) ([]Cache, error) {
	s.sMux.RLock()
	defer s.sMux.RUnlock()

	ids := s.geostore.FindNearest(lat, long, maxDistance, limit)
	// Now that we have the ids from the GeoStore, get the details for the specific caches and
	// return them to the caller.
	// TODO: need to add some error checking here to ensure that the datastore is not in some
	// inconsistent state. If it is, there is a bug and this situation should never happen.
	retval := make([]Cache, len(ids))
	for i := 0; i < len(ids); i++ {
		s := s.caches[ids[i]]
		retval[i] = copyCache(s)
	}

	return retval, nil
}

func (s *InMemCacheStore) GetAll() ([]Cache, error) {
	s.sMux.RLock()
	defer s.sMux.RUnlock()

	// TODO:
	return nil, nil
}

func (s *InMemCacheStore) GetById(id uint64) (Cache, error) {
	s.sMux.RLock()
	defer s.sMux.RUnlock()

	cache, ok := s.caches[id]
	if !ok {
		return Cache{}, &CacheNotFoundErr{id: id}
	}
	retval := copyCache(cache)
	return retval, nil
}

func (s *InMemCacheStore) GetByName(name string) (Cache, error) {
	s.sMux.RLock()
	defer s.sMux.RUnlock()

	cache, ok := s.cachesByName[name]
	if !ok {
		return Cache{}, &CacheNotFoundErr{name: name}
	}
	retval := copyCache(cache)
	return retval, nil
}

func (s *InMemCacheStore) GetByTags(tags []string) ([]Cache, error) {
	s.sMux.RLock()
	defer s.sMux.RUnlock()

	var caches []Cache
	for _, tag := range tags {
		m, ok := s.cachesByTag[tag]
		if !ok {
			// There aren't any Caches with the provided tag
			continue
		}
		for cache := range m {
			caches = append(caches, copyCache(cache))
		}
	}
	return caches, nil
}

func (s *InMemCacheStore) Delete(id uint64) error {
	return nil
}

func (s *InMemCacheStore) DeleteAll() error {
	return nil
}

func (s *InMemCacheStore) Update(name string, cache Cache) (Cache, error) {
	s.sMux.Lock()
	defer s.sMux.Unlock()

	existingCache, ok := s.cachesByName[name]
	if !ok {
		return Cache{}, fmt.Errorf("cache not found to update; name=%s", name)
	}

	// Since we have a pointer to the cache we can just update the values of the pointer and then
	// return a copy of the Cache to the caller and unlock the mutex.
	existingCache.Lat = cache.Lat
	existingCache.Long = cache.Long
	existingCache.Tags = cache.Tags

	// TODO: Update the record in the GeoStore

	retval := copyCache(existingCache)
	return retval, nil
}

func (s *InMemCacheStore) Shutdown() error {
	return nil
}
