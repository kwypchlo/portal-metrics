all: dependencies update merge

dependencies: splitter stats-builder metrics-joiner power-analyzer

splitter:
	(cd splitter && go build && cp nginx-data-splitter ../)

stats-builder:
	(cd stats-builder && go build && cp stats ../)

metrics-joiner:
	(cd metrics-joiner && go build && cp joiner ../)

power-analyzer:
	(cd power-analyzer && go build && cp power ../)

update:
	./update.sh

merge:
	./merge.sh
