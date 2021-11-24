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

// splitLines does a strings.Split(string(b), " "), but without allocating new
// memory.
func splitLines(b []byte) [][]byte {
	ret := make([][]byte, 0, 100e3)

	// We can check long subslices using bytes.Contains to improve performance.
	// Without that, this function takes surprisingly a long time to execute. A
	// length of 20 was chosen by trial and error.
	subsliceSize := 20
	start := 0
	for i := 0; i+subsliceSize < len(b); i += subsliceSize {
		if !bytes.Contains(b[i:i+subsliceSize], []byte{'\n'}) {
			continue
		} else {
			for j := i; j < i+subsliceSize; j++ {
				if b[j] == '\n' {
					ret = append(ret, b[start:j])
					start = j+1
				}
			}
		}
	}
	return ret
}

// getFields does a strings.Split with a space, but it re-merges fields that
// are encased by quotes.
func getFields(line []byte) [][]byte {
	finalFields := make([][]byte, 0, 30)

	// Advance one character at a time, deciding based on quote status whether
	// to split at a space or not.
	start := 0
	quoteOpen := false
	for i := 0; i < len(line); i++ {
		if !quoteOpen {
			if line[i] == '"' {
				quoteOpen = true
			}
			if line[i] == ' ' && line[i+1] != ':' && line[i-1] != ':' {
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

// getCacheResult cleans up the cache result field.
func getCacheResult(field string) string {
	// Trim the quotes
	return field[1 : len(field)-1]
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

// getEndpoint will return the endpoint field, eliminating much of the extra
// data.
func getEndpoint(field string) string {
	// Trim the first quote.
	field = field[1:]

	// Leave the method and following space.
	endpointStart := 0
	for endpointStart = 0; endpointStart < len(field); endpointStart++ {
		if field[endpointStart] == '/' {
			break
		}
	}
	// Kill the endpoint at either a space or a question mark, as this marks the
	// end of the endpoint.
	var endpointEnd int
	for endpointEnd = endpointStart; endpointEnd < len(field); endpointEnd++ {
		if field[endpointEnd] == ' ' || field[endpointEnd] == '?' {
			break
		}
	}

	return field[:endpointEnd]
}

func main() {
	// Look for a file that says how much of the log has already been processed,
	// we will resume from there.
	bytesProcessedFile, err := os.OpenFile("bytesProcessed.txt", os.O_RDWR, 0644)
	if err != nil && !os.IsNotExist(err) {
		fmt.Println("unable to open bytesProcessed.txt:", err)
		return
	}

	// Read the file. If it doesn't exist, set the processed bytes to zero.
	bytesProcessed := 0
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
		// Close the bytesProcessedFile.
		err = bytesProcessedFile.Close()
		if err != nil {
			fmt.Println("unable to close the bytes processed file:", err)
			return
		}
	}

	// Create the directory to house the daily results.
	err = os.MkdirAll("days", 0755)
	if err != nil {
		fmt.Println("Unable to create 'days' directory:", err)
		return
	}

	// Open the access.log and seek to the provided byte offset to begin parsing
	// the file.
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

	// Read a chunk of the access.log at a time. We are assuming that the chunk
	// will have at least one full day in it, otherwise an error will be thrown.
	// Only one day is processed at a time, bufOffset is used to increase the
	// efficiency of the program when multiple days fit in memory at once.
	//
	// The loop will kick out when the end of the file has been reached.
	bufOffset := 0
	buf := make([]byte, 100e6) // buf = readBuf
	writeBuf := make([]byte, 100e6)
	for {
		n, readErr := log.Read(buf[bufOffset:])
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			fmt.Println("unable to read the file")
			return
		}
		bufOffset += n
		writeOffset := 0

		// Split the buffer into lines. Only grab the bytes that were actually
		// filled, which is indicated by bufOffset.
		lines := splitLines(buf[:bufOffset])
		// If less than 3 lines total were read, we don't want to try processing
		// them. Instead, we'll fetch more data. If these are the last 1 or 2
		// lines, they may not get processed at all, but that's okay we'll get
		// them next time.
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
