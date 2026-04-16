package internal

import (
	"context"
	"net/http"
	"net/http/httputil"

	"example.com/m/v2/internal/auth"
	"example.com/m/v2/internal/ratelimitting"
	"example.com/m/v2/internal/routing"
	"example.com/m/v2/internal/utils"

	"github.com/google/uuid"
)

type Router struct {
	trie        *routing.Trie
	roundRobin  *routing.RoundRobin
	client      *http.Client
	rateLimiter *ratelimitting.RateLimiter
}

func NewRouter(routes []utils.Route) *Router {
	// routing.BuildTrie()
	return &Router{
		trie:        routing.NewTrie(routes),
		roundRobin:  routing.NewRoundRobin(routes),
		client:      &http.Client{},
		rateLimiter: ratelimitting.NewRateLimiter(routes),
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

func (router *Router) serve(route *utils.Route, path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		selectedStreamIndex := router.roundRobin.GetServerIndex(*route)
		selectedStream := route.Upstreams[selectedStreamIndex]

		requestId := uuid.New().String()

		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = "http"
				req.URL.Host = selectedStream
				req.URL.Path = path
				req.Header.Set("X-Request-ID", requestId)
			},
		}

		proxy.ServeHTTP(w, r)
	})

}
