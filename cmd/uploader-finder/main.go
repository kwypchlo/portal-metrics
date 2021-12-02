package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// banfinder is an executable which runs against an uploadIPs.txt file and an
// evilSkylinks.txt file. It'll load the evil skylinks into memory and then scan
// through the set of uploaded files to see if there are any matches.
func main() {
	if len(os.Args) != 2 {
		fmt.Println("banfinder is being used incorrectly, the one and only arg should be the metrics directory")
		fmt.Println("Example: /home/user/metrics/banfinder /home/user/metrics/")
		return
	}

	// Load the evilSkylinks into a map.
	skylinksFile, err := os.Open(filepath.Join(os.Args[1], "evilSkylinks.txt"))
	if err != nil {
		fmt.Println("Unable to open evilSkylinks.txt:", err)
		return
	}
	skylinkData, err := ioutil.ReadAll(skylinksFile)
	if err != nil {
		fmt.Println("Unable to read evilSkylinks.txt:", err)
	}
	skylinks := bytes.Split(skylinkData, []byte{'\n'})
	evilSkylinks := make(map[[46]byte]struct{})
	for _, skylink := range skylinks {
		var link [46]byte
		copy(link[:], skylink)
		evilSkylinks[link] = struct{}{}
	}

	// Begin reading through the uploadIPs file.
	uploadsFile, err := os.Open(filepath.Join(os.Args[1], "uploadIPs.txt"))
	if err != nil {
		fmt.Println("Unable to open uploadIPs.txt:", err)
		return
	}
	buf := make([]byte, 100e6)
	bufOffset := 0
	hits := 0
	for {
		n, err := uploadsFile.Read(buf[bufOffset:])
		if err == io.EOF {
			// The full file has been read.
			return
		}
		// If there's an unexpected EOF, it means some data was read but not all
		// of it. We still need to process the data, so we're just going to
		// ignore this error.
		if err == io.ErrUnexpectedEOF {
			err = nil
		}
		if err != nil {
			fmt.Println("Error while reading from uploadIPs.txt:", err)
			return
		}

		// Split the buf by the number of lines.
		lines := bytes.Split(buf[:n], []byte{'\n'})
		if len(lines) < 2 {
			// We're going to skip the last line because we don't want to check
			// whether its a partial line.
			return
		}
		// Drop the final line as it might be a partial line.
		final := len(lines)-1
		dropLen := len(lines[final])
		lines = lines[:final]
		bytesConsumed := n - dropLen

		// Go through the lines one at a time, check if any of the IPs are
		// suspect.
		for i := 0; i < len(lines); i++ {
			var skylink [46]byte
			copy(skylink[:], lines[i])
			_, exists := evilSkylinks[skylink]
			if !exists {
				continue
			}
			// This skylink is in the set of evil skylinks, record the ban.
			fmt.Println(string(lines[i][47:]))
			hits++
		}

		// Prepare buf for the next iteration.
		copy(buf, buf[bytesConsumed:])
		bufOffset = dropLen
	}
}
