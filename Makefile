.PHONY: all clean api dhis2gw worker migrate linux-build linux-release run-server run-worker release-check release-snapshot release

DHIS2GW_VERSION ?= 1.0.0

all: api worker

clean:
	rm -f dhis2gw worker migrate

api:
	swag init -g cmd/api/main.go -o docs
	go build -ldflags="-s -w" -o dhis2gw ./cmd/api

worker:
	go build -ldflags="-s -w" -o worker ./cmd/worker

migrate:
	go build -ldflags="-s -w" -o migrate ./cmd/migrate

linux-build:
	DHIS2GW_VERSION="$(DHIS2GW_VERSION)" ./scripts/linux-build.sh

linux-release:
	goreleaser release --snapshot --clean

dhis2gw: api

run-server: dhis2gw
	./dhis2gw

run-worker: worker
	./worker

release-check:
	goreleaser check

release-snapshot:
	goreleaser release --snapshot --clean

release:
	goreleaser release --clean
