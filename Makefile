build: build-client build-server

build-client:
	go build -o bin/client cmd/client/*.go
	sudo setcap cap_net_raw,cap_net_admin+eip bin/client

build-server:
	go build -o bin/server cmd/server/*.go

run-client:
	sudo setcap cap_net_raw,cap_net_admin+eip $(which go)
	go run cmd/client/main.go

run-server:
	go run cmd/server/main.go

install:
	go mod tidy

install-go:
	bash scripts/install-go.sh

compile: compile-client compile-server

compile-client:
	echo "Compiling Client for ARM"
	GOOS=linux GOARCH=arm go build -o bin/client cmd/client/*.go
	sudo setcap cap_net_raw,cap_net_admin+eip bin/client

compile-server:
	echo "Compiling Server for ARM"
	GOOS=linux GOARCH=arm go build -o bin/server cmd/server/*.go

run-compiled-client:
	chmod +x bin/client
	./bin/client

run-compiled-server:
	chmod +x bin/server
	./bin/server

client: install compile-client run-compiled-client

server: install compile-server run-compiled-server

server-devaus: install build-server run-compiled-server
