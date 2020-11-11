GOLANGCI_VERSION=v1.32.2

generate:
	@echo 'Generating files...'
	go generate ./...

test-fast:
	@echo 'Running all tests (no race nor coverage)...'
	go test ./... -timeout 5s

bench:
	@echo 'Running all benchmarks...'
	go test -benchmem -bench .

test:
	# Does not run on arm processors. See https://github.com/golang/go/issues/25682
	# TODO Add if conditional on architecture
	@echo 'Running all tests...'
	go test -race -coverprofile=coverage.txm -covermode=atomic ./... -timeout 5s

lint:
	@echo 'Running golangci-lint...'
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:$(GOLANGCI_VERSION) golangci-lint run