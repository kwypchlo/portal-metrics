# Skynet Portal Metrics

This repo contains code which can be used to build a static application which
shows a series of metrics pulled from the nginx logs of a skynet portal. The
process is broken into several steps.

The main goal of this project was to get some low-cost metrics from Skynet.
Other solutions that we explored (mainly ELK and goaccess) either were not cheap
enough or not powerful enough to produce the results that we wanted.

## Setup

To get started, run `make dependencies`. You will need go 1.16 or later for this
step to work. After that, you need to create a folder inside of the repo called
`data`, and inside of that folder you will need createcalled `server-keys`.
Inside of `server-keys`, you will need to create one file per server, where the
name of the file is the hostname of the server.

Example structure:

```
data/
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
