SRC := $(shell find . -name "*.go" -not -path "./vendor/*" -not -name "*_test.go")
TESTS := $(shell find . -name "*_test.go" -not -path "./vendor/*")

BINARY=bmo

format: $(SRC)
	@goimports -w $?
	@gofmt -s -w $?

test: $(TESTS)
	@go test -v -cover $(SRC) $(TESTS)

build: bin/$(BINARY)
bin/$(BINARY): $(SRC)
	@go build -o $@ .
	
deps:
	@dep ensure
	
clean:
	@rm -v bin/*; trap

.PHONY: format test build deps clean