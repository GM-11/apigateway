package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"example.com/m/v2/internal/auth"
)

type Router struct {
	trie       *Trie
	roundRobin *RoundRobin
	client     *http.Client
}

type authHeaderKeyType string

const authHeaderKey authHeaderKeyType = "authHeader"

func (router *Router) ServeHTTP(path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := router.trie.Match(path)
		if route == nil {
			http.Error(w, "Path does not exist", http.StatusBadRequest)
			return
		}

		if route.AuthRequired {
			authHeader := r.Header.Get("Authorization")
			ctx := context.WithValue(r.Context(), authHeaderKey, authHeader)
			auth.Middleware(router.serve(route, path)).ServeHTTP(w, r.WithContext(ctx))
		} else {
			router.serve(route, path).ServeHTTP(w, r)
		}

	})
}

func (router *Router) serve(route *Route, path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		selectedStreamIndex := router.roundRobin.GetServerIndex(*route)
		selectedStream := route.Upstreams[selectedStreamIndex]

		url := fmt.Sprintf("%s%s", selectedStream, path)

		req, err := http.NewRequest(r.Method, url, r.Body)

		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
		}

		if route.AuthRequired {
			authHeader := r.Context().Value(authHeaderKey).(string)
			if len(authHeader) == 0 {
				http.Error(w, "Something Went Wrong", http.StatusInternalServerError)
				return
			}
			req.Header.Add("Authorization", authHeader)
		}

		for header, values := range r.Header {
			for _, value := range values {
				req.Header.Add(header, value)
			}
		}

		resp, err := router.client.Do(req)
		if err != nil {
			log.Println(err)
			http.Error(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		for header, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(header, value)
			}
		}

		var response any

		json.NewDecoder(resp.Body).Decode(&response)
		w.WriteHeader(resp.StatusCode)
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Println(err)
			http.Error(w, "Error encoding json", http.StatusInternalServerError)
			return
		}
	})

}
