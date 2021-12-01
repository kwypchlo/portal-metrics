package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/NebulousLabs/errors"
)

// testOnlyAccessLog checks that the gzReader works when there are no archive files.
func testOnlyAccessLog(testDir string) error {
	// The very first test is what happens when there's just an access.log that
	// is ready to be read.
	accessLogFilename := filepath.Join(testDir, "access.log")
	accessLogFile, err := os.OpenFile(accessLogFilename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	accessLogData := make([]byte, 1e6)
	for i := 0; i < len(accessLogData); i++ {
		accessLogData[i] = byte(i)
	}
	_, err = accessLogFile.Write(accessLogData)
	if err != nil {
		return err
	}
	err = accessLogFile.Close()
	if err != nil {
		return err
	}

	// We have just an access.log, test that the gzReader can pull data out of
	// it with no issues.
	gzr, err := openGZReader(testDir)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}

	// Try some seeking.
	_, err = gzr.Seek(0,io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(500, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[500:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	return nil
}

// testOneArchive tests that everything works when there is one archive.
func testOneArchive(testDir string) error {
	// Generate the access log data.
	accessLogData := make([]byte, 1e6)
	for i := 0; i < len(accessLogData); i++ {
		accessLogData[i] = byte(i)
	}

	// Change the test environment so that there's a gzipped access.log that
	// hasn't been parsed, and a live access.log.
	accessLogFilename := filepath.Join(testDir, "access.log")
	accessLogFile, err := os.OpenFile(accessLogFilename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	gzipData := accessLogData[:300101]
	liveData := accessLogData[300101:]
	_, err = accessLogFile.Write(liveData)
	if err != nil {
		return err
	}
	gzipLogFilename := filepath.Join(testDir, "access.log-1.gz")
	gzipFile, err := os.OpenFile(gzipLogFilename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	zip := gzip.NewWriter(gzipFile)
	_, err = zip.Write(gzipData)
	if err != nil {
		return err
	}
	err = zip.Close()
	if err != nil {
		return err
	}
	err = gzipFile.Close()
	if err != nil {
		return err
	}
	// There should now be an access.log with 700kb in it, and a gzipped log
	// with 300kb in it. The reader should read from the compressed file first,
	// then the access.log, and it should return the same values as before.
	gzr, err := openGZReader(testDir)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}

	// Try some seeking.
	_, err = gzr.Seek(0,io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(500, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[500:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(300e3, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[300e3:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(300100, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[300100:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(300101, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[300101:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(300102, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[300102:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(500e3, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[500e3:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}

	return nil
}

// testTwoArchives tests that everything works when there are two archives. The
// first seek operation will seek past the first archive to test that seeking
// before initializing the archive legnths works.
func testTwoArchives(testDir string) error {
	// Generate the access log data.
	accessLogData := make([]byte, 1e6)
	for i := 0; i < len(accessLogData); i++ {
		accessLogData[i] = byte(i)
	}

	// Write the first archive.
	gzipData := accessLogData[:300101]
	liveData := accessLogData[300101:]
	gzipLogFilename := filepath.Join(testDir, "access.log-1.gz")
	gzipFile, err := os.OpenFile(gzipLogFilename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	zip := gzip.NewWriter(gzipFile)
	_, err = zip.Write(gzipData)
	if err != nil {
		return err
	}
	err = zip.Close()
	if err != nil {
		return err
	}
	err = gzipFile.Close()
	if err != nil {
		return err
	}
	// Write the second archive.
	gzipData = liveData[:200e3]
	liveData = liveData[200e3:]
	gzipLogFilename = filepath.Join(testDir, "access.log-2.gz")
	gzipFile, err = os.OpenFile(gzipLogFilename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	zip = gzip.NewWriter(gzipFile)
	_, err = zip.Write(gzipData)
	if err != nil {
		return err
	}
	err = zip.Close()
	if err != nil {
		return err
	}
	err = gzipFile.Close()
	if err != nil {
		return err
	}
	// Write the live data.
	accessLogFilename := filepath.Join(testDir, "access.log")
	accessLogFile, err := os.OpenFile(accessLogFilename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	_, err = accessLogFile.Write(liveData)
	if err != nil {
		return err
	}

	// There should now be a 300kb archive, a 200kb archive, and a 500kb live
	// file.
	gzr, err := openGZReader(testDir)
	if err != nil {
		return err
	}
	// Seek before doing a read. Seek to the middle of the second file.
	_, err = gzr.Seek(400e3, io.SeekStart)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[400e3:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}

	// Try some seeking.
	_, err = gzr.Seek(0,io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(500, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[500:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(300e3, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[300e3:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(300100, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[300100:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(300101, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[300101:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(300102, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[300102:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(500100, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[500100:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(500101, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[500101:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(500102, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[500102:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	_, err = gzr.Seek(500103, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData[500103:]) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}
	// One last try, seek to the very front.
	_, err = gzr.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	data, err = ioutil.ReadAll(gzr)
	if err != nil {
		return err
	}
	if !bytes.Equal(data, accessLogData) {
		return errors.New("gzReader for basic access.log example is not returning the correct data")
	}

	return nil
}

// prepareTestDir will take a parent and a child as input, returning the joined
// path. The child dir will be created and empty.
func prepareTestDir(parent string, child string) (string, error) {
	childPath := filepath.Join(parent, child)
	err := os.RemoveAll(childPath)
	if err != nil {
		return "", errors.AddContext(err, "unable to remove child test dir")
	}
	err = os.MkdirAll(childPath, 0755)
	if err != nil {
		return "", errors.AddContext(err, "unable to create child test dir")
	}
	return childPath, nil
}

// TestGZReader runs some basic tests on the gzReader to make sure that
// everything is in working order.
func TestGZReader(t *testing.T) {
	// Get the TempDir so we can build some test directories.
	tmpDir := os.TempDir()
	testDir := filepath.Join(tmpDir, "TestGZReader")

	// Run the first test.
	testOnlyAccessLogPath, err := prepareTestDir(testDir, "testOnlyAccessLog")
	if err != nil {
		t.Fatal(err)
	}
	err = testOnlyAccessLog(testOnlyAccessLogPath)
	if err != nil {
		t.Error(err)
	}

	// Run the second test.
	testOneArchivePath, err := prepareTestDir(testDir, "testOneArchive")
	if err != nil {
		t.Fatal(err)
	}
	err = testOneArchive(testOneArchivePath)
	if err != nil {
		t.Error(err)
	}

	// Run the third test.
	testTwoArchivesPath, err := prepareTestDir(testDir, "testTwoArchives")
	if err != nil {
		t.Fatal(err)
	}
	err = testTwoArchives(testTwoArchivesPath)
	if err != nil {
		t.Error(err)
	}
}
