package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/justinas/alice"
	"github.com/stretchr/testify/assert"
)

func TestLimiterWithHeavyHandler(t *testing.T) {
	result := ConcurrencyTest(t, HeavyHandler, 0, 0) // no requests and no limits set
	assert.Equal(t, 0, result.maxConcurrency)        // no concurrent requests
	assert.Equal(t, 0, result.accepted)
	assert.Equal(t, 0, result.denied)

	result = ConcurrencyTest(t, HeavyHandler, 0, 10) // no requests and limit of 10 set
	assert.Equal(t, 0, result.maxConcurrency)        // no concurrent requests
	assert.Equal(t, 0, result.accepted)
	assert.Equal(t, 0, result.denied)

	result = ConcurrencyTest(t, HeavyHandler, 100, 0) // 100 requests and no limits set
	assert.Equal(t, 100, result.maxConcurrency)       // 100 concurrent requests observed
	assert.Equal(t, 100, result.accepted)
	assert.Equal(t, 0, result.denied)

	result = ConcurrencyTest(t, HeavyHandler, 100, 1) // 100 requests and limit of 1 set
	assert.Equal(t, 1, result.maxConcurrency)         // 1 concurrent request observed
	assert.Equal(t, 1, result.accepted)
	assert.Equal(t, 99, result.denied)

	result = ConcurrencyTest(t, HeavyHandler, 100, 10) // 100 requests and limit of 10 set
	assert.Equal(t, 10, result.maxConcurrency)         // 10 concurrent requests observed
	assert.Equal(t, 10, result.accepted)
	assert.Equal(t, 90, result.denied)

	result = ConcurrencyTest(t, HeavyHandler, 100, 100) // 100 requests and limit of 100 set
	assert.Equal(t, 100, result.maxConcurrency)         // 100 concurrent requests observed
	assert.Equal(t, 100, result.accepted)
	assert.Equal(t, 0, result.denied)

	result = ConcurrencyTest(t, HeavyHandler, 100, 1000) // 100 requests and limit of 1000 set
	assert.Equal(t, 100, result.maxConcurrency)          // 100 concurrent requests observed
	assert.Equal(t, 100, result.accepted)
	assert.Equal(t, 0, result.denied)
}

func TestLimiterWithLightHandler(t *testing.T) {
	result := ConcurrencyTest(t, LightHandler, 1000, 10) // 1000 requests and limit of 10 set
	assert.Equal(t, 10, result.maxConcurrency)           // no more than 10 concurrent requests at all times
	assert.GreaterOrEqual(t, result.accepted, 10)        // but more than 10 were accepted overall
	assert.Equal(t, 1000, result.accepted+result.denied) // no other responses than accepted or denied

}

type ConcurrencyTestResult struct {
	maxConcurrency int
	accepted       int
	denied         int
}

type HandlerConstructor func(requestNumber int, limit int, counter chan<- int) http.Handler

// HeavyHandler simulates a handler that performs task that is long enough for the server
// to exhaust the maximum allowed number of concurrent handlers.
// It does that by waiting in each handler until all of the other handlers have started
// to execute. The number of handlers it waits for is decided by the requestNumber parameter.
func HeavyHandler(requestNumber int, limit int, counter chan<- int) http.Handler {
	var preLimiterWg sync.WaitGroup
	preLimiterWg.Add(requestNumber)
	preLimiter := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			preLimiterWg.Done()
			h.ServeHTTP(w, r)
		})
	}

	var handlerWg sync.WaitGroup
	limiter := Limiter(limit)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter <- 1
		handlerWg.Add(1)
		preLimiterWg.Wait()
		handlerWg.Done()
		handlerWg.Wait()
		w.WriteHeader(200)
		counter <- -1
	})

	return alice.New(preLimiter, limiter).Then(handler)
}

// LightHandler simply returns 200 OK status and finishes.
// This is useful to test that the limiter allows next handlers as soon
// as the previous have finished running.
func LightHandler(requestNumber int, limit int, counter chan<- int) http.Handler {
	limiter := Limiter(limit)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter <- 1
		w.WriteHeader(200)
		counter <- -1
	})
	return alice.New(limiter).Then(handler)
}

// ConcurrencyTest is a test helper that makes a given number of requests to a handler
// created by HandlerConstructor that is concurrency limited to a given limit.
// While running, it observes the current actual handler concurrency and returns
// its maximum encountered value, as well as the number of 200 OK (allowed)
// and 429 Too Many Requests (denied) responses received.
func ConcurrencyTest(t *testing.T, hc HandlerConstructor, requestNumber int, limit int) ConcurrencyTestResult {

	counter := make(chan int)
	codes := make(chan int)

	handler := hc(requestNumber, limit, counter)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	var requestsWg sync.WaitGroup
	for i := 0; i < requestNumber; i++ {
		requestsWg.Add(1)
		go func() {
			res, err := http.Get(ts.URL)
			assert.NoError(t, err)
			codes <- res.StatusCode
			requestsWg.Done()
		}()
	}

	var resultsWg sync.WaitGroup
	var result ConcurrencyTestResult

	resultsWg.Add(2)
	go func() {
		var concurrencyNow int
		for number := range counter {
			concurrencyNow += number
			if concurrencyNow > result.maxConcurrency {
				result.maxConcurrency = concurrencyNow
			}
		}
		resultsWg.Done()
	}()
	go func() {
		for number := range codes {
			switch number {
			case http.StatusOK:
				result.accepted++
			case http.StatusTooManyRequests:
				result.denied++
			default:
				assert.Failf(t, "bad response", "unexpected status code: %v", number)
			}
		}
		resultsWg.Done()
	}()

	requestsWg.Wait()
	close(counter)
	close(codes)
	resultsWg.Wait()

	return result
}
