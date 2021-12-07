# Skynet Portal Metrics

This repo contains code which can be used to build indexes and metrics
information from a fleet of servers running skynet portal software. The process
is broken into several stages. This repo can be used to build metrics
information such as daily downloads and uploads for each skapp, and it can be
used to perform tasks like collecting a list of IP addresses responsible for
uploading malicious content such as malware or phishing applications.

#### Eventual TODOs:

+ We should grab the server list from a portal API. We can use either an
  environment variable or some input parameter to specify the portal, and then
  we can get the server list from its API.
+ We should get the list of evil skylinks from the portal API. This is going to
  require authentication, as we need the exposed skylinks as opposed to the
  shielded ones. There might be some way to modify the nginx logs so that going
  forward it actually is sufficient to grab only the hidden skylinks, which
  would remove the need for auth

## Server Setup

To get started, run `make dependencies`. You will need go 1.16 or later for this
step to work. After that, you need to create a folder inside of the repo called
`build`, and inside of that folder you will need createcalled `server-keys`.
Inside of `server-keys`, you will need to create one file per server, where the
name of the file is the hostname of the server.

Example structure:

```
build/
	server-keys/
		eu-ger-1.siasky.net
		us-la-1.siasky.net
		us-ny-1.siasky.net
		us-ny-2.siaskynet
```

Initially, the files can be emtpy, or can contain a falsey message like "false".
The contents of the files will be set to "true" after the first time the machine
connects successfully to the server over ssh. If the data in the file is not
"true", the update script will ignore any connection warnings and add the
server's key to the users list of hosts. If the contents are "true", then the
script assumes that any ssh key errors indicate an attack, and the scripts will
not connect to that server until the issue is resolved.

## Get Results

If you just want to run everything, follow the steps above and then use the
following command:

`./update.sh && ./build-banlist.sh && ./fetch-metrics.sh && ./combine-metrics.sh`

After that there will be an index.html in build/graphs and another in
build/graphs-pruned.  Each of these can be uploaded to Skynet, which will
produce a webapp that allows you to explore metrics. There will also be files at
build/ip-bans-24.txt and build/ip-bans.txt which suggest IP addresses to ban
based on the file placed at build/evilSkylinks.txt. For more information, read
further.

## Updating Portals

Once you've populated the 'server-keys' folder, you are ready to begin
processing your nginx logs. Run `./update.sh` to begin. The update script will
go through the servers one at a time and run the 'filter' binary (found in
'cmd/nginx-log-filter'), which will break the log into a series of indexes that
will be written to '/home/user/metrics' on each server.

The files will include a 'days' folder, a file called 'archiveOffsets.dat', a
file called 'bytesProcessed.txt', and a file called 'uploadIPs.txt'. The days
folder contains a slimmed down version of the nginx log for each day. The days
folder is used to compute things like number of uploads each day, number of
downloads each day, and number of unique IPs each day.

uploadIPs.txt contains a list of Skylinks that were uploaded to the server, and
the corresponding IP addresses that they were uploaded from. This list is mainly
useful for banning IPs that are either uploading abusive content (malware,
phishing, csam, terrorism).

archiveOffset.dat and bytesProcessed.txt are two files that are used to record
the progress in the log parsing. If you call 'update.sh' multiple times in a
row, it will skip over the parts of the log that it has already processed,
ensruing that getting updated indexes is fast.

The update process supports gzipped logs. When you gzip an access.log, be sure
to follow the format 'access.log-YYYY-MM-DD.gz'. This will allow the script to
parse the gzipped logs in order.

## Running the Banscript

TODO: Need to specify where one acquires the evilSkylinks.txt file. This will be
some sort of interaction with the blocker database, and it might even be
something that we can automate inside of `build-banlist.sh`.

To use the ban script, you need to add a file 'evilSkylinks.txt' to the build
folder. The evil skylinks file should have a list of skylinks that have been
identified as problematic, one per line. Once you have done this, your build
folder might look something like:

```
build/
	evilSkylinks.txt
	server-keys/
		eu-ger-1.siasky.net
		us-la-1.siasky.net
		us-ny-1.siasky.net
		us-ny-2.siaskynet
```

Once that is complete, you can run `build-banlist.sh`, which will check the
upload history of each server, and identify any IP addresses which are
responsible for uploading data in the evil skylinks list.

A file 'ip-bans.txt' will be placed into the build folder which contains every
IP address which is associated with at least one evil upload. A second file
'ip-bans-24.txt' will be created which contains every /24 that had multiple IP
addresses each upload at least one evil file.

Our recommendation is that you ban every IP address and every /24 in the list.

## Running the Metrics Script

The only requirement for the metrics script is that the update script is run
first. The metrics script will load some utilities (the 'stats' utility and the
'server-metrics.sh' utility) onto each server and then use them to build an
index. After the index has been built, it will create a tar.gz file containing
the index, which it will download and house locally.

Similar to the update script, the metrics script will only run on new data. This
makes it relatively fast once the initial historic data has been processed.

## Running the Merge Metrics Script

The merge script works by going through the tarballs that get downloaded by the
metrics script and combining all of the stats together into one global picture
of metrics for each app. Part of building the global picture will be producing a
skapp in the build folder at build/graphs and a smaller skapp in
build/graphs-pruned which will provide access to historic metrics for all apps
on a per-app basis.

Because getting historic unique IPs is computationally expensive, only the top
apps will have IP data. Typically this will be between 300 and 500 apps total,
it's the top 100 apps of each category (most uploads over past timeframe, most
downloads over past timeframe, for each timeframe in 1 day, 7 days, 30 days, 90
days, and all time).

The app attempts to distinguish between traditional internet apps that are using
the portal as a CDN and fully decentralized skapps that are hosted fully on
Skynet. The distinction isn't perfect and is primarily based on the domain name.

## Resetting

Calling ./reset.sh will iterate over all of the servers, removing their metrics
directory. Then it will remove the local build folder, except for the list of
server keys and evil skylinks.
