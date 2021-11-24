#!/bin/bash

mkdir -p combined-metrics/server-dates

# For every server in the servers folder, perform the merging operation. New
# data can be fetched using update.sh.
servers=$(ls -1 combined-metrics/server-keys)
for server in $servers
do
	# Extract the latest data from the server to the working directory.
	echo merging $server
	rm -rf combined-metrics/tmp
	mkdir -p combined-metrics/tmp
	tar -xzf combined-metrics/servers/$server-metric-results.tar.gz -C combined-metrics/tmp | continue
	processedUntil=$(cat combined-metrics/server-dates/$server) || processedUntil=0
	serverUntil=$(cat combined-metrics/tmp/latestScan.txt)

	# Skip this server if we've already processed everything it has.
	if [[ "$processedUntil" == "$serverUntil" ]];
	then
		echo "local is up to date with $server"
		continue
	fi

	# Run the extractor on the main folder, then run the extractor on all the
	# apps folders. We keep plotly.min.js in a level above the main folder and
	# apps folder so that we don't have to copy the whole 3 MB file into each
	# app folder.
	mkdir -p combined-metrics/graphs/main
	mkdir -p combined-metrics/graphs/apps
	cp graph-code/graph.html combined-metrics/graphs/main/
	cp graph-code/plotly-2.6.3.min.js combined-metrics/graphs/
	cp graph-code/plotly-2.6.3.min.js combined-metrics/graphs/apps
	./joiner $processedUntil main
	apps=$(ls -1 combined-metrics/tmp/apps)
	for app in $apps
	do
		./joiner $processedUntil apps/$app
		cp graph-code/graph.html combined-metrics/graphs/apps/$app/
	done

	# Update the cache so that we don't look at the same data again.
	echo $serverUntil > combined-metrics/server-dates/$server
done

# Get the sortings for all the apps.
apps=$(ls -1 combined-metrics/joined-data/apps)
skapps=$(ls -1 combined-metrics/joined-data/apps | grep ".siasky.net")
date=$(date +'%Y.%m.%d')
sortings="combined-metrics/joined-data/sortings"
rm -rf $sortings
mkdir -p $sortings/apps/raw
mkdir -p $sortings/skapps/raw
for decay in 1 7 30 90 0
do
	for metric in "uploads" "downloads"
	do
		echo "getting sorting for $metric-$decay"
		for app in $apps
		do
			power=$(./power $date apps/$app $metric $decay)
			echo "$power $app" >> $sortings/apps/raw/$metric-$decay.txt
		done
		cat $sortings/apps/raw/$metric-$decay.txt | sort -n -r -k 1 > $sortings/apps/$metric-$decay.txt

		for app in $skapps
		do
			power=$(./power $date apps/$app $metric $decay)
			echo "$power $app" >> $sortings/skapps/raw/$metric-$decay.txt
		done
		cat $sortings/skapps/raw/$metric-$decay.txt | sort -n -r -k 1 > $sortings/skapps/$metric-$decay.txt
	done
done

# Begin making the index page of the app.
echo "<html><body>" > combined-metrics/graphs/index.html
echo "<a href=main/graph.html>main</a><br>" >> combined-metrics/graphs/index.html
echo "<br>" >> combined-metrics/graphs/index.html

# Prepare for the pruned page.
rm -rf combined-metrics/graphs-pruned
mkdir -p combined-metrics/graphs-pruned

# Create a page for each skapp sorting.
for decay in 1 7 30 90 0
do
	for metric in "uploads" "downloads"
	do
		# Make the skapps link for this type
		echo "<a href=skapps-$metric-$decay.html>skapps-$metric-$decay</a><br>" >> combined-metrics/graphs/index.html

		# Create the file itself.
		sortedApps=$(cat $sortings/skapps/$metric-$decay.txt | cut -d' ' -f2- | head -25)
		echo "<html><body>" > combined-metrics/graphs/skapps-$metric-$decay.html
		for app in $sortedApps
		do
			echo $app >> combined-metrics/graphs-pruned/applist.txt
			echo "<a href=apps/$app/graph.html>app/$app</a><br>" >> combined-metrics/graphs/skapps-$metric-$decay.html
		done
		echo "</body></html>" >> combined-metrics/graphs/skapps-$metric-$decay.html
	done
	echo "<br>" >> combined-metrics/graphs/index.html
done

# Create a page for each app sorting.
for decay in 1 7 30 90 0
do
	for metric in "uploads" "downloads"
	do
		# Link to the file for this sorting from the index page.
		echo "<a href=apps-$metric-$decay.html>apps-$metric-$decay</a><br>" >> combined-metrics/graphs/index.html

		# Create the file itself.
		sortedApps=$(cat $sortings/apps/$metric-$decay.txt | cut -d' ' -f2- | head -100)
		echo "<html><body>" > combined-metrics/graphs/apps-$metric-$decay.html
		for app in $sortedApps
		do
			echo $app >> combined-metrics/graphs-pruned/applist.txt
			echo "<a href=apps/$app/graph.html>$app</a><br>" >> combined-metrics/graphs/apps-$metric-$decay.html
		done
		echo "</body></html>" >> combined-metrics/graphs/apps-$metric-$decay.html
	done
	echo "<br>" >> combined-metrics/graphs/index.html
done

# Create a pruned applist. We have to do a weird dance with the filename because
# if we '>' to the file we start reading from we end up deleting the file before
# we've read it all.
cat combined-metrics/graphs-pruned/applist.txt | sort | uniq > combined-metrics/graphs-pruned/appList.txt
rm combined-metrics/graphs-pruned/applist.txt
applist=$(cat combined-metrics/graphs-pruned/appList.txt)

# Go back through the servers to build the ipData for the top ranking apps. We
# only build the ipData for the top ranked apps because it is computationally
# very expensive, and doesn't currently leverage indexing to avoid doing repeat
# work.
servers=$(ls -1 combined-metrics/server-keys)
for server in $servers
do
	# Extract the latest data from the server to the working directory.
	echo getting ip-data from $server
	rm -rf combined-metrics/tmp
	mkdir -p combined-metrics/tmp
	tar -xzf combined-metrics/servers/$server-metric-results.tar.gz -C combined-metrics/tmp | continue

	# Grap the ip-data from this server for the chosen set of apps. We need to
	# copy in a special graph.html which knows to look for the ip data.
	./joiner $processedUntil main ips
	cp graph-code/graph-ips.html combined-metrics/graphs/main/graph.html
	for app in $applist
	do
		./joiner $processedUntil apps/$app ips
		cp graph-code/graph-ips.html combined-metrics/graphs/apps/$app/graph.html
	done
done

# Copy the high ranking apps into the list of pruned graphs.
cp -r combined-metrics/graphs/main combined-metrics/graphs-pruned/
cp combined-metrics/graphs/*.html combined-metrics/graphs-pruned/
for app in $applist
do
	cp -r combined-metrics/graphs/apps/$app combined-metrics/graphs-pruned/apps/
done

# Copy over the plotly libs.
cp graph-code/plotly-2.6.3.min.js combined-metrics/graphs-pruned/
cp graph-code/plotly-2.6.3.min.js combined-metrics/graphs-pruned/apps
