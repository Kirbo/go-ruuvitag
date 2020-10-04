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

run-compiled-client:
	chmod +x bin/client
	./bin/client

run-compiled-server:
	chmod +x bin/server
	./bin/server

client: install build-client run-compiled-client

server: install build-server run-compiled-server


startup-client:
	sudo bash scripts/install-client-startup.sh ${HOME}

startup-server:
	sudo bash scripts/install-server-startup.sh ${HOME}

logs-client:
	sudo journalctl -u ruuvitag-client.service

logs-server:
	sudo journalctl -u ruuvitag-server.service

follow-client:
	sudo journalctl -u ruuvitag-client.service --follow

follow-server:
	sudo journalctl -u ruuvitag-server.service --follow
