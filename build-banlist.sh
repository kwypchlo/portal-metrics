#!/bin/bash

# Get the list of servers
servers=$(ls -1 build/server-keys)

for server in $servers
do
	# Transfer the necessary scrips and binaries to the server and run them.
	ssh $server "/home/user/metrics/banfinder /home/user/metrics > /home/user/metrics/ipbans.txt"
	scp $server:/home/user/metrics/ipbans.txt build/ip-bans/$server-ipbans.txt
done

# Merge all of the ipbans files together.
files=$(ls -1 build/ip-bans)
rm -f build/ip-bans.txt
for file in $files
do
	cat build/ip-bans/$file >> build/ip-bans.txt
done
cat build/ip-bans.txt | sort | uniq > build/ip-bans-unique.txt
mv build/ip-bans-unique.txt build/ip-bans.txt
