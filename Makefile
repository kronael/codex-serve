.PHONY: build test smoke lint clean run

BINARY_NAME=codex-serve

build:
	@echo "building..."
	go build -o ./$(BINARY_NAME) .

test:
	@echo "running unit tests (mocked)..."
	go test -v -short -timeout 5s ./...

smoke:
	@echo "running smoke tests (real codex CLI)..."
	@echo "Note: requires codex CLI installed and authenticated"
	go test -v -timeout 30s ./...

lint:
	@echo "linting..."
	go vet ./...
	go fmt ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "warning: golangci-lint not installed (optional)"; \
	fi

run: build
	@echo "starting server..."
	./$(BINARY_NAME)

clean:
	@echo "cleaning..."
	@rm -f ./$(BINARY_NAME)
	@rm -rf ./tmp
	@rm -rf ./log
	go clean
