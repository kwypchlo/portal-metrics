# Skynet Portal Metrics

This repo contains code which can be used to build indexes and metrics
information from a fleet of servers running skynet portal software. The process
is broken into several stages. This repo can be used to build metrics
information such as daily downloads and uploads for each skapp, and it can be
used to perform tasks like collecting a list of IP addresses responsible for
uploading malicious content such as malware or phishing applications.

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

## (shortcut - TODO)

If you just want to run everything, follow the steps above and then use the
following command:

`./update.sh && ./build-banlist.sh && ./fetch-metrics.sh && <more to come>`

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

TODO: Need to specify where one acquires the evilSkylinks.txt file. This will be
some sort of interaction with the blocker database, and it might even be
something that we can automate inside of `build-banlist.sh`.

Once that is complete, you can run `build-banlist.sh`, which will check the
upload history of each server, and identify any IP addresses which are
responsible for uploading data in the evil skylinks list.

A file 'ip-bans.txt' will be placed into the build folder which contains every
IP address which is associated with at least one evil upload. A second file
'ip-bans-24.txt' will be created which contains every /24 that had multiple IP
addresses each upload at least one evil file.

Our recommendation is that you ban every IP address and every /24 in the list.

## Running the Metrics Script
