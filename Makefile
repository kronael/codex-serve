.PHONY: build test smoke lint clean run

BINARY_NAME=codex-serve
BUILD_DIR=./dist

build:
	@echo "building..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

test:
	@echo "running unit tests..."
	go test -v -short -timeout 5s ./...

smoke:
	@echo "running integration tests..."
	go test -v -timeout 80s ./...

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
	$(BUILD_DIR)/$(BINARY_NAME)

clean:
	@echo "cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf ./tmp
	@rm -rf ./log
	go clean
