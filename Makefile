.PHONY: all clean rtcgw worker

all: dhis2gw worker

clean:
	rm -f rtcgw-go workers/workers

dhis2gw:
	go build -ldflags="-s -w" -o dhis2gw

worker:
	go build  -ldflags="-s -w" -o workers/workers ./workers

run-server: dhis2gw
	./dhis2gw

run-worker: worker
	./workers/workers