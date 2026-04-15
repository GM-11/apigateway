package routing

import (
	"net/http"
	"net/http/httputil"

	"example.com/m/v2/internal/auth"
	"github.com/google/uuid"
)

type Router struct {
	trie       *Trie
	roundRobin *RoundRobin
	client     *http.Client
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	route := router.trie.Match(path)
	if route == nil {
		http.Error(w, "Path does not exist", http.StatusNotFound)
		return
	}

	if route.AuthRequired {
		auth.Middleware(router.serve(route, path)).ServeHTTP(w, r)
	} else {
		router.serve(route, path).ServeHTTP(w, r)
	}

}

func (router *Router) serve(route *Route, path string) http.Handler {
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
