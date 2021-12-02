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

## Updating Portals

Once you've populated the 'server-keys' folder, you are ready to begin
processing your nginx logs. Run `./update.sh` to begin. The update script will
go through the servers one at a time and run 'metrics.sh' within the servers.
This will split the nginx access.log files (including any archives) into a set
of index files. Those index files will be used by other scripts later.

The first time you run the script it can take a while. Subsequent runs of
'update.sh' will track previous work that was completed and only process new log
lines, which means the process will go much faster.

