TAGS := desktop,production,webkit2_41

.PHONY: frontend build test

frontend:
	cd frontend && npm ci && npm run build

build: frontend
	go build -tags $(TAGS) -o skye ./cmd/skye

test:
	go test ./internal/...
	cd frontend && npm test
