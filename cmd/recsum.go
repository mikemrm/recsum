package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"crypto"
	_ "crypto/md5"
	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"

	"github.com/mikemrm/recsum"
)

var (
	logger  = log.New(os.Stderr, "", 0)
	output  = flag.String("o", "-", "Output file path")
	hash    = flag.String("h", "sha256", "Hash algorithm to use [md5, sha1, sha256, sha512]")
	workers = flag.Int("w", 3, "Simultaneous workers")
	verbose = flag.Bool("v", false, "Verbose")
)

func determineHash(h string) (crypto.Hash, error) {
	switch h {
	case "md5":
		return crypto.MD5, nil
	case "sha1":
		return crypto.SHA1, nil
	case "sha256":
		return crypto.SHA256, nil
	case "sha512":
		return crypto.SHA512, nil
	}
	return 0, fmt.Errorf("Unknown hash '%s'", h)
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] FILE...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "version: %s\n\n", recsum.Version)
		fmt.Fprintf(os.Stderr, "recsum is a tool for recursively generating hash sums\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}
	var err error

	h, err := determineHash(*hash)
	if err != nil {
		logger.Printf("%s", err)
		flag.Usage()
		os.Exit(1)
	}

	var file *os.File
	if *output != "-" {
		file, err = os.Create(*output)
		if err != nil {
			panic(err)
		}
		defer file.Close()
	}

	outch := make(chan *recsum.HashResult, 1)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for result := range outch {
			if result.Error == nil {
				if *output == "-" {
					fmt.Printf("%s  %s\n", result.Hash, result.Path)
				} else {
					fmt.Fprintf(file, "%s  %s\n", result.Hash, result.Path)
				}
			} else {
				logger.Printf("%s failed with error '%s'\n", result.Path, result.Error)
			}
			if *verbose {
				logger.Printf("%s completed in %s\n", result.Path, result.End.Sub(result.Start))
			}
		}
	}()

	start := time.Now().UTC()
	paths := flag.Args()
	for _, path := range paths {
		recursor, err := recsum.New(path, h, outch, *workers)
		if err != nil {
			panic(err)
		}
		if err = recursor.Walk(); err != nil {
			panic(err)
		}
	}
	close(outch)
	wg.Wait()
	end := time.Now().UTC()

	if *output != "-" {
		logger.Printf("Output written to '%s'\n", *output)
	}
	if *verbose || *output != "-" {
		logger.Printf("Completed in %s\n", end.Sub(start))
	}
}
