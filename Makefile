
pkgs = client/apiclient	client/cli	client/shared	env	fs	peer	server	server/api

all: build

clean:
	@rm -rf build
	@go fmt ./...
	@for package in $(pkgs); do \
		goimports -w ./$$package ; \
	done

build: clean
	@mkdir -p build/executables
	@go build -x -o build/executables/server github.com/sdcoffey/olympus/server
	@go build -x -o build/executables/cli github.com/sdcoffey/olympus/client/cli

install:
	cp build/executables/server /usr/local/bin/olympus
	@olympus&

install-cli:
	cp build/executables/cli /usr/local/bin/olympus-cli

test: clean
	@go test -v ./...

testcover: clean
	@go test -cover ./...

