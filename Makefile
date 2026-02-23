.PHONY: build test test-cover lint run docker-up docker-down fixture

build:
	go build ./...

test:
	go test ./... -v -count=1

test-cover:
	go test ./... -coverprofile=coverage.out

lint:
	go vet ./...

run:
	go run ./cmd/server

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

fixture:
	@if [ -z "$(URL)" ] || [ -z "$(OUT)" ]; then \
		echo "Usage: make fixture URL=https://... OUT=testdata/..."; \
		exit 1; \
	fi
	./scripts/capture-fixture.sh "$(URL)" "$(OUT)"
