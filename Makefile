all: dependencies build

dependencies:
	(cd cmd/metrics-joiner && go build && cp joiner ../../build/)
	(cd cmd/nginx-log-filter && go build && mv filter ../../build/)
	(cd cmd/power-analyzer && go build && cp power ../../build/)
	(cd cmd/stats-builder && go build && cp stats ../../build/)
	(cd cmd/uploader-finder && go build && mv banfinder ../../build/)

build:
	./update.sh
	./build-banlist.sh
	./fetch-metrics.sh
	./combine-metrics.sh

clean:
	./reset.sh
