package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "scan",
		Usage: "scans a directory for artifacts",
		Action: func(context.Context, *cli.Command) error {
			version, err := readVersionFile()
			if err != nil {
				return err
			}

			artifacts, err := readArtifacts()
			if err != nil {
				return err
			}

			fmt.Printf("Reading artifact definitions version %s\n", version)
			found := false

			err = filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					// Skip directories we don't have permission to read
					return nil
				}

				// Optimization 1: Skip large, irrelevant directories entirely
				if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules") {
					return filepath.SkipDir
				}

				for _, artifact := range artifacts {
					if d.IsDir() && artifact.Type == "directory" {
						for _, identifier := range artifact.Identifiers {
							_, statErr := os.Stat(filepath.Join(path, identifier))
							if statErr == nil {
								fmt.Printf("Found %s at: %s\n", artifact.Name, path)
								found = true
								break
							}
							if !errors.Is(statErr, os.ErrNotExist) {
								return fmt.Errorf("check identifier %q in %s: %w", identifier, path, statErr)
							}
						}
					}

					if !d.IsDir() && artifact.Type == "file" {
						for _, identifier := range artifact.Identifiers {
							if d.Name() == identifier {
								fmt.Printf("Found %s at: %s\n", artifact.Name, path)
								found = true
								break
							}
						}
					}
				}

				return nil
			})

			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("no configured artifacts found")
			}
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
