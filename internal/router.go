package internal

import (
	"context"
	"net/http"
	"strings"

	"example.com/m/v2/internal/auth"
	"example.com/m/v2/internal/metrics"
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

	middlwares := []func(http.Handler) http.Handler{
		metrics.Middleware,
	}

	if route.AuthRequired {
		middlwares = append(middlwares, auth.Middleware)
	}

	if route.RateLimit != nil {
		middlwares = append(middlwares, router.rateLimiter.Middleware)
	}

	connection := r.Header.Get("Connection")
	upgrade := r.Header.Get("Upgrade")

	isUpgrade := strings.Contains(strings.ToLower(connection), "upgrade") &&
		strings.EqualFold(upgrade, "websocket")

	var finalHandler http.Handler

	if isUpgrade {
		finalHandler = router.serveWebSocket(route)
	} else {
		finalHandler = router.serve(route)
	}
	chain(finalHandler, middlwares...).ServeHTTP(w, r.WithContext(ctx))

}
