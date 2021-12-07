package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gitlab.com/NebulousLabs/errors"
)

// calculatePower pulls the power out of a single file.
func calculatePower(filepath string, decay float64) (uint64, error) {
	// Read the file into memory.
	file, err := os.Open(filepath)
	if err != nil {
		return 0, errors.AddContext(err, "unable to open file")
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return 0, errors.AddContext(err, "unable to read file")
	}
	lines := strings.Split(string(data), "\n")
	prevDateNum := 0
	power := 0.0
	for i := 0; i < len(lines); i++ {
		if len(lines[i]) < 11 {
			continue
		}
		year, err := strconv.Atoi(lines[i][:4])
		if err != nil {
			return 0, errors.AddContext(err, "cound not parse year")
		}
		month, err := strconv.Atoi(lines[i][5:7])
		if err != nil {
			return 0, errors.AddContext(err, "could not parse month")
		}
		day, err := strconv.Atoi(lines[i][8:10])
		if err != nil {
			return 0, errors.AddContext(err, "could not parse day")
		}
		points, err := strconv.Atoi(lines[i][11:])
		if err != nil {
			return 0, errors.AddContext(err, "could not parse point count")
		}

		// Get the current datenum.
		curDateNum := year*12*31+month*31+day // NOTE: assumes all months are 31 days, too lazy to fix.
		// Decay the value based on the previous datenum.
		if prevDateNum != 0 {
			for prevDateNum < curDateNum {
				power *= decay
				prevDateNum++
			}
		} else {
			prevDateNum = curDateNum
		}
		// Add the new power.
		power += float64(points)
	}

	// Finish off by decaying the power to the current date.
	year, err := strconv.Atoi(os.Args[1][:4])
	if err != nil {
		return 0, errors.AddContext(err, "unable to read input year")
	}
	month, err := strconv.Atoi(os.Args[1][5:7])
	if err != nil {
		return 0, errors.AddContext(err, "unable to read input month")
	}
	day, err := strconv.Atoi(os.Args[1][8:10])
	if err != nil {
		return 0, errors.AddContext(err, "unable to read input day")
	}
	curDateNum := year*12*31+month*31+day
	if prevDateNum > 0 {
		for prevDateNum < curDateNum {
			power *= decay
			prevDateNum++
		}
	}
	return uint64(power), nil
}

func main() {
	// Check the usage.
	if len(os.Args) != 5 {
		fmt.Println("Usage: ./power [current-date] [data-dir]\nExample: ./power 2021.11.21 main download 30")
		return
	}

	// Determine the decay.
	var decay float64
	if os.Args[4] == "1" {
		decay = 0.5
	} else if os.Args[4] == "7" {
		decay = 0.90572
	} else if os.Args[4] == "30" {
		decay = 0.97716
	} else if os.Args[4] == "90" {
		decay = 0.99232
	} else if os.Args[4] == "0" {
		decay = 1
	} else {
		fmt.Println("Invalid decay, options are 1, 7, 30, 90, 0")
		return
	}

	// Determine the sorting type.
	var path string
	if os.Args[3] == "downloads" {
		path = filepath.Join(os.Args[2], "downloads.txt")
	} else if os.Args[3] == "uploads" {
		path = filepath.Join(os.Args[2], "uploads.txt")
	} else {
		fmt.Println("Invalid sort type, options are 'downloads' and 'uploads'")
		return
	}

	// Open the downloads and uploads file for the provided dir.
	power, err := calculatePower(path, decay)
	if err != nil {
		fmt.Println("could not calculate power:", err)
		return
	}
	fmt.Println(power)
}
