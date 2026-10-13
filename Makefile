.PHONY: all clean api dhis2gw worker migrate run-server run-worker release-check release-snapshot release

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

dhis2gw: api

run-server: dhis2gw
	./dhis2gw

run-worker: worker
	./worker

release-check:
	goreleaser check

release-snapshot:
	goreleaser release --snapshot --rm-dist

release:
	goreleaser release --rm-dist
