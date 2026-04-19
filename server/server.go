package server

import (
	"net"
	"sync"
	"syscall"
)

// Ctx holds connection and buffer
type Ctx struct {
	buf     []byte // pooled
	conn    net.Conn
	scratch [1024]byte // stack scratch
}

// NewCtx creates a new context
func NewCtx() *Ctx {
	return &Ctx{buf: make([]byte, 8192)}
}

var (
	pool   = sync.Pool{New: func() any { return NewCtx() }}
	OkResp = []byte(
		"HTTP/1.1 200 OK\r\n" +
			"Content-Type: text/plain; charset=utf-8\r\n" +
			"Content-Length: 18\r\n" +
			"Connection: keep-alive\r\n\r\n" +
			"hello from pure go")
	HealthResp = []byte(
		"HTTP/1.1 200 OK\r\n" +
			"Content-Type: text/plain; charset=utf-8\r\n" +
			"Content-Length: 4\r\n" +
			"Connection: keep-alive\r\n\r\n" +
			"ok")
	rootKey   = []byte("GET / ")
	healthKey = []byte("GET /health ")
)

// ResponseWriter interface for handlers
type ResponseWriter interface {
	Write([]byte) (int, error)
}

// HandlerFunc type for HTTP handlers
type HandlerFunc func(ResponseWriter, *Ctx)

// Server holds router and connection
type Server struct {
	router Router
}

// NewServer creates a new server with given router
func NewServer(router Router) *Server {
	return &Server{router: router}
}

// Serve handles incoming connections
func (s *Server) Serve(conn net.Conn) {
	defer conn.Close()
	ctx := pool.Get().(*Ctx)
	ctx.conn = conn
	defer pool.Put(ctx)
	rb := ctx.buf[:0] // reset len

	for {
		n, err := conn.Read(rb[len(rb):cap(rb)])
		if err != nil || n == 0 {
			return
		}
		rb = rb[:len(rb)+n]

		// Fast prefix match on bytes
		if len(rb) >= 9 && hasPrefix(rb, rootKey) {
			if handler, ok := s.router.Match("GET", "/"); ok {
				handler(&responseWriter{conn: conn}, ctx)
				drainTo(rb)
				rb = rb[:0]
				continue
			}
		}

		if len(rb) >= 14 && hasPrefix(rb, healthKey) {
			if handler, ok := s.router.Match("GET", "/health"); ok {
				handler(&responseWriter{conn: conn}, ctx)
				drainTo(rb)
				rb = rb[:0]
				continue
			}
		}

		// 404 + close
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\nConnection: close\r\n\r\n"))
		return
	}
}

// responseWriter wraps net.Conn for writing responses
type responseWriter struct {
	conn net.Conn
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.conn.Write(b)
}

// hasPrefix optimized memcmp-like
func hasPrefix(b, prefix []byte) bool {
	if len(b) < len(prefix) {
		return false
	}
	for i := range prefix {
		if b[i] != prefix[i] {
			return false
		}
	}
	return true
}

// drainTo finds \r\n\r\n, discards prefix (zero-copy)
func drainTo(b []byte) {
	for i := 0; i+3 < len(b); i++ {
		if b[i] == '\r' && b[i+1] == '\n' && b[i+2] == '\r' && b[i+3] == '\n' {
			copy(b, b[i+4:])
			return
		}
	}
}

// Start starts the server
func (s *Server) Start(address string) error {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	if tcpln, ok := ln.(*net.TCPListener); ok {
		file, err := tcpln.File()
		if err == nil {
			syscall.SetsockoptInt(int(file.Fd()), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			file.Close()
		}
	}

	const maxGoroutines = 100
	sem := make(chan struct{}, maxGoroutines)

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				continue
			}
			sem <- struct{}{}
			go func() {
				defer func() { <-sem }()
				s.Serve(c)
			}()
		}
	}()

	return nil
}
