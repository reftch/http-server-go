package main

import (
	"fmt"
	"log"
	"net"
	"sync"
	"syscall"
	"time"
)

type Ctx struct {
	buf     []byte // pooled
	conn    net.Conn
	scratch [1024]byte // stack scratch
}

var (
	pool      = sync.Pool{New: func() any { c := &Ctx{buf: make([]byte, 8192)}; return c }}
	okResp    = []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: 18\r\nConnection: keep-alive\r\n\r\nhello from pure go")
	rootKey   = []byte("GET / ")
	healthKey = []byte("GET /health ")
	reqCount  uint64 // Atomic request counter
)

func main() {
	before := time.Now()

	// SO_REUSEPORT for max concurrency (Linux)
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}

	if tcpln, ok := ln.(*net.TCPListener); ok {
		file, err := tcpln.File()
		if err == nil {
			syscall.SetsockoptInt(int(file.Fd()), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			file.Close()
		}
	}

	after := time.Now()
	startupMicros := after.Sub(before).Microseconds()
	log.Printf("server started at on http://localhost:8080, startup time %d µs", startupMicros)

	const maxGoroutines = 1024
	sem := make(chan struct{}, maxGoroutines)
	go statsPrinter() // Print req/s every 5s

	for {
		c, err := ln.Accept()
		if err != nil {
			continue
		}
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			handleRequest(c)
		}()
	}
}

func statsPrinter() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		fmt.Printf("requests/sec: ~%.0f (total: %d)\n", float64(reqCount)/5, reqCount)
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()
	ctx := pool.Get().(*Ctx)
	ctx.conn = conn
	defer pool.Put(ctx)
	rb := ctx.buf[:0] // reset len

	for {
		// Tight read loop - fasthttp-style
		n, err := conn.Read(rb[len(rb):cap(rb)])
		if err != nil || n == 0 {
			return
		}
		rb = rb[:len(rb)+n]
		reqCount++ // Bump counter

		// Fast prefix match on bytes
		if len(rb) >= 9 && (hasPrefix(rb, rootKey) || hasPrefix(rb, healthKey)) {
			_, _ = conn.Write(okResp)
			// Drain rest of request line/headers for keep-alive
			drainTo(rb)
			rb = rb[:0]
			continue
		}
		// 404 + close
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\nConnection: close\r\n\r\n"))
		return
	}
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
	// Manual search - no strings.Index
	for i := 0; i+3 < len(b); i++ {
		if b[i] == '\r' && b[i+1] == '\n' && b[i+2] == '\r' && b[i+3] == '\n' {
			copy(b, b[i+4:])
			return
		}
	}
}
