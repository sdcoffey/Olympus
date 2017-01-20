pkgs = $(shell glide novendor)

all: build-all

clean:
	@rm -rf build
	@go fmt $(pkgs)

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

install: 
	@ps aux | grep [o]lympus | awk '{print $$2}' | xargs kill -9
	cp build/bin/server /usr/local/bin/olympus
	@olympus&

install-cli:
	cp build/bin/cli /usr/local/bin/olympus-cli

test: clean
	@for package in $(pkgs); do \
		go test ./$$package ; \
		if [ -d "./$$package/test" ]; then go test ./$$package/test ; fi \
	done

testcover: clean
	@go test -cover $(pkgs)

