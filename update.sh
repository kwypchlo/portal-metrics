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

# Run the filter on every server, which should resume from the last scanned log
# line and continue building out the log indexes.
for server in $servers
do
	# Transfer the necessary scrips and binaries to the server and run them.
	echo "updating $server"
	ssh $server "mkdir -p /home/user/metrics/days" || continue
	scp build/filter $server:/home/user/metrics/ || continue
	ssh $server "/home/user/metrics/filter /home/user/skynet-webportal/docker/data/nginx/logs /home/user/metrics"
done
