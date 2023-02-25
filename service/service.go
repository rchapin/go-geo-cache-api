package service

//go:generate mockgen -destination=../mocks/mock_service.go -package=mocks github.com/rchapin/go-geocache-api/service Service

import (
	"context"
	"sync"

	"github.com/rchapin/go-geocache-api/model"
)

type Service interface {
	Create(name string, lat, long float64, tags []string) (uint64, error)
	FindNearest(lat, long, maxDistance float64, limit int) ([]model.Cache, error)
	GetAll() ([]model.Cache, error)
	GetById(id uint64) (model.Cache, error)
	GetByName(name string) (model.Cache, error)
	GetByTags(tags []string) ([]model.Cache, error)
	Delete(id uint64) error
	DeleteAll() error
	Update(name string, cache model.Cache) (model.Cache, error)
}

type ServiceImpl struct {
	ctx         context.Context
	cancel      context.CancelFunc
	wg          *sync.WaitGroup
	cacheStore model.CacheStore
}

func NewService(
	ctx context.Context,
	cancel context.CancelFunc,
	wg *sync.WaitGroup,
	cacheStore model.CacheStore,
) *ServiceImpl {
	return &ServiceImpl{
		ctx:         ctx,
		cancel:      cancel,
		wg:          wg,
		cacheStore: cacheStore,
	}
}

func (s *ServiceImpl) Create(
	name string,
	lat, long float64,
	tags []string,
) (uint64, error) {
	// Apply RBAC rules, other business logic, etc.
	return s.cacheStore.Create(name, lat, long, tags)
}

func (s *ServiceImpl) FindNearest(lat, long, maxDistance float64, limit int) ([]model.Cache, error) {
	return s.cacheStore.FindNearest(lat, long, maxDistance, limit)
}

func (s *ServiceImpl) GetAll() ([]model.Cache, error) {
	return nil, nil
}

func (s *ServiceImpl) GetById(id uint64) (model.Cache, error) {
	return s.cacheStore.GetById(id)
}

func (s *ServiceImpl) GetByName(name string) (model.Cache, error) {
	return s.cacheStore.GetByName(name)
}

func (s *ServiceImpl) GetByTags(tags []string) ([]model.Cache, error) {
	return s.cacheStore.GetByTags(tags)
}

func (s *ServiceImpl) Delete(id uint64) error {
	return nil
}

func (s *ServiceImpl) DeleteAll() error {
	return nil
}

func (s *ServiceImpl) Update(name string, cache model.Cache) (model.Cache, error) {
	return s.cacheStore.Update(name, cache)
}
