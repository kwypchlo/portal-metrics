package main

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/NebulousLabs/errors"
)

// gzReader is a special reader for the log splitter that will scan the
// directory for logs that have been compressed into a .gz files. It will load
// them in lexographic order, and it will receive seek commands to navigate to
// the correct spot. After it has read or seeked through all of the compressed
// logs, it will open the latest log and begin seeking through that.
//
// NOTE: Eventually this will need to be extended to handle logs that have been
// migrated to Skynet as well, we might even be able to to do the migration from
// the log parser script.
type gzReader struct {
	// dir is the directory from which the gzReader is reading.
	logDir string
	metricsDir string
	init bool

	// These variables list the archive files and the logical lengths (the
	// length once the archive is decompressed) of each archive. The list of
	// archiveLengths may be shorter than the archiveList if there are archives
	// which haven't been fully scanned yet.
	archiveList    []string
	archiveLengths []uint64

	// logicalOffset is the position of the reader within the logical data of
	// the concatenated, decompressed archives. fileIndex is the index of the
	// file (either an archive, or the live file) that contains the current
	// logicalOffset.
	logicalOffset                 int64
	fileIndex                     int
	progressThroughCurrentArchive int

	// archiveLengths is the file that contains the metadata about what logical
	// offsets each archive file starts at. currentFile is the open file handle
	// for whatever file the gzReader is currently reading from.
	archiveLengthsFile *os.File
	currentFile        *os.File
	currentReader      io.Reader
}

// openGZReader opens a gz reader that can seek through the data that is stored
// in the compressed logs as well as the data stored in the real logs.
//
// openGZReader is expecting everything to have a naming prefix of access.log.
// It expects access.log to be the first file, and then everything else should
// be sorted chronologically.
func openGZReader(logDir, metricsDir string) (*gzReader, error) {
	var gzr gzReader
	gzr.logDir = logDir
	gzr.metricsDir = metricsDir

	// Get the directory that contains the access.log and all of the compressed
	// historical logs.
	files, err := os.ReadDir(gzr.logDir)
	if err != nil {
		return nil, errors.AddContext(err, "unable to readdir nginx log dir")
	}
	// 'files' is already sorted by filename. Build an array that contains all
	// of the gzipped access logs. The first one should be the active
	// access.log, and then the rest should be sorted in chronological order
	// already.
	liveLogFound := false
	for i := 0; i < len(files); i++ {
		// Skip any files that aren't part of the access.log chain of files.
		if !strings.HasPrefix(files[i].Name(), "access.log") {
			continue
		}
		// Ensure that the first file is the actual 'access.log'.
		if !liveLogFound {
			if files[i].Name() != "access.log" {
				return nil, errors.New("directory does not seem to be sorted correctly, first access.log was not the live log")
			}
			liveLogFound = true
			continue
		}
		gzr.archiveList = append(gzr.archiveList, files[i].Name())
	}

	// Open the file that contains all of the offset and length information for
	// each file in the archive list.
	archiveLengthsFilename := filepath.Join(gzr.metricsDir, "archiveOffsets.dat")
	gzr.archiveLengthsFile, err = os.OpenFile(archiveLengthsFilename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, errors.AddContext(err, "unable to open archive offsets file")
	}
	offsetsData, err := ioutil.ReadAll(gzr.archiveLengthsFile)
	if err != nil {
		return nil, errors.AddContext(err, "unable to read archive offsets file")
	}
	for i := 0; (i*8)+7 < len(offsetsData); i++ {
		gzr.archiveLengths = append(gzr.archiveLengths, binary.LittleEndian.Uint64(offsetsData[i*8:]))
	}
	return &gzr, nil
}

// Seek will seek through the logical data of the gzReader to the provided
// offset.
func (gzr *gzReader) Seek(offset int64, whence int) (int64, error) {
	// Only SeekStart is implemented, error if any other seek type is requested.
	if whence != io.SeekStart {
		return 0, errors.New("gzReader does not support seeking except in absolute values")
	}

	// The outer loop iterates over the archive files to find the one with the
	// correct index.
	gzr.logicalOffset = 0
	gzr.fileIndex = 0
	for {
		// Check if we've run through the whole set of archives.
		if gzr.fileIndex >= len(gzr.archiveList) {
			accessLogPath := filepath.Join(gzr.logDir, "access.log")
			liveFile, err := os.Open(accessLogPath)
			if err != nil {
				return 0, errors.AddContext(err, "unable to open the live access.log")
			}
			_, err = liveFile.Seek(offset-gzr.logicalOffset, io.SeekStart)
			if err != nil {
				return 0, errors.AddContext(err, "unable to seek within the live file")
			}
			gzr.logicalOffset = offset
			gzr.currentFile = liveFile
			gzr.currentReader = liveFile
			gzr.init = true
			return offset, nil
		}

		// Determine whether we can skip to the next archive.
		if len(gzr.archiveLengths) > gzr.fileIndex && gzr.logicalOffset+int64(gzr.archiveLengths[gzr.fileIndex]) < offset {
			gzr.logicalOffset += int64(gzr.archiveLengths[gzr.fileIndex])
			gzr.fileIndex++
			continue
		}

		// We are left with two options. Either the desired offset is contained
		// within the current archive, or the current archive is unscanned and
		// we don't know whether the desired offset is contained within the
		// current archive. Either way, we need to scan through the archive
		// until we hit the next milestone.
		archivePath := filepath.Join(gzr.logDir, gzr.archiveList[gzr.fileIndex])
		archiveFile, err := os.Open(archivePath)
		if err != nil {
			return 0, errors.AddContext(err, "unable to open the archive file")
		}
		// Open the file with gzip. We need to wrap the archive with a
		// bufio.Reader so that it correctly implements ReadByte.
		bufioArchive := bufio.NewReader(archiveFile)
		gzipReader, err := gzip.NewReader(bufioArchive)
		if err != nil {
			return 0, errors.AddContext(err, "unable to open the archive as a gzip file")
		}
		// Use at most a 100 MB buffer to scan through the archive.
		bufSize := int64(100e6)
		if bufSize > offset-gzr.logicalOffset {
			bufSize = offset - gzr.logicalOffset
		}
		gzr.progressThroughCurrentArchive = 0
		buf := make([]byte, bufSize)
		// The inner loop keeps reading from the current archive until the
		// desired index is reached or until the end of the archive is reached.
		for {
			// Read the next set of data from the archive.
			readSize := bufSize
			if readSize > offset-gzr.logicalOffset {
				readSize = offset-gzr.logicalOffset
			}
			smolBuf := buf[:readSize]
			n, err := gzipReader.Read(smolBuf)
			gzr.logicalOffset += int64(n)
			gzr.progressThroughCurrentArchive += n
			// Check whether we've reached the end of the current archive. If
			// so, we need to record the length in the set of archive lengths.
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// Update the current offset, then compute the length of the
				// archive that just finished, then write that length fo the
				// archiveLengths file.
				gzr.archiveLengths = append(gzr.archiveLengths, uint64(gzr.progressThroughCurrentArchive))
				byteLength := make([]byte, 8)
				binary.LittleEndian.PutUint64(byteLength, uint64(gzr.progressThroughCurrentArchive))
				_, err := gzr.archiveLengthsFile.Write(byteLength)
				if err != nil {
					return 0, errors.AddContext(err, "could not write new length to archive lengths file")
				}

				// Update the active index and move on to the next file.
				gzr.fileIndex++
				break
			}
			if err != nil {
				return 0, errors.AddContext(err, "could not read from archive")
			}

			// Determine whether we've reached the index that we want.
			if gzr.logicalOffset == offset {
				gzr.currentFile = archiveFile
				gzr.currentReader = gzipReader
				gzr.init = true
				return offset, nil
			}
		}

		// Close out the files we opened before moving onto the next file.
		err = archiveFile.Close()
		if err != nil {
			return 0, errors.AddContext(err, "unable to close archive file")
		}
	}
}

// Read will read the next logical bytes from the gzReader.
func (gzr *gzReader) Read(p []byte) (int, error) {
	// If the gzr hasn't been initialized, we can initialize it by calling Seek.
	if !gzr.init {
		_, err := gzr.Seek(0, 0)
		if err != nil {
			return 0, errors.AddContext(err, "unable to init the archive reader")
		}
	}

	// Remember how much we've read total from all the archives.
	totalRead := 0

	// The outer loop iterates over the archive files, moving onto the next
	// archive file every time one is depleted.
	for {
		// The inner loop continually performs Read() operations until either
		// 'p' is filled or the current file is emptied.
		for {
			n, err := gzr.currentReader.Read(p)
			gzr.logicalOffset += int64(n)
			totalRead += n
			gzr.progressThroughCurrentArchive += n
			p = p[n:]
			if (err == io.EOF || err == io.ErrUnexpectedEOF) && gzr.fileIndex == len(gzr.archiveList) {
				return totalRead, io.EOF
			}
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			if err != nil {
				return totalRead, errors.AddContext(err, "unable to read from the current reader")
			}
			if len(p) == 0 {
				return totalRead, nil
			}
		}

		// If we got here it means we finished the current archive and need to
		// move onto the next one. First step is to check whether we need to
		// write the length of the current archive to the lengths file.
		if gzr.fileIndex == len(gzr.archiveLengths) {
			byteLength := make([]byte, 8)
			binary.LittleEndian.PutUint64(byteLength, uint64(gzr.progressThroughCurrentArchive))
			_, err := gzr.archiveLengthsFile.Write(byteLength)
			if err != nil {
				return totalRead, errors.AddContext(err, "unable to write to the archiveLengths file")
			}
		}
		// Close out the current file.
		err := gzr.currentFile.Close()
		if err != nil {
			return totalRead, errors.AddContext(err, "unable to close archive file")
		}
		// Increment the fileIndex
		gzr.fileIndex++

		// Check if the next file is the active access.log.
		if gzr.fileIndex >= len(gzr.archiveList) {
			accessLogPath := filepath.Join(gzr.logDir, "access.log")
			liveFile, err := os.Open(accessLogPath)
			if err != nil {
				return totalRead, errors.AddContext(err, "unable to open the live access.log")
			}
			gzr.currentFile = liveFile
			gzr.currentReader = liveFile
			continue
		}

		// Open the next archive.
		archivePath := filepath.Join(gzr.logDir, gzr.archiveList[gzr.fileIndex])
		archiveFile, err := os.Open(archivePath)
		if err != nil {
			return totalRead, errors.AddContext(err, "unable to open the archive file")
		}
		// Open the file with gzip. We need to wrap the archive with a
		// bufio.Reader so that it correctly implements ReadByte.
		bufioArchive := bufio.NewReader(archiveFile)
		gzipReader, err := gzip.NewReader(bufioArchive)
		if err != nil {
			return totalRead, errors.AddContext(err, "unable to open the archive as a gzip file")
		}
		gzr.progressThroughCurrentArchive = 0
		gzr.currentFile = archiveFile
		gzr.currentReader = gzipReader
	}
}
