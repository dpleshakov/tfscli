//go:build ignore

// Command rm removes files and directories, so that the Makefile does not have
// to rely on a Unix shell being present. A path that does not exist is not an
// error: the point is to reach a state, not to perform a deletion.
//
// Usage:
//
//	go run tools/rm.go [-r] path...
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
)

func main() {
	recursive := flag.Bool("r", false, "remove directories and their contents recursively")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: go run tools/rm.go [-r] path...")
		os.Exit(2)
	}

	failed := false
	for _, path := range flag.Args() {
		if err := remove(path, *recursive); err != nil {
			fmt.Fprintf(os.Stderr, "rm: %v\n", err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func remove(path string, recursive bool) error {
	var err error
	if recursive {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
