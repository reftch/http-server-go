.PHONY: certs dev build
.SILENT:

certs:
	openssl req -x509 -newkey rsa:4096 -keyout certs/server.key -out certs/server.crt -days 365 -nodes

dev:
	wgo run cmd/main.go

build:
	GOARCH=amd64 go build -ldflags="-s -w" -o build/server cmd/main.go

clean:
	rm -rf build