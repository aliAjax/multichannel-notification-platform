SHELL := /bin/sh
APP := notification-api

.PHONY: fmt test vet race verify run smoke build clean
fmt:
	gofmt -w $$(find . -name '*.go')
test:
	go test ./...
vet:
	go vet ./...
race:
	go test -race ./...
verify: fmt vet test race
build:
	go build -o build/$(APP) ./cmd/notification-api
run:
	go run ./cmd/notification-api
smoke:
	./scripts/smoke.sh
clean:
	rm -rf build data

