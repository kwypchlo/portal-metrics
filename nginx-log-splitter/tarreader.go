package main

// gzReader is a special reader for the log splitter that will scan the
// directory for logs that have been compressed into a .gz files. It will load
// them in lexographic order, and it will receive seek commands to navigate to
// the correct spot.
type gzReader struct {
}
