package api

import "net/http"

// Limiter is a middleware that makes sure that no more than n handlers
// are being executed in parallel.
//
// If more traffic comes in while all n handlers are running, the Limiter
// drops the extra requests by returning HTTP status code of 429 (Too Many Requests)
// and never executing the inner handler.
//
// If n is 0, no concurrency limits are applied at all.
func Limiter(n int) func(http.Handler) http.Handler {
	counter := make(chan struct{}, n)
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if n < 1 {
				h.ServeHTTP(w, r)
				return
			}
			select {
			case counter <- struct{}{}:
				defer func() { <-counter }()
				h.ServeHTTP(w, r)
			default:
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
		})
	}
}
