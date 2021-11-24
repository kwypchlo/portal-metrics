#!/bin/bash

# Get the list of servers
servers=$(ls -1 data/server-keys)

# Get the keys for each server.
for server in $servers
do
	# Declare we are working on the server, and scan the host's key.
	keyKnown=$(cat combined-metrics/servers-keys/$server)
	if [[ "$keyKnown" != "true" ]];
	then
		ssh-keyscan $server >> ~/.ssh/known_hosts
		echo "true" > combined-metrics/servers-keys/$server
	fi
done

# Run the updater on each server.
for server in $servers
do
	# Transfer the scripts to the server and run them.
	echo "working on $server"
	ssh $server "mkdir -p /home/user/metrics" || continue
	scp ../server-metrics/{metrics.sh,stats,nginx-data-splitter} $server:/home/user/metrics/ || continue
	ssh $server "/home/user/metrics/metrics.sh" || continue
	ssh $server "cd /home/user/metrics && tar -czf metric-results.tar.gz apps main latestScan.txt" || continue
	scp $server:/home/user/metrics/metric-results.tar.gz combined-metrics/servers//$server-metric-results.tar.gz || continue
done
