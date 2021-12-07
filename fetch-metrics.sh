#!/bin/bash

# Get the list of servers
servers=$(ls -1 build/server-keys)

# Run the metrics collector on every server and download the results as a
# tar.gz.
mkdir -p build/servers
for server in $servers
do
	echo "running server-metrics.sh on $server"
	scp server-metrics.sh $server:/home/user/metrics/ || continue
	scp build/stats $server:/home/user/metrics/ || continue
	ssh $server "cd /home/user/metrics && ./server-metrics.sh && tar -czf metric-results.tar.gz apps main latestScan.txt" || continue
	scp $server:/home/user/metrics/metric-results.tar.gz build/servers/$server-metric-results.tar.gz || continue
done
