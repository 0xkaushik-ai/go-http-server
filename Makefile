BINARY := bin/httpserver

.PHONY: run build test fmt vet clean

## run: start the server on :8080
run:
	go run .

## build: compile a binary into ./bin
build:
	go build -o $(BINARY) .

## test: run all tests with the race detector
test:
	go test -race ./...

## fmt: format all Go code
fmt:
	gofmt -w .

## vet: run static analysis
vet:
	go vet ./...

## clean: remove build output
clean:
	rm -rf bin
