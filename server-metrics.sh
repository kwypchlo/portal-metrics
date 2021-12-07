#!/bin/bash

# We assume that the script is hosted in /home/user/metrics. Perform all actions
# from this directory. We need to 'cd' there because the script is typically
# actually running from ssh.
cd /home/user/metrics

# Grab the list of dayfiles created by the log filter. We omit the final file,
# because that file may not yet be complete. We will parse the final file once
# more files/logs have been added.
dayFiles=$(ls -1 days | head -n -1)

# Grab the date of the latest scan, which is contained in latestScan.txt. This
# will enable us to avoid scanning days that we have already scanned.
latestScan=$(cat latestScan.txt || echo 0)

# Go through the dayfiles one at a time, skipping over any days that have
# already been processed. We run the 'stats' binary against each dayfile. The
# stats binary will scroll through the dayfile and count the number of downloads
# (identifed by a GET request), uploads (identified by a POST request), and it
# will list out the unique IP addresses that accessed the file that day in the
# ips.dat file.
#
# The results of the scan are placed into multiple folders. A 'main' folder is
# created which contains sitewide statistics, and then an 'apps' folder is
# created which contains statistics for each app. The apps folder can get quite
# large, with tens of thousands of subfolders.
#
# Inside of each folder there will be three files, 'downloads.txt',
# 'uploads.txt', and 'ips.dat'. The downloads and upload files will contain a
# list, one line per day, of the number of uploads and downloads that occurred
# on that day. 'ips.dat' will contain a list of unique ip addresses that
# accessed the app on that day. We need to enumerate the unique IPs so that they
# can be accurately combined with other servers.
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
	# our files by processing the same day multiple times. If for some reason
	# the stats process fails, we nuke the indexes that it creates.
	./stats days/$dayFile || (rm -rf latestScan.txt main/ app/ && exit 1)

	# Update the latest day so that we don't process this dayfile again. This
	# file is also used by scripts that download the processed data, so we need
	# it even if we are deleting the dayfiles as we process them.
	echo $dayFile > latestScan.txt
done
