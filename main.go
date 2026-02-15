package main

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var (
	upg   = &websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conns = make(map[*websocket.Conn]bool)
	lastOffer []byte
	mu    sync.Mutex
)

func main() {
	l, err := zap.NewDevelopment()
	if err != nil { panic(err) }
	r := gin.Default()

	upg = &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conns = make(map[*websocket.Conn]bool)

	r.GET("/", func(ctx *gin.Context) {
		ctx.File("frontend/index.html")
	})

	r.GET("/ws", func(ctx *gin.Context) {
		conn, err := upg.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			l.Error("WS upgrade failed", zap.Error(err))
			http.Error(ctx.Writer, err.Error(), http.StatusBadRequest)
			return
		}
		defer conn.Close()

		mu.Lock()
		conns[conn] = true
		if lastOffer != nil && !ctx.Request.URL.Query().Has("stream") {
			conn.WriteMessage(websocket.TextMessage, lastOffer)
		}
		mu.Unlock()

		defer func(){
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

	if err := r.Run("0.0.0.0:8080"); err != nil {
		l.Panic("Server run failed", zap.Error(err))
	}
}
