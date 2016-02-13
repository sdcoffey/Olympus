
pkgs = client/apiclient	client/cli	client/shared	env	graph	peer	server	server/api

all: build-all

clean:
	@rm -rf build
	@go fmt ./...
	@for package in $(pkgs); do \
		goimports -w ./$$package ; \
	done

build-all: clean
	@mkdir -p build/bin
	go build -o build/bin/server github.com/sdcoffey/olympus/server
	go build -o build/bin/cli github.com/sdcoffey/olympus/client/cli

build: clean
	@mkdir -p build/bin
	go build -o build/bin/server github.com/sdcoffey/olympus/server

build-cli: clean
	@mkdir -p build/bin
	go build -o build/bin/cli github.com/sdcoffey/olympus/client/cli

install: copy-resources
	@ps aux | grep [o]lympus | awk '{print $$2}' | xargs kill -9
	cp build/bin/server /usr/local/bin/olympus
	@olympus&

copy-resources:
	mkdir -p /usr/local/etc/olympus/www
	cp -R client/www/app/* /usr/local/etc/olympus/www

install-cli:
	cp build/bin/cli /usr/local/bin/olympus-cli

test: clean copy-resources
	@go test -v ./...

testcover: clean
	@go test -cover ./...

