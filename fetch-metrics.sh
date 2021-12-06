#!/bin/bash

# Get the list of servers
servers=$(ls -1 build/server-keys)

# Get the keys for each server.
for server in $servers
do
	# Declare we are working on the server, and scan the host's key.
	keyKnown=$(cat build/server-keys/$server)
	if [[ "$keyKnown" != "true" ]];
	then
		ssh-keyscan $server >> ~/.ssh/known_hosts
		echo "true" > build/server-keys/$server
	fi
done

# Run the metrics collector on every server and download the results as a
# tar.gz.
mkdir -p build/servers
for server in $servers
do
	echo "running server-metrics.sh on $server"
	scp server-metrics.sh $server:/home/user/metrics/ || continue
	scp build/stats $server:/home/user/metrics/ || continue
	ssh $server "/home/user/metrics/server-metrics.sh && tar -czf metric-results.tar.gz apps main latestScan.txt" || continue
	scp $server:/home/user/metrics/metric-results.tar.gz data/servers/$server-metric-results.tar.gz || continue
done
