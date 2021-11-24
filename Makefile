all: dependencies update merge

dependencies:
	(cd nginx-log-splitter && go build && cp splitter ../server-updater)
	(cd stats-builder && go build && cp stats ../server-updater)
	(cd metrics-joiner && go build && cp joiner ../)
	(cd power-analyzer && go build && cp power ../)

update:
	./update.sh

merge:
	./merge.sh
