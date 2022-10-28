package api

import "net/http"

func Limiter(n int) func(http.Handler) http.Handler {
	counter := make(chan struct{}, n)
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if n < 1 {
				h.ServeHTTP(w, r)
				return
			}
			counter <- struct{}{}
			defer func() { <-counter }()

			h.ServeHTTP(w, r)
		})
	}
}

var n = 5

func LimiterMidd(h http.Handler) http.Handler {
	counter := make(chan struct{}, n)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter <- struct{}{}
		defer func() { <-counter }()

		h.ServeHTTP(w, r)
	})
}

func LimiterHandler(h http.Handler, n int) http.Handler {
	counter := make(chan struct{}, n)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter <- struct{}{}
		defer func() { <-counter }()

		h.ServeHTTP(w, r)
	})
}
