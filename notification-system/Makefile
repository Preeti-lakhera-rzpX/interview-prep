.PHONY: build test race cover fmt vet lint check tidy

build:
	go build ./...

test:
	go test ./...

race:
	go test -race ./...

cover:
	go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

fmt:
	gofmt -l -w .

vet:
	go vet ./...

# Built-in lint: fail if anything is unformatted, then vet.
lint:
	@test -z "$$(gofmt -l .)" || (echo "unformatted files:"; gofmt -l .; exit 1)
	go vet ./...

tidy:
	go mod tidy

# One-shot sanity gate.
check: lint test
