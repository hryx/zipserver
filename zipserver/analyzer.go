package zipserver

import (
	"errors"
	"io"
)

// ErrSkipped is non-critical and indicates that analysis
// chose to ignore a file. The file should not be uploaded.
var ErrSkipped = errors.New("skipped file")

// Analyzer analyzes individual files in a zip archive.
// Behavior may change based on the intended workload.
type Analyzer interface {
	// Analyze should return info about the contained file.
	// It should return ErrSkipped if a file was ignored.
	Analyze(r io.Reader, key string) (AnalyzeResult, error)
}

type AnalyzeResult struct {
	Key             string // Analysis may choose a new key/filename
	Metadata        interface{}
	ContentType     string
	ContentEncoding string
}
