#!/bin/bash

# TODO: Need a separate 'processedUntil' file for ips

# Prepare the build directory. We have a 'graphs' folder where all of the graphs
# will be placed, and a 'servers-dates-processed' folder which tracks the dates we've
# already processed for each server.
mkdir -p build/servers-dates-processed
mkdir -p build/joined-data/main
mkdir -p build/joined-data/apps
mkdir -p build/graphs/main
mkdir -p build/graphs/apps
cp graph-code/graph.html build/graphs/main/
cp graph-code/plotly-2.6.3.min.js build/graphs/
cp graph-code/plotly-2.6.3.min.js build/graphs/apps

# For every server in the servers folder, perform the merging operation. This
# will extract the tarball of indexes from the server into a tmp folder and then
# process those indexes by merging the numbers with the main index.
servers=$(ls -1 build/server-keys)
for server in $servers
do
	# Extract the latest data from the server to a tmp directory.
	echo "adding metrics from $server"
	rm -rf build/tmp
	mkdir -p build/tmp
	tar -xzf build/servers/$server-metric-results.tar.gz -C build/tmp || continue
	processedUntil=$(cat build/servers-dates-processed/$server-main.txt) || processedUntil=0
	serverUntil=$(cat build/tmp/latestScan.txt)

	# Skip this server if we've already processed everything it has.
	if [[ "$processedUntil" == "$serverUntil" ]];
	then
		echo "local is up to date with $server"
		continue
	fi

	# Merge the metrics currently in the tmp dir into the main set of metrics.
	# The joiner binary works one app at a time, so we will need to loop over
	# all of the apps. We'll need to copy in the graph.html file for each app as
	# well, since that can't be done outside of the loop.
	./build/joiner $processedUntil build/tmp/main build/joined-data/main
	apps=$(ls -1 build/tmp/apps)
	for app in $apps
	do
		mkdir -p build/joined-data/apps/$app
		mkdir -p build/graphs/apps/$app
		./build/joiner $processedUntil build/tmp/apps/$app build/joined-data/apps/$app
		cp graph-code/graph.html build/graphs/apps/$app/
	done

	# Update the cache so that we don't look at the same data again.
	echo $serverUntil > build/servers-dates-processed/$server-main.txt
done

# Get the sortings for all the apps.
apps=$(ls -1 build/joined-data/apps)
skapps=$(ls -1 build/joined-data/apps | grep ".siasky.net")
date=$(date +'%Y.%m.%d')
sortings="build/joined-data/sortings"
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
			power=$(./build/power $date build/joined-data/apps/$app $metric $decay)
			echo "$power $app" >> $sortings/apps/raw/$metric-$decay.txt
		done
		cat $sortings/apps/raw/$metric-$decay.txt | sort -n -r -k 1 > $sortings/apps/$metric-$decay.txt

		for app in $skapps
		do
			power=$(./build/power $date build/joined-data/apps/$app $metric $decay)
			echo "$power $app" >> $sortings/skapps/raw/$metric-$decay.txt
		done
		cat $sortings/skapps/raw/$metric-$decay.txt | sort -n -r -k 1 > $sortings/skapps/$metric-$decay.txt
	done
done

# Begin making the index page of the metrics app.
echo "<html><body>" > build/graphs/index.html
echo "<a href=main/graph.html>main</a><br>" >> build/graphs/index.html
echo "<br>" >> build/graphs/index.html

# Prepare for the pruned page.
rm -rf build/graphs-pruned
mkdir -p build/graphs-pruned

# Create a page for each skapp sorting.
for decay in 1 7 30 90 0
do
	for metric in "uploads" "downloads"
	do
		# Make the skapps link for this type
		echo "<a href=skapps-$metric-$decay.html>skapps-$metric-$decay</a><br>" >> build/graphs/index.html

		# Create the file itself.
		sortedApps=$(cat $sortings/skapps/$metric-$decay.txt | cut -d' ' -f2- | head -25)
		echo "<html><body>" > build/graphs/skapps-$metric-$decay.html
		for app in $sortedApps
		do
			echo $app >> build/graphs-pruned/applist.txt
			echo "<a href=apps/$app/graph.html>app/$app</a><br>" >> build/graphs/skapps-$metric-$decay.html
		done
		echo "</body></html>" >> build/graphs/skapps-$metric-$decay.html
	done
	echo "<br>" >> build/graphs/index.html
done

# Create a page for each app sorting.
for decay in 1 7 30 90 0
do
	for metric in "uploads" "downloads"
	do
		# Link to the file for this sorting from the index page.
		echo "<a href=apps-$metric-$decay.html>apps-$metric-$decay</a><br>" >> build/graphs/index.html

		# Create the file itself.
		sortedApps=$(cat $sortings/apps/$metric-$decay.txt | cut -d' ' -f2- | head -100)
		echo "<html><body>" > build/graphs/apps-$metric-$decay.html
		for app in $sortedApps
		do
			echo $app >> build/graphs-pruned/applist.txt
			echo "<a href=apps/$app/graph.html>$app</a><br>" >> build/graphs/apps-$metric-$decay.html
		done
		echo "</body></html>" >> build/graphs/apps-$metric-$decay.html
	done
	echo "<br>" >> build/graphs/index.html
done

# Create a pruned applist. We have to do a weird dance with the filename because
# if we '>' to the file we start reading from we end up deleting the file before
# we've read it all.
cat build/graphs-pruned/applist.txt | sort | uniq > build/graphs-pruned/appList.txt
rm build/graphs-pruned/applist.txt
applist=$(cat build/graphs-pruned/appList.txt)

# TODO: REMOVE, this is just for debugging
exit 0

# Go back through the servers to build the ipData for the top ranking apps. We
# only build the ipData for the top ranked apps because it is computationally
# very expensive, and doesn't currently leverage indexing to avoid doing repeat
# work.
servers=$(ls -1 build/server-keys)
for server in $servers
do
	# Extract the latest data from the server to the working directory.
	echo getting ip-data from $server
	rm -rf build/tmp
	mkdir -p build/tmp
	tar -xzf build/servers/$server-metric-results.tar.gz -C build/tmp || continue

	# Grap the ip-data from this server for the chosen set of apps. We need to
	# copy in a special graph.html which knows to look for the ip data.
	./joiner $processedUntil main ips
	cp graph-code/graph-ips.html build/graphs/main/graph.html
	for app in $applist
	do
		./joiner $processedUntil apps/$app ips
		cp graph-code/graph-ips.html build/graphs/apps/$app/graph.html
	done
done

# Copy the high ranking apps into the list of pruned graphs.
cp -r build/graphs/main build/graphs-pruned/
cp build/graphs/*.html build/graphs-pruned/
for app in $applist
do
	cp -r build/graphs/apps/$app build/graphs-pruned/apps/
done

# Copy over the plotly libs.
cp graph-code/plotly-2.6.3.min.js build/graphs-pruned/
cp graph-code/plotly-2.6.3.min.js build/graphs-pruned/apps
