
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
	go build -o build/executables/server sdcoffey/olympus/server
	go build -o build/executables/cli/cli github.com/sdcoffey/olympus/client/cli

test: clean
	go test -v ./...

cover:
		go test -coverprofile=build/cover/$name.out ./$$package
		go tool cover -html=build/cover/$name.out -o=build/cover/html/$name.html
	done

cover: clean
	@mkdir -p build/cover
	@mkdir -p build/cover/html
	@for package in $(pkgs); do \
		echo $$package \
		name=basename $$package \
	done

