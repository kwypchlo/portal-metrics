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

# Run the updater on every server.
for server in $servers
do
	# Transfer the necessary scrips and binaries to the server and run them.
	echo "updating $server"
	ssh $server "mkdir -p /home/user/metrics" || continue
	scp build/{banfinder,filter,evilSkylinks.txt} $server:/home/user/metrics/ || continue
	ssh $server "/home/user/metrics/filter /home/user/skynet-webportal/docker/data/nginx/logs /home/user/metrics"
done
exit 0

# Run the updater on each server.
#
# TODO: Re-enable, need to code review metrics.sh first and make sure it still
# matches the updated dir structures.
mkdir -p build/servers
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
