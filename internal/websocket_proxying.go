package internal

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"

	"example.com/m/v2/internal/utils"
)

func (router *Router) callWs(w http.ResponseWriter, r *http.Request, upstreamURL string) error {
	parsed, err := url.Parse(upstreamURL)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Failed to parse upstream URL", http.StatusInternalServerError)
		return err
	}
	conn, err := net.Dial("tcp", parsed.Host)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Failed to establish tcp client", http.StatusBadGateway)
		return err
	}
	defer conn.Close()

	err = r.Write(conn)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Failed to write to upstream", http.StatusInternalServerError)
		return err
	}

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, r)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Failed to connect", http.StatusInternalServerError)
		return err
	}

	if resp.StatusCode != 101 {
		log.Println(resp.StatusCode)
		http.Error(w, "Upstream rejected the request", http.StatusBadRequest)
		return fmt.Errorf("upstream rejected websocket upgrade: %d", resp.StatusCode)
	}

	hijacker, ok := w.(http.Hijacker)

	if !ok {
		http.Error(w, "websocket not supported", http.StatusInternalServerError)
		return err
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		return err
	}

	clientConn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\n"))

	for key, values := range resp.Header {
		for _, value := range values {
			clientConn.Write([]byte(key + ": " + value + "\r\n"))
		}
	}

	clientConn.Write([]byte("\r\n"))

	done := make(chan struct{}, 2)

	go func() {
		io.Copy(conn, clientConn)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(clientConn, conn)
		done <- struct{}{}
	}()

	<-done

	clientConn.Close()

	<-done

	return nil
}

func (router *Router) serveWebSocket(route *utils.Route) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		success := false

		for i := 0; i < len(route.Upstreams); i++ {
			selectedStreamIndex := router.roundRobin.GetServerIndex(*route)
			selectedStream := route.Upstreams[selectedStreamIndex]

			if selectedStream.CircuitBreaker.Allow(selectedStream.Config.RecoveryWindow) {
				err := router.callWs(w, r, selectedStream.Config.URL)
				if err != nil {
					selectedStream.CircuitBreaker.RecordFailure(selectedStream.Config.FailureThreshold, selectedStream.Config.FailureWindow)
					continue
				}

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
