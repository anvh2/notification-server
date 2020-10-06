BIN = notification-server

clean:
	rm -f $(BIN)

build: clean
	go mod vendor
	GOOS=linux go build -mod vendor -o $(BIN)

genpb:
	protoc --proto_path=idl \
		--go_out=plugins=grpc:./grpc-gen \
		-I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		-I$$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway \
		idl/notification.proto

rsync:
	rsync -avz config.* $(BIN) root@104.248.148.244:/source/${BIN}
	ssh root@104.248.148.244 sh /source/${BIN}/runserver restart

rsync-runserver:
	rsync -avz --perms --chmod=700 runserver root@104.248.148.244:/source/${BIN}

deploy: build rsync

set-up:
	mkdir ./log && cd ./log && touch server.log && cd ..

run-local: build 
	./$(BIN) start --config config.local.toml