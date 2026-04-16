package internal

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"example.com/m/v2/internal/auth"
	"example.com/m/v2/internal/ratelimit"
	"example.com/m/v2/internal/routing"
	"example.com/m/v2/internal/utils"
)

type Router struct {
	trie        *routing.Trie
	roundRobin  *routing.RoundRobin
	client      *http.Client
	rateLimiter *ratelimit.RateLimiter
}

func NewRouter(routes []utils.Route) *Router {
	return &Router{
		trie:        routing.NewTrie(routes),
		roundRobin:  routing.NewRoundRobin(routes),
		client:      &http.Client{},
		rateLimiter: ratelimit.NewRateLimiter(routes),
	}
}

func chain(final http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		middleware := middlewares[i]
		final = middleware(final)
	}
	return final
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	route := router.trie.Match(path)
	if route == nil {
		http.Error(w, "Path does not exist", http.StatusNotFound)
		return
	}

	ctx := context.WithValue(r.Context(), utils.GetRoutePrefixKey(), route.Prefix)

	middlwares := make([]func(http.Handler) http.Handler, 0)

	if route.AuthRequired {
		middlwares = append(middlwares, auth.Middleware)
	}

	if route.RateLimit != nil {
		middlwares = append(middlwares, router.rateLimiter.Middleware)
	}

	finalHandler := router.serve(route, path)
	chain(finalHandler, middlwares...).ServeHTTP(w, r.WithContext(ctx))

}

func (router *Router) call(upstreamUrl string, r *http.Request) (*http.Response, error) {
	req, err := http.NewRequestWithContext(r.Context(), r.Method, fmt.Sprintf("%s%s", upstreamUrl, r.URL.RequestURI()), r.Body)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := router.client.Do(req)

	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	return resp, nil
}

func (router *Router) serve(route *utils.Route) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		success := false

		for i := 0; i < len(route.Upstreams); i++ {
			selectedStreamIndex := router.roundRobin.GetServerIndex(*route)
			selectedStream := route.Upstreams[selectedStreamIndex]

			if selectedStream.CircuitBreaker.Allow(selectedStream.Config.RecoveryWindow) {
				resp, err := router.call(selectedStream.Config.URL, r)
				if err != nil {
					selectedStream.CircuitBreaker.RecordFailure(selectedStream.Config.FailureThreshold, selectedStream.Config.FailureWindow)
					continue
				}

				if resp.StatusCode >= 500 {
					selectedStream.CircuitBreaker.RecordFailure(selectedStream.Config.FailureThreshold, selectedStream.Config.FailureWindow)
					resp.Body.Close()
					continue
				}

				for key, values := range resp.Header {
					for _, value := range values {
						w.Header().Add(key, value)
					}
				}

				w.WriteHeader(resp.StatusCode)
				_, err = io.Copy(w, resp.Body)
				if err != nil {
					log.Println(err.Error())
				}
				resp.Body.Close()

				selectedStream.CircuitBreaker.RecordSuccess()
				success = true
				break
			}

		}

		if !success {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}
	})

}
