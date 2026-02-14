package main

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
	"github.com/gin-gonic/gin"
)

func main() {
	l, err := zap.NewDevelopment()
	if err != nil { panic(err) }
	r := gin.Default()

	r.GET("/hello", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, "Hello world")
	})

	fmt.Print("started")
	if err := r.Run(":8080"); err != nil {
		l.Panic("Server run failed", zap.Error(err))
	}
}
