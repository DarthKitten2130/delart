package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

var errFound = fmt.Errorf("file found")

func main() {
	cmd := &cli.Command{
		Name:  "scan",
		Usage: "scans a directory for artifacts",
		Action: func(context.Context, *cli.Command) error {
			var foundPath string
			var rootDir = "."               // Starting directory for the scan
			var targetFileName = "main.txt" // The file we are looking for

			err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					// Skip directories we don't have permission to read
					return nil
				}

				// Optimization 1: Skip large, irrelevant directories entirely
				if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules") {
					return filepath.SkipDir
				}

				// Check if it's the file we want
				if !d.IsDir() && d.Name() == targetFileName {
					foundPath = path
					fmt.Printf("Found file at: %s\n", foundPath)
					// Optimization 2: Return a custom error to stop the search immediately
					return errFound
				}

				return nil
			})

			if err == errFound {
				return nil
			}
			if err != nil {
				return err
			}

			return fmt.Errorf("file '%s' not found", targetFileName)
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
