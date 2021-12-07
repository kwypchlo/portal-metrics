#!/bin/bash

# Clear the metrics folder on every server.
servers=$(ls -1 build/server-keys)
for server in $servers
do
	echo "cleaning $server"
	ssh $server "rm -rf /home/user/metrics"
done

# Delete the build folder, but save the server-keys and evilSkylinks
rm -rf tmp
mkdir -p tmp
mv build/evilSkylinks.txt tmp/evilSkylinks.txt
mv build/server-keys tmp/server-keys
rm -rf build
mkdir -p build
mv tmp/* build/
rm -rf tmp
