.PHONY: install test run

install:
	go install .

test:
	go test ./...

run:
	go run .
