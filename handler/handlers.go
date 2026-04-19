package handler

import (
	"reftch.com/http-server/server"
)

// HandlerFunc type for HTTP handlers
type HandlerFunc func(*server.Ctx)

// HomeHandler returns home page
func HomeHandler(w server.ResponseWriter, ctx *server.Ctx) {
	w.Write(server.OkResp)
}

// HealthHandler returns health check
func HealthHandler(w server.ResponseWriter, ctx *server.Ctx) {
	w.Write(server.HealthResp)
}

// NotFoundHandler returns 404
func NotFoundHandler(w server.ResponseWriter, ctx *server.Ctx) {
	w.Write([]byte("HTTP/1.1 404 Not Found\r\nConnection: close\r\n\r\n"))
}
