package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/rchapin/go-geocache-api/mocks"
	"github.com/stretchr/testify/assert"
)

func TestRuokRoute(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	mockService := mocks.NewMockService(mockCtrl)
	server := NewController(ctx, cancel, wg, mockService, "8080")

	path := "/ruok"
	router := gin.Default()
	router.GET(path, server.ruok)
	req, _ := http.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "ack", w.Body.String())
}
