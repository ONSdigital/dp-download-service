package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLimiterConcurrency(t *testing.T) {
	maxConcurrency := MaxConcurrency(t, 0, 0) // no requests and no limits set
	assert.Equal(t, 0, maxConcurrency)        // no concurrent requests

	maxConcurrency = MaxConcurrency(t, 0, 10) // no requests and limit of 10 set
	assert.Equal(t, 0, maxConcurrency)        // no concurrent requests

	maxConcurrency = MaxConcurrency(t, 100, 0) // 100 requests and no limits set
	assert.Equal(t, 100, maxConcurrency)       // 100 concurrent requests observed

	maxConcurrency = MaxConcurrency(t, 100, 1) // 100 requests and limit of 1 set
	assert.Equal(t, 1, maxConcurrency)         // 1 concurrent request observed

	maxConcurrency = MaxConcurrency(t, 100, 10) // 100 requests and limit of 10 set
	assert.Equal(t, 10, maxConcurrency)         // 10 concurrent requests observed

	maxConcurrency = MaxConcurrency(t, 100, 100) // 100 requests and limit of 100 set
	assert.Equal(t, 100, maxConcurrency)         // 100 concurrent requests observed

	maxConcurrency = MaxConcurrency(t, 100, 1000) // 100 requests and limit of 1000 set
	assert.Equal(t, 100, maxConcurrency)          // 100 concurrent requests observed
}

func TestStatusCodes(t *testing.T) {
	accepted, denied := StatusCodes(t, 0, 0) // no requests and no limits set
	assert.Equal(t, 0, accepted)
	assert.Equal(t, 0, denied)

	accepted, denied = StatusCodes(t, 0, 10) // no requests and limit of 10 set
	assert.Equal(t, 0, accepted)
	assert.Equal(t, 0, denied)

	accepted, denied = StatusCodes(t, 100, 0) // 100 requests and no limits set
	assert.Equal(t, 100, accepted)
	assert.Equal(t, 0, denied)

	accepted, denied = StatusCodes(t, 100, 1) // 100 requests and limit of 1 set
	assert.Equal(t, 1, accepted)
	assert.Equal(t, 99, denied)

	accepted, denied = StatusCodes(t, 100, 10) // 100 requests and limit of 10 set
	assert.Equal(t, 10, accepted)
	assert.Equal(t, 90, denied)

	accepted, denied = StatusCodes(t, 100, 100) // 100 requests and limit of 100 set
	assert.Equal(t, 100, accepted)
	assert.Equal(t, 0, denied)

	accepted, denied = StatusCodes(t, 100, 1000) // 100 requests and limit of 1000 set
	assert.Equal(t, 100, accepted)
	assert.Equal(t, 0, denied)

}

func MaxConcurrency(t *testing.T, requestNumber int, limit int) int {

	counter := make(chan int)

	limiter := Limiter(limit)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter <- 1
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(200)
	}))

	var wg sync.WaitGroup
	for i := 0; i < requestNumber; i++ {
		wg.Add(1)
		go func() {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, nil)
			counter <- -1

			//assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
			wg.Done()
		}()
	}

	wait := make(chan struct{})
	var concurrencyMax int
	go func() {
		var concurrencyNow int
		for number := range counter {
			concurrencyNow += number
			if concurrencyNow > concurrencyMax {
				concurrencyMax = concurrencyNow
			}
		}
		close(wait)
	}()

	wg.Wait()
	close(counter)
	<-wait
	return concurrencyMax
}

func StatusCodes(t *testing.T, requestNumber int, limit int) (int, int) {

	codes := make(chan int)

	limiter := Limiter(limit)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(200)
	}))

	var wg sync.WaitGroup
	for i := 0; i < requestNumber; i++ {
		wg.Add(1)
		go func() {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, nil)
			codes <- rec.Result().StatusCode

			wg.Done()
		}()
	}

	wait := make(chan struct{})
	var accepted, denied int
	go func() {

		for number := range codes {
			switch number {
			case http.StatusOK:
				accepted++
			case http.StatusTooManyRequests:
				denied++
			default:
				assert.Failf(t, "bad response", "unexpected status code: %v", number)
			}
		}
		close(wait)
	}()

	wg.Wait()
	close(codes)
	<-wait
	return accepted, denied
}
