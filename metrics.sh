#!/bin/bash

# We assume that the script is hosted in /home/user/metrics. Perform all actions
# from this directory. We need to 'cd' there because the script is typically
# actually running from ssh.
cd /home/user/metrics

# Run the log splitter. This will create a 'days' folder on the server that
# contains one file per day of nginx logs. Instead of containing the full nginx
# logs, it'll contain some condensed/processed logs that have a much smaller
# footprint. There is still one line per log.
#
# The splitter uses a file called 'bytesProcessed.txt' to know which parts of
# the logfile it has already processed. Running the splitter multiple times in a
# row will be efficient - it'll only process new logs that were added to the
# logfile after each run.
#
# NOTE: Any logrotate will significantly disrupt the splitter. When rotating
# logs, the 'bytesProcessed.txt' file needs to be updated to reflect that data
# has been moved out of the logfile.
./splitter /home/user/skynet-webportal/docker/data/nginx/logs || exit 1

# Once we have split the access.log into a bunch of dayfiles, we are going to
# begin processing them. We use the list of dayfiles to know what needs to be
# processed, and we use a file called 'latestScan.txt' to know which dayfiles
# have already been processed. This script can only process one dayfile at a
# time.
dayFiles=$(ls -1 days | head -n -1)
latestScan=$(cat latestScan.txt || echo 0)

# Go through the dayfiles one at a time, skipping over any days that have
# already been processed. We run the 'stats' binary against each dayfile. The
# stats binary will scroll through the dayfile and count the number of downloads
# (identifed by a GET request), uploads (identified by a POST request), and it
# will list out the unique IP addresses that accessed the file that day.
#
# The results of the scan are placed into two major folders. The first is a
# 'main' folder, which contains sitewide statistics. The second is an 'apps'
# folder, which tracks the statistics for each referrer that was making requests
# to the portal. Portals can have tens of thousands of referrers, so the 'apps'
# folder can get quite large. Inside that folder there will be three files, a
# 'downloads.txt' which lists the number of downloads each day (one line per
# day), an 'uploads.txt' which lists the number of uploads each day (one line
# per day), and 'ips.txt' which lists all of the unique IP addresses for each
# day.
#
# TODO: The 'ips.txt' file is currently organized into sections, where the first
# line of each section is the date of that section, every line after that lists
# one unique IP address, and then the section is closed off with a single line
# containing a '!'. It turns out that this format is very difficult to parse,
# and creates significant slowdowns in other parts of the logger. The plan is to
# replace this format with a binary format which does not use newlines as
# delimiters but rather uses 8 bytes each section to indicate the length of the
# section, 10 bytes to write the date (in format 'yyyy.mm.dd'), and then 4 bytes
# per IP address to list all of the IP addresses. This will both significantly
# shrink the files, and also substantially reduce the computational power
# required to process the files later.
mkdir -p main
mkdir -p apps
for dayFile in $dayFiles
do
	# If we have already scanned this day, keep going.
	if [[ "$dayFile" < "$latestScan" ]];
	then
		continue
	fi
	# Can't do '<=' so we need another statement.
	if [[ "$dayFile" == "$latestScan" ]];
	then
		continue
	fi

	# If the stats command fails, abort immediately, we do not want to corrupt
	# our files by processing the same day multiple times.
	#
	# TODO: If there is an error, we should probably nuke the main/ and app/
	# folder, and delete 'latestScan.txt' before exiting, to force a user to
	# retry rather than potentially give them corrupted data.
	./stats days/$dayFile || exit 1

	# Update the latest day so that we don't process this dayfile again. This
	# file is also used by scripts that download the processed data, so we need
	# it even if we are deleting the dayfiles as we process them.
	echo $dayFile > latestScan.txt

	# TODO: Delete the dayfiles after processing them. For now, while the
	# metrics strategy is still evolving, we are leaving the dayfiles there so
	# we don't have to keep reprocessing the full nginx logs every time we make
	# a change.
done
