package inttest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/rchapin/go-geocache-api/utils"
	log "github.com/rchapin/rlog"
	"github.com/stretchr/testify/assert"
)

var rm *ResourceManager

type IdAble interface {
	GetId() uint64
}

type TestCache struct {
	Name string   `json:"name"`
	Lat  float64  `json:"lat"`
	Long float64  `json:"long"`
	Tags []string `json:"tags"`
}

type TestCacheUpdate struct {
	Lat  float64  `json:"lat"`
	Long float64  `json:"long"`
	Tags []string `json:"tags"`
}

type TestIdResponse struct {
	Id uint64 `json:"id"`
}

type TestGetCacheResponse struct {
	Id   uint64   `json:"id"`
	Name string   `json:"name"`
	Lat  float64  `json:"lat"`
	Long float64  `json:"long"`
	Tags []string `json:"tags"`
}

func (t TestGetCacheResponse) GetId() uint64 {
	return t.Id
}

func createUrlPrefix() string {
	return "http://localhost:" + testPort + "/v1"
}

func postCache(s TestCache) *http.Response {
	url := createUrlPrefix() + "/geocaches"
	return sendJson(s, "POST", url)
}

func putCache(name string, s TestCacheUpdate) *http.Response {
	url := createUrlPrefix() + "/geocaches/" + name
	return sendJson(s, "PUT", url)
}

func sendJson[T any](s T, httpVerb string, url string) *http.Response {
	json, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	request, err := http.NewRequest(httpVerb, url, bytes.NewBuffer(json))
	if err != nil {
		panic(err)
	}

	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	client := &http.Client{}
	retval, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	return retval
}

func setUpTest() {
	rm = NewResourceManager()
}

func setUpSubTest() *TestRunner {
	rm.refreshContextsWg()
	return NewTestRunner(rm.testRunnerCtx, rm.testRunnerCancel, rm.testRunnerWg)
}

func startServer(t *testing.T) *TestRunner {
	tr := setUpSubTest()
	if err := tr.runServer(); err != nil {
		assert.Fail(t, "error returned when attempting to start http server; err=%s\n", err)
		tr.cancel()
		panic(err)
	}
	return tr
}

func TestMain(m *testing.M) {
	if os.Getenv("INTEGRATION") == "" {
		fmt.Printf("Skipping integration tests.  To run set env var INTEGRATION=1")
		return
	}

	utils.SetupLogging("info")
	log.Info("From integration test TestMain")
	setUpTest()
	utils.SetupSignalHandler(rm.tCtx, rm.tCancel, rm.tWg)
	runExitCode := m.Run()
	log.Info("Integration tests complete")
	os.Exit(runExitCode)
}

// Integration Tests -----------------------------------------------------------

func TestRuok(t *testing.T) {
	tr := startServer(t)
	resp := execGet(t, createUrlPrefix()+"/ruok")

	validateStatus(t, 200, resp)
	actualBody := getResponseBodyString(t, resp)
	assert.Equal(t, "ack", actualBody)

	resp.Body.Close()
	tr.shutdownServer()
}

// TestCreateCache test creating a Cache with and without tags
func TestCreateCache(t *testing.T) {
	testData := []struct {
		testCache           TestCache
		expectedPostResp    TestIdResponse
		expectedGetByIdResp TestGetCacheResponse
	}{
		{
			testCache: TestCache{
				Name: "cache_one",
				Lat:  39.24196703747868,
				Long: -77.97336975938909,
				Tags: nil,
			},
			expectedPostResp: TestIdResponse{Id: 1},
			expectedGetByIdResp: TestGetCacheResponse{
				Id:   1,
				Name: "cache_one",
				Lat:  39.24196703747868,
				Long: -77.97336975938909,
				Tags: nil,
			},
		},
		{
			testCache: TestCache{
				Name: "cache_two",
				Lat:  39.24196703747868,
				Long: -77.97336975938909,
				Tags: []string{"wv", "temp", "voltage"},
			},
			expectedPostResp: TestIdResponse{Id: 1},
			expectedGetByIdResp: TestGetCacheResponse{
				Id:   1,
				Name: "cache_two",
				Lat:  39.24196703747868,
				Long: -77.97336975938909,
				Tags: []string{"temp", "voltage", "wv"},
			},
		},
	}

	for _, td := range testData {
		execTestCreateCache(
			t,
			td.testCache,
			td.expectedPostResp,
			td.expectedGetByIdResp,
		)
	}
}

func execTestCreateCache(
	t *testing.T,
	testCache TestCache,
	expectedPostResp TestIdResponse,
	expectedGetByIdResp TestGetCacheResponse,
) {
	tr := startServer(t)

	resp := postCache(testCache)
	validateStatus(t, 200, resp)
	actualResponseStr := getResponseBodyString(t, resp)
	actualIdResp := TestIdResponse{}
	_ = json.Unmarshal([]byte(actualResponseStr), &actualIdResp)
	assert.Equal(t, expectedPostResp, actualIdResp)
	resp.Body.Close()

	resp = execGet(t, createUrlPrefix()+"/geocaches/"+testCache.Name)
	validateStatus(t, 200, resp)
	actualResponseStr = getResponseBodyString(t, resp)
	actualGetByIdResp := TestGetCacheResponse{}
	_ = json.Unmarshal([]byte(actualResponseStr), &actualGetByIdResp)
	assert.Equal(t, expectedGetByIdResp, actualGetByIdResp)
	resp.Body.Close()

	tr.shutdownServer()
}

func TestFindNearest(t *testing.T) {
	tr := startServer(t)

	// The default configurations in the run.Run method configures the GeoStore for a max number of
	// 4 nodes in each QuadTree.  As a result we have to add 5 nodes before it will partition the
	// GeoStore and we can then query it and have the previously entered nodes partitioned to setup
	// a valid pre-condition for the test.  So, we will add 5 nodes.
	tCaches := []TestCache{
		{
			Name: "canada",
			Lat:  53.61760431337473,
			Long: -106.72319029988779,
			Tags: []string{"ocean", "atlantic", "flowrate"},
		},
		{
			Name: "oregon",
			Long: -120.54074440145642,
			Lat:  43.38552157601114,
			Tags: nil,
		},
		{
			Name: "mongolia",
			Lat:  46.88910832340091,
			Long: 97.00726436805842,
			Tags: nil,
		},
		{
			Name: "peru",
			Long: -72.27006768132226,
			Lat:  -36.351849320377774,
			Tags: nil,
		},
		{
			Name: "australia",
			Long: 124.42913747263096,
			Lat:  -23.605766549164937,
			Tags: nil,
		},
	}
	for _, ts := range tCaches {
		resp := postCache(ts)
		resp.Body.Close()
	}

	expectedS1 := TestGetCacheResponse{
		Id:   1,
		Name: "canada",
		Lat:  53.61760431337473,
		Long: -106.72319029988779,
		Tags: []string{"atlantic", "flowrate", "ocean"},
	}
	expectedS2 := TestGetCacheResponse{
		Id:   2,
		Name: "oregon",
		Long: -120.54074440145642,
		Lat:  43.38552157601114,
		Tags: nil,
	}

	url := createUrlPrefix() + "/geocaches/nearest?lat=55.87272342&long=-104.9234282&maxdistance=0&limit=-1"
	resp := execGet(t, url)
	validateStatus(t, 200, resp)
	expectedResp := []TestGetCacheResponse{expectedS1, expectedS2}
	validateGetResults(t, resp, expectedResp)

	tr.shutdownServer()
}

func TestGetCacheByName(t *testing.T) {
	tr := startServer(t)

	// Post a handful of Caches
	tCaches := []TestCache{
		{
			Name: "s1",
			Lat:  38.394432064782755,
			Long: -75.0613367366317,
			Tags: []string{"ocean", "atlantic", "flowrate"},
		},
		{
			Name: "s2",
			Lat:  39.33030191224595,
			Long: -77.74073236877527,
			Tags: []string{"river", "flowrate"},
		},
		{
			Name: "s3",
			Lat:  37.79088776167161,
			Long: -122.50578266113429,
			Tags: []string{"ocean", "pacific"},
		},
	}
	for _, ts := range tCaches {
		resp := postCache(ts)
		resp.Body.Close()
	}

	expectedS1 := TestGetCacheResponse{
		Id:   1,
		Name: "s1",
		Lat:  38.394432064782755,
		Long: -75.0613367366317,
		Tags: []string{"atlantic", "flowrate", "ocean"},
	}

	resp := execGet(t, createUrlPrefix()+"/geocaches/s1")
	validateStatus(t, 200, resp)
	validateGetResult(t, resp, expectedS1)

	tr.shutdownServer()
}

func TestGetCacheByTags(t *testing.T) {
	tr := startServer(t)

	// Post a handful of Caches
	tCaches := []TestCache{
		{
			Name: "s1",
			Lat:  38.394432064782755,
			Long: -75.0613367366317,
			Tags: []string{"ocean", "atlantic", "flowrate"},
		},
		{
			Name: "s2",
			Lat:  39.33030191224595,
			Long: -77.74073236877527,
			Tags: []string{"river", "flowrate"},
		},
		{
			Name: "s3",
			Lat:  37.79088776167161,
			Long: -122.50578266113429,
			Tags: []string{"ocean", "pacific"},
		},
	}
	for _, ts := range tCaches {
		resp := postCache(ts)
		resp.Body.Close()
	}

	expectedS1 := TestGetCacheResponse{
		Id:   1,
		Name: "s1",
		Lat:  38.394432064782755,
		Long: -75.0613367366317,
		Tags: []string{"atlantic", "flowrate", "ocean"},
	}
	expectedS2 := TestGetCacheResponse{
		Id:   2,
		Name: "s2",
		Lat:  39.33030191224595,
		Long: -77.74073236877527,
		Tags: []string{"flowrate", "river"},
	}
	expectedS3 := TestGetCacheResponse{
		Id:   3,
		Name: "s3",
		Lat:  37.79088776167161,
		Long: -122.50578266113429,
		Tags: []string{"ocean", "pacific"},
	}

	// Include the 'anemometer' tag for which there are no Caches with that tag
	resp := execGet(t, createUrlPrefix()+"/geocaches?tags=ocean,anemometer")
	validateStatus(t, 200, resp)
	expectedResp := []TestGetCacheResponse{expectedS1, expectedS3}
	validateGetResults(t, resp, expectedResp)
	resp.Body.Close()

	// Just look for the 'flowrate' tag
	resp = execGet(t, createUrlPrefix()+"/geocaches?tags=flowrate")
	validateStatus(t, 200, resp)
	expectedResp = []TestGetCacheResponse{expectedS1, expectedS2}
	validateGetResults(t, resp, expectedResp)
	resp.Body.Close()

	tr.shutdownServer()
}

func TestUpdateCacheByName(t *testing.T) {
	tr := startServer(t)

	// Post a cache
	name := "s1"
	testCache := TestCache{
		Name: name,
		Lat:  38.394432064782755,
		Long: -75.0613367366317,
		Tags: []string{"ocean", "atlantic", "flowrate"},
	}
	resp := postCache(testCache)
	resp.Body.Close()

	// Update the lat and add a tag
	tU1 := TestCacheUpdate{
		Lat:  39.423423,
		Long: -75.0613367366317,
		Tags: []string{"ocean", "atlantic", "flowrate", "temp"},
	}
	resp = putCache(name, tU1)
	validateStatus(t, 200, resp)

	// Get the same record and validate that it has been changed as expected
	expectedResp := TestGetCacheResponse{
		Id:   1,
		Name: name,
		Lat:  39.423423,
		Long: -75.0613367366317,
		Tags: []string{"atlantic", "flowrate", "ocean", "temp"},
	}
	resp = execGet(t, createUrlPrefix()+"/geocaches/"+name)
	validateStatus(t, 200, resp)
	validateGetResult(t, resp, expectedResp)
	resp.Body.Close()

	tr.shutdownServer()
}

func execGet(t *testing.T, url string) *http.Response {
	retval, err := http.Get(url)
	if err != nil {
		assert.Fail(t, "error returned when attempting to hit test endpoint; err=%s\n", err)
		return nil
	}
	return retval
}

func getResponseBodyString(t *testing.T, resp *http.Response) string {
	body, err := io.ReadAll((resp.Body))
	if err != nil {
		assert.Fail(t, "unable to extract string from response body", err)
		return ""
	}
	return string(body)
}

func validateStatus(t *testing.T, expected int, resp *http.Response) {
	actualCode := resp.StatusCode
	assert.Equal(t, expected, actualCode)
}

func validateGetResult(t *testing.T, resp *http.Response, expectedResp TestGetCacheResponse) {
	actualRespStr := getResponseBodyString(t, resp)
	actualResp := TestGetCacheResponse{}
	err := json.Unmarshal([]byte(actualRespStr), &actualResp)
	if err != nil {
		panic(err)
	}
	assert.True(t, reflect.DeepEqual(expectedResp, actualResp))
}

func validateGetResults(t *testing.T, resp *http.Response, expectedResp []TestGetCacheResponse) {
	actualRespStr := getResponseBodyString(t, resp)
	actualResp := []TestGetCacheResponse{}
	err := json.Unmarshal([]byte(actualRespStr), &actualResp)
	if err != nil {
		panic(err)
	}

	// In order to avoid the problem whereby the lists are not ordered in the same way, we will
	// iterate over each and build a map that is keyed by the id of the elements within.  Then we
	// will do the comparison.
	expected := sliceToMap(expectedResp)
	actual := sliceToMap(actualResp)

	assert.True(
		t,
		reflect.DeepEqual(expected, actual),
		fmt.Sprintf("expectedResp=%+v\nactualResp=%+v\n", expected, actual),
	)
}

func sliceToMap[T IdAble](s []T) map[uint64]T {
	retval := make(map[uint64]T, len(s))
	for _, e := range s {
		retval[e.GetId()] = e
	}
	return retval
}
