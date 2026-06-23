.PHONY: check fmt install run test

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

install:
	go install .

test:
	go test ./...

check:
	test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*'))"
	go test -race ./...
	go vet ./...

run:
	go run .
