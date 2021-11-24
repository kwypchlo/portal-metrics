#!/bin/bash

# Get the list of servers
servers=$(ls -1 data/server-keys)

# Get the keys for each server.
for server in $servers
do
	# Declare we are working on the server, and scan the host's key.
	keyKnown=$(cat data/server-keys/$server)
	if [[ "$keyKnown" != "true" ]];
	then
		ssh-keyscan $server >> ~/.ssh/known_hosts
		echo "true" > data/server-keys/$server
	fi
done

# Run the updater on each server.
mkdir -p data/servers
for server in $servers
do
	# Transfer the necessary scrips and binaries to the server and run them.
	echo "running metrics.sh on $server"
	ssh $server "mkdir -p /home/user/metrics" || continue
	scp server-updater/{metrics.sh,stats,splitter} $server:/home/user/metrics/ || continue
	ssh $server "/home/user/metrics/metrics.sh" || continue

	# Tar the resulting directories and download the tarballs to the local data
	# folder.
	ssh $server "cd /home/user/metrics && tar -czf metric-results.tar.gz apps main latestScan.txt" || continue
	scp $server:/home/user/metrics/metric-results.tar.gz data/servers/$server-metric-results.tar.gz || continue
done
