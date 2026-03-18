.PHONY: build install clean test

build:
	go build -o plaincode ./cmd/plaincode/

install:
	./install.sh

clean:
	rm -f plaincode

test:
	go test ./...
