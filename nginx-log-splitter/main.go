package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// getFields will return the different fields of an nginx log lines as separate
// byte slices. It pays attention to elements like colons and quotes. Notably,
// it does not currently pay attention to brackets, but this has largely not
// caused problems yet.
func getFields(line []byte) [][]byte {
	// There are typically about 24 fields in an nginx line.
	finalFields := make([][]byte, 0, 30)

	// Advance one character at a time, deciding based on quote status whether
	// to split at a space or not.
	start := 0
	quoteOpen := false
	for i := 0; i < len(line); i++ {
		if !quoteOpen {
			if line[i] == '"' {
				quoteOpen = true
			} else if line[i] == ' ' && line[i+1] != ':' && line[i-1] != ':' {
				// This conditional will not trigger if the space has a colon on
				// either side of it, because some nginx elements use colons to
				// concetenate data.
				finalFields = append(finalFields, line[start:i])
				start = i+1
			}
		} else {
			if line[i] == '"' {
				i++
				quoteOpen = false
				finalFields = append(finalFields, line[start:i])
				start = i+1
			}
		}
	}
	return finalFields
}

// getDateFromField will return the date for the presented field.
func getDateFromField(field []byte) []byte {
	// To improve performance, we started modifying things in-place. We also
	// copy the buffers around, which means we may run into memory we already
	// modified. If the first field already shows modifications, just return
	// things as-is.
	if field[0] != '[' {
		return field[:10]
	}

	// First figure out what month we need.
	var m1, m2 byte
	if field[4] == 'J' {
		if field[5] == 'a' {
			m1 = '0'
			m2 = '1'
		} else if field[6] == 'n' {
			m1 = '0'
			m2 = '6'
		} else {
			m1 = '0'
			m2 = '7'
		}
	} else if field[4] == 'F' {
		m1 = '0'
		m2 = '2'
	} else if field[4] == 'M' {
		if field[6] == 'r' {
			m1 = '0'
			m2 = '3'
		} else {
			m1 = '0'
			m2 = '5'
		}
	} else if field[4] == 'A' {
		if field[5] == 'p' {
			m1 = '0'
			m2 = '4'
		} else {
			m1 = '0'
			m2 = '8'
		}
	} else if field[4] == 'S' {
		m1 = '0'
		m2 = '9'
	} else if field[4] == 'O' {
		m1 = '1'
		m2 = '0'
	} else if field[4] == 'N' {
		m1 = '1'
		m2 = '1'
	} else {
		m1 = '1'
		m2 = '2'
	}

	// Swap the dates around correctly.
	field[0] = field[8]
	field[8] = field[1]
	field[1] = field[9]
	field[9] = field[2]
	field[2] = field[10]
	field[3] = field[11]
	field[4] = '.'
	field[5] = m1
	field[6] = m2
	field[7] = '.'
	return field[:10]
}

// getDomain cleans up the domain field.
func getDomain(field []byte, line []byte) []byte {
	if len(field) < 2 {
		fmt.Println("bad log line")
		fmt.Println(string(line))
		return []byte{'-'}
	}
	// Trim the quotes
	field = field[1 : len(field)-1]

	// Trim the protocol
	var i int
	for i = 0; i < len(field)-1; i++ {
		if field[i] == '/' && field[i+1] == '/' {
			i++
			break
		}
	}
	if len(field) >= i+1 {
		field = field[i+1:]
	}

	// Trim everything after the first '/'
	for i = 0; i < len(field); i++ {
		if field[i] == '/' {
			break
		}
	}
	field = field[:i]

	if len(field) == 0 || (len(field) == 1 && field[0] == ' ') {
		return []byte{'-'}
	}
	return field
}

// getMethod returns just the method of an endpoint.
func getMethod(field []byte) []byte {
	// Trim the first quote.
	field = field[1:]

	// Stop at the first space, that's when we see the method.
	methodEnd := 0
	for methodEnd = 0; methodEnd < len(field); methodEnd++ {
		if field[methodEnd] == ' ' {
			break
		}
	}
	return field[:methodEnd]
}

func main() {
	// Look for a file that says how much of the log has already been processed,
	// we will resume from there.
	bytesProcessed := 0
	bytesProcessedFile, err := os.OpenFile("bytesProcessed.txt", os.O_RDWR, 0644)
	if err != nil && !os.IsNotExist(err) {
		fmt.Println("unable to open bytesProcessed.txt:", err)
		return
	}
	if !os.IsNotExist(err) {
		lineCountBytes, err := ioutil.ReadAll(bytesProcessedFile)
		if err != nil {
			fmt.Println("unable to read the bytesProcessedFile:", err)
			return
		}
		bytesProcessedStr := strings.TrimSpace(string(lineCountBytes))
		bytesProcessed, err = strconv.Atoi(bytesProcessedStr)
		if err != nil {
			fmt.Println("bytesProcessed file could not be parsed:", err)
			return
		}
		fmt.Println("processing logfile starting from byte number:", bytesProcessed)
		err = bytesProcessedFile.Close()
		if err != nil {
			fmt.Println("unable to close the bytes processed file:", err)
			return
		}
	}

	// Create the directory to house the daily results. There's one file created
	// per day.
	err = os.MkdirAll("days", 0755)
	if err != nil {
		fmt.Println("Unable to create 'days' directory:", err)
		return
	}

	// Open the access.log and seek to the provided byte offset to begin parsing
	// the file.
	//
	// TODO: Change the log to being a reader that wraps around an object which
	// searches through sorted logs that have been compressed into a .tar.gz. We
	// are going to need another file which tells us how many .tar.gz files to
	// skip, and then we'll use a gzip streamer to open and read through any
	// gzipped logs that we haven't read yet. The file that tells us which
	// gzipped file we've read most recently will also tell us how many logical
	// bytes (uncompressed) were in all of the gzips up to and including that
	// log, so when we move onto the next log or the access.log itself, we know
	// what offset to start at.
	//
	// The most common scenario will be one where we read all the data in a
	// prior set of gzips, then read a fraction of the access.log. When we come
	// back, we'll see that we have new gzips to read through, but our
	// bytesProcessed field will tell us that we've already read at least some
	// of the data in the new gzips. We'll have to seek through the gzipped
	// files until we've gone through the right number of bytes, then we can
	// start reading as usual.
	log, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(os.Args[1])
		fmt.Println("unable to open access.log:", err)
		return
	}
	_, err = log.Seek(int64(bytesProcessed), 0)
	if err != nil {
		fmt.Println("unable to seed within the access.log", err)
		return
	}

	// This loop reads one chunk of the access.log at a time. We limit the
	// buffers to 100 MB each to reduce the amount of memory. We are able to
	// make the writeBuf 100 MB because we know that the data written to the
	// writeBuf will always be strictly fewer bytes than the data read from the
	// streamer into 'buf'.
	//
	// This loop processes at most one day of logs per iteration, which means
	// only a fraction of the buffer may be processed. This means that the
	// buffer shifting needs to be implemented carefully.
	readBufOffset := 0
	readBuf := make([]byte, 100e6)
	writeBuf := make([]byte, 100e6)
	for {
		n, readErr := log.Read(readBuf[bufOffset:])
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			fmt.Println("unable to read the file")
			return
		}
		readBufOffset += n
		writeOffset := 0

		// Split the buffer into lines. Only grab the bytes that were actually
		// filled, which is indicated by bufOffset.
		lines := bytes.Split(buf[:bufOffset], []byte{' '})
		// If less than 3 lines total were read, we don't want to try processing
		// them. Instead, we'll fetch more data. If these are the last 1 or 2
		// lines, they may not get processed at all, but that's okay we'll get
		// those lines later after more logs are available.
		if len(lines) < 3 {
			continue
		}

		// Get the date from the first line.
		fields := getFields(lines[0])
		date := getDateFromField(fields[3])
		fmt.Println("Processing date:", string(date))

		// Drop the final line, as the final line may only be a partial set of
		// data.
		lines = lines[:len(lines)-1]

		// Process lines until we finish with the current date, or until the
		// buffer runs out.
		bytesProcessedCurrentDay := 0
		for i := 0; i < len(lines); i++ {
			// Split each line into the characteristic fields from nginx.
			fields := getFields(lines[i])

			// Get the date for this line. If the date does not match the date
			// we started at, we need to stop processing and move onto the next
			// day file.
			lineDate := getDateFromField(fields[3])
			if !bytes.Equal(lineDate, date) {
				break
			}

			// Create the condensed log line and write it to the day file.
			// [IP ENDPOINT DOMAIN]
			ip := fields[0]
			method := getMethod(fields[5])
			domain := getDomain(fields[8], lines[i])
			copy(writeBuf[writeOffset:], ip)
			writeOffset += len(ip)
			writeBuf[writeOffset] = ' '
			writeOffset++
			copy(writeBuf[writeOffset:], method)
			writeOffset += len(method)
			writeBuf[writeOffset] = ' '
			writeOffset++
			copy(writeBuf[writeOffset:], domain)
			writeOffset += len(domain)
			writeBuf[writeOffset] = '\n'
			writeOffset++

			// Count the number of bytes processed to be accurate on the next
			// iteration. Add one for the newline that got removed by the Split
			// call.
			bytesProcessedCurrentDay += len(lines[i]) + 1
		}

		// Open a file for the first date.
		dayFilepath := filepath.Join("days", string(date))
		dayFile, err := os.OpenFile(dayFilepath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			fmt.Println("Unable to open the dayfile:", err)
			return
		}
		_, err = dayFile.Write(writeBuf[:writeOffset])
		if err != nil {
			fmt.Println("unable to write to the dayfile:", err)
			return
		}

		// We've reached the end of the day, prepare for the next day. We copy
		// the unread part of the buf to the beginning, and then set the
		// bufOffset so that the next read doesn't have to do a full read, it
		// can re-use the unread data.
		copy(buf, buf[bytesProcessedCurrentDay:])
		bufOffset -= bytesProcessedCurrentDay
		bytesProcessed += bytesProcessedCurrentDay

		// Update the bytesProcessed file to contain the new bytes processed. We
		// update this write after writing to the dayfile to minimize the chance
		// that the two fall out of sync.
		bytesProcessedFile, err = os.OpenFile("bytesProcessed.txt", os.O_RDWR|os.O_CREATE, 0644)
		if err != nil && !os.IsNotExist(err) {
			fmt.Println("CORRUPTION WARNING - DATA MAY BE CORRUPTED NOW, ESPECIALLY DAYFILE:", dayFilepath)
			fmt.Println("unable to open bytesProcessed.txt:", err)
			return
		}
		_, err = fmt.Fprintf(bytesProcessedFile, "%v\n", bytesProcessed)
		if err != nil {
			fmt.Println("CORRUPTION WARNING - DATA MAY BE CORRUPTED NOW, ESPECIALLY DAYFILE:", dayFilepath)
			fmt.Println("error writing to the bytes processed file:", err)
			return
		}
		// Close the dayfile that was opened earlier.
		err = dayFile.Close()
		if err != nil {
			fmt.Println("error closing dayfile:", err)
			return
		}
		err = bytesProcessedFile.Close()
		if err != nil {
			fmt.Println("error closing bytes processed file:", err)
			return
		}
	}

	err = log.Close()
	if err != nil {
		fmt.Println("error closing access.log:", err)
		return
	}
}
