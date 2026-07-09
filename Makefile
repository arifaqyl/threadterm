.PHONY: build run demo test tidy install release-snapshot

build:
	go build -ldflags "-X github.com/arifaqyl/threadterm/internal/cli.Version=dev" -o threadterm.exe ./cmd/threadterm

run: build
	./threadterm.exe --demo

demo: run

test:
	go test ./...

tidy:
	go mod tidy

install:
	go install ./cmd/threadterm

release-snapshot:
	goreleaser release --snapshot --clean
