build:
	go build -o bin/ ./cmd/...

fmt:
	go fmt ./cmd/...
	go fmt ./app/...
