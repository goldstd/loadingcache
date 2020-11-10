generate:
	@echo 'Generating files...'
	go generate github.com/Hartimer/loadingcache/...

test: generate
	@echo 'Running all tests...'
	go test github.com/Hartimer/loadingcache/... -timeout 5s -count 1

test-race:
	# Does not run on arm processors. See https://github.com/golang/go/issues/25682
	@echo 'Running all tests with race detection...'
	go test -race github.com/Hartimer/loadingcache/... -timeout 5s -count 1

lint:
	@echo 'Running golangci-lint...'
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v1.32.2 golangci-lint run