
pkgs = client/apiclient	client/cli	client/shared	env	fs	peer	server	server/api

all: build

clean:
	@rm -rf build
	go fmt ./...
	@for package in $(pkgs); do \
		goimports -w ./$$package ; \
	done

build: test
	@mkdir -p build/executables
	go build -o build/executables/server github.com/sdcoffey/olympus/server
	go build -o build/executables/cli github.com/sdcoffey/olympus/client/cli

test: clean
	go test -v ./...

cover:
		go test -coverprofile=build/cover/$name.out ./$$package
		go tool cover -html=build/cover/$name.out -o=build/cover/html/$name.html
	done

testcover: clean
	go test -cover ./...

