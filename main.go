package main

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

//go:embed frontend/index.html
var indexHTML []byte

var (

	upg       = &websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conns     = make(map[*websocket.Conn]bool)
	lastOffer []byte
	mu        sync.Mutex
)

func main() {
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	r := http.NewServeMux()

	upg = &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conns = make(map[*websocket.Conn]bool)

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(indexHTML)
	})

	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upg.Upgrade(w, r, nil)
		if err != nil {
			l.Error("WS upgrade failed", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer conn.Close()

		mu.Lock()
		conns[conn] = true
		if lastOffer != nil && !r.URL.Query().Has("stream") {
			conn.WriteMessage(websocket.TextMessage, lastOffer)
		}
		mu.Unlock()

		defer func() {
			mu.Lock()
			delete(conns, conn)
			mu.Unlock()
			conn.Close()
		}()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				l.Error("Read message failed", zap.Error(err))
				break
			}

			var data map[string]any
			if err := json.Unmarshal(msg, &data); err != nil {
				if _, has := data["sdp"]; has {
					mu.Lock()
					lastOffer = msg
					mu.Unlock()
				}
			}

			mu.Lock()
			for c := range conns {
				if c != conn {
					c.WriteMessage(websocket.TextMessage, msg)
				}
			}
			mu.Unlock()
		}
	})

	s := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	if err := s.ListenAndServe(); err != nil {
		l.Panic("Server run failed", zap.Error(err))
	}
}
