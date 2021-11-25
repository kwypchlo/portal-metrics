package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/NebulousLabs/errors"
)

// stats represents the three major fields we'll be tracking on a per-app basis.
type stats struct {
	downloads uint64
	uploads uint64
	ips map[string]struct{}
}

// splitLines does a strings.Split(string(b), " "), but without allocating new
// memory.
func splitLines(b []byte) [][]byte {
	var ret [][]byte
	start := 0
	for i := 0; i < len(b); i++ {
		if b[i] == '\n' {
			ret = append(ret, b[start:i])
			start = i+1
		}
	}
	return ret
}

// writeStats will write the stats from the provided stats object to the
// provided path.
func writeStats(s stats, path string) error {
	// Make sure the required dirs all exist.
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return errors.AddContext(err, "unable to create app dir")
	}

	// Write the download data.
	downloadsPath := filepath.Join(path, "downloads.txt")
	downloadsFile, err := os.OpenFile(downloadsPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return errors.AddContext(err, "unable to open downloads file")
	}
	_, err = fmt.Fprintf(downloadsFile, "%s %v\n", filepath.Base(os.Args[1]), s.downloads)
	if err != nil {
		return errors.AddContext(err, "unable to write to downloads file")
	}
	err = downloadsFile.Close()
	if err != nil {
		return errors.AddContext(err, "unable to close downloads file")
	}

	// Write the upload data.
	uploadsPath := filepath.Join(path, "uploads.txt")
	uploadsFile, err := os.OpenFile(uploadsPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return errors.AddContext(err, "unable to open uploads file")
	}
	_, err = fmt.Fprintf(uploadsFile, "%s %v\n", filepath.Base(os.Args[1]), s.uploads)
	if err != nil {
		return errors.AddContext(err, "unable to write to uploads file")
	}
	err = uploadsFile.Close()
	if err != nil {
		return errors.AddContext(err, "unable to close uploads file")
	}

	// Write the ip data.
	ipsPath := filepath.Join(path, "ips.txt")
	ipsFile, err := os.OpenFile(ipsPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return errors.AddContext(err, "unable to open ips file")
	}
	// Build the ip data section. The format is the date [10 bytes], followed by
	// a uint64 specifying the number of ip addresses, followed by the ip
	// addresses.
	ipData := make([]byte, 10+8+4(len(s.ips)))
	copy(ipData, []byte(os.Args[1]))
	binary.LittleEndian.PutUint64(ipData[10:], uint64(len(s.ips)))
	i := 0
	for ip, _ := range s.ips {
		nip := net.ParseIP(ip)
		nip4 := nip.To4()
		ip32 := binary.LittleEndian.Uint32(nip4)
		binary.LittleEndian.PutUint64(ipData[10+8+4*i:], ip32)
		i++
	}
	_, err = ipsFile.Write(ipData)
	if err != nil {
		return errors.AddContext(err, "unable to write to ipsFile")
	}
	err = ipsFile.Close()
	if err != nil {
		return errors.AddContext(err, "unable to close ips file")
	}
	return nil
}

func main() {
	// Check that the right number of args are in use.
	if len(os.Args) != 2 {
		fmt.Println("usage: ./stats [dayfile]")
		return
	}

	// We're going to be going through a log one line at a time. Each line we'll
	// dissect into some key metrics. All metrics go into the main app, and then
	// each metric also goes into the specific app marked in the line.
	var mainStats stats
	mainStats.ips = make(map[string]struct{})
	appStats := make(map[string]stats)

	// Open the day file.
	dayFile, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println("unable to open dayfile:", err)
		return
	}
	// Read the whole dayfile.
	dayfileData, err := ioutil.ReadAll(dayFile)
	if err != nil {
		fmt.Println("unable to read full dayfile into memory:", err)
		return
	}
	// Parse the dayfile into lines.
	lines := splitLines(dayfileData)

	// Go through the lines one at a time and grab all the stats.
	for i := 0; i < len(lines); i++ {
		line := strings.Split(string(lines[i]), " ")
		ip := line[0]
		method := line[1]
		app := line[2]

		download := method == "GET"
		upload := method == "POST"

		if download {
			mainStats.downloads++
		}
		if upload {
			mainStats.uploads++
		}
		mainStats.ips[ip] = struct{}{}

		// Don't do custom apps for empty names.
		if app == "" || app == "-" {
			continue
		}

		// Ensure that the stats for the app exist.
		_, exists := appStats[app]
		if !exists {
			var st stats
			st.ips = make(map[string]struct{})
			appStats[app] = st
		}

		// Fill out the stats for the app.
		if download {
			st := appStats[app]
			st.downloads++
			appStats[app] = st
		}
		if upload {
			st := appStats[app]
			st.uploads++
			appStats[app] = st
		}
		appStats[app].ips[ip] = struct{}{}
	}

	// Write the main stats.
	err = writeStats(mainStats, "main")
	if err != nil {
		fmt.Println("couldn't write main stats:", err)
		return
	}

	// Write all the app stats.
	for app, stats := range appStats {
		appPath := filepath.Join("apps", app)
		err = writeStats(stats, appPath)
		if err != nil {
			fmt.Println("couldn't write app stats:", err)
			return
		}
	}
}
