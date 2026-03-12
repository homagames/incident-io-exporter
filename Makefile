BINARY_NAME := incident-io-exporter
COVERAGE_THRESHOLD := 80

.PHONY: all build test test-coverage lint docker-build helm-lint clean

all: lint test build

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/$(BINARY_NAME) .

test:
	go test -race ./...

test-coverage:
	go test -race -coverprofile=coverage.out ./...
	@total=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Total coverage: $${total}%"; \
	threshold=$(COVERAGE_THRESHOLD); \
	if [ $$(echo "$${total} < $${threshold}" | bc) -eq 1 ]; then \
		echo "Coverage $${total}% is below threshold $${threshold}%"; \
		exit 1; \
	fi

lint:
	golangci-lint run ./...

docker-build:
	docker build -t $(BINARY_NAME):local .

helm-lint:
	helm lint charts/$(BINARY_NAME)

clean:
	rm -rf bin/ coverage.out
