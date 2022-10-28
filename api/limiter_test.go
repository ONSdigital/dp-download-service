package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLimiter(t *testing.T) {

	limit := 1
	limiter := Limiter(limit)

	var counter int64

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		concurrentHandlers := atomic.AddInt64(&counter, 1)
		assert.LessOrEqual(t, concurrentHandlers, int64(limit))
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
	}))
	handler = limiter(handler)

	req, err := http.NewRequest("GET", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			atomic.AddInt64(&counter, -1)

			assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
			wg.Done()
		}()
	}

	wg.Wait()
}
