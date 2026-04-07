APP=orbit
DAEMON=orbitd
VERSION?=dev
LDFLAGS=-s -w -X github.com/saad/orbit/internal/version.Version=$(VERSION)

.PHONY: build test clean

build:
	go build -ldflags "$(LDFLAGS)" ./cmd/$(APP)
	go build -ldflags "$(LDFLAGS)" ./cmd/$(DAEMON)

test:
	go test ./...

clean:
	rm -f $(APP) $(DAEMON)