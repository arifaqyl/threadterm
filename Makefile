.PHONY: build run demo test tidy

build:
	go build -o threadterm.exe ./cmd/threadterm

run: build
	./threadterm.exe --demo

demo: run

test:
	go test ./...

tidy:
	go mod tidy

install:
	go install ./cmd/threadterm
