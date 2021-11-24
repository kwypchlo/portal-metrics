#!/bin/bash

# Only uncomment this line if 'go' is installed, otherwise provide the binary.
#(cd splitter && go build && cp nginx-data-splitter ../) || exit 1
#(cd stats-builder && go build && cp stats ../) || exit 1
#./nginx-data-splitter access.log || exit 1

# TODO: This assumes a specific folder structure on the server.
cd /home/user/metrics
./nginx-data-splitter ../skynet-webportal/docker/data/nginx/logs/access.log || exit 1

# Get the list of logfiles for each day. Ignore the final file, we assume it is
# incomplete.
dayFiles=$(ls -1 days | head -n -1)

mkdir -p main
mkdir -p apps

# Determine how many files have been processed already, remove those from
# dayFiles
latestScan=$(cat latestScan.txt || echo 0)

# Go through the files one at a time.
for dayFile in $dayFiles
do
	# If we have already scanned this day, keep going.
	if [[ "$dayFile" < "$latestScan" ]];
	then
		continue
	fi
	# Can't do '<=' so we need another statement.
	if [[ "$dayFile" == "$latestScan" ]];
	then
		continue
	fi

	./stats days/$dayFile || exit 1

	# Update the latest day.
	echo $dayFile > latestScan.txt
done
