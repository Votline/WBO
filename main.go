package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
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
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

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
				log.Println("Read message error: " + err.Error())
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
		panic("Server run failed" + err.Error())
	}
}
