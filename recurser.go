package recsum

import (
	"crypto"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var Version = "v0.0.1"

// BuildFileHash opens the specified file and generates a hash
// using the provided hashing algorithm
func BuildFileHash(hash crypto.Hash, filename string) (string, error) {
	hasher := hash.New()
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// HashResult contains the hash results
type HashResult struct {
	Path  string
	Hash  string
	Start time.Time
	End   time.Time
	Error error
}

type recursiveHashBuilder struct {
	path     string
	buildch  chan string
	buildwg  sync.WaitGroup
	OutputCh chan *HashResult
	filter   func(string, os.FileInfo, error) bool
	workers  int
	hash     crypto.Hash
}

type RecursiveHashBuilder interface {
	//Path returns the root path being walked
	Path() string

	// Walk begins walking the path, returning an error if one is raised
	// by filepath.Walk
	Walk() error

	// SetFilter allows you to override the files / folders being filtered
	// out. The default is to filter out anything that is not a standard
	// file or symlink.
	SetFilter(func(string, os.FileInfo, error) bool)
}

// Path returns the root path set to be walked
func (b *recursiveHashBuilder) Path() string {
	return b.path
}

// buildhash generates a the hash for each file on the channel and sends the
// results to OutputCh defined by the user.
func (b *recursiveHashBuilder) buildhash(jobs <-chan string) {
	defer b.buildwg.Done()
	var time_start time.Time
	var time_end time.Time
	for path := range jobs {
		time_start = time.Now().UTC()
		hash, err := BuildFileHash(b.hash, path)
		time_end = time.Now().UTC()
		b.OutputCh <- &HashResult{path, hash, time_start, time_end, err}
	}
}

// walkfunc adds unfiltered file paths to the buildch channel
func (b *recursiveHashBuilder) walkfunc(path string, info os.FileInfo, err error) error {
	if b.filter(path, info, err) {
		b.buildch <- path
	}
	return nil
}

// Walk starts the requested amount of workers and begins walking the
// specified path. Blocking until all hashes have been built.
//
// An error may be returned from filepath.Walk
func (b *recursiveHashBuilder) Walk() error {
	defer b.buildwg.Wait()
	defer close(b.buildch)

	for j := 0; j < b.workers; j++ {
		b.buildwg.Add(1)
		go b.buildhash(b.buildch)
	}

	return filepath.Walk(b.path, b.walkfunc)
}

// SetFilter allows you to specify a function that determines whether or not to
// filter out the specified file.
func (b *recursiveHashBuilder) SetFilter(f func(path string, info os.FileInfo, err error) bool) {
	b.filter = f
}

// DefaultFilter filters out all files that are not a regular file or a SymLink.
func DefaultFilter(path string, info os.FileInfo, err error) bool {
	return info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0
}

// New creates a new RecursiveHashBuilder. The path string specifies the root
// path to begin walking.
// The hash argument sets the hashing algorithm used to generate the hash for
// the file.
// The outch argument defines the channel in which to send completed hashes
// through.
// The workers argument sets how many concurrent workers should process the list
// of file paths discovered during the walk.
func New(path string, hash crypto.Hash, outch chan *HashResult, workers int) (*recursiveHashBuilder, error) {
	if workers < 1 {
		return nil, fmt.Errorf("Must have at least 1 worker")
	}
	return &recursiveHashBuilder{
		path:     path,
		buildch:  make(chan string, 20),
		filter:   DefaultFilter,
		OutputCh: outch,
		workers:  workers,
		hash:     hash,
	}, nil
}
