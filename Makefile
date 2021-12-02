all: dependencies

dependencies:
	(cd cmd/nginx-log-indexer && go build && mv indexer ../../build/)
	(cd cmd/stats-builder && go build && cp stats ../../build/)
	(cd cmd/metrics-joiner && go build && cp joiner ../../build/)
	(cd cmd/power-analyzer && go build && cp power ../../build/)
	(cd cmd/uploader-finder && go build && mv finder ../../build/)
