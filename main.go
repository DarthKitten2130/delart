package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
)

// main configures and runs the artifact scanning command.
func main() {
	cmd := &cli.Command{
		Name:  "scan",
		Usage: "scans a directory for artifacts; build and run with sudo to scan protected directories",
		Action: func(context.Context, *cli.Command) error {
			if os.Geteuid() != 0 {
				return fmt.Errorf("permission denied: build the scanner, then run it with sudo: `go build -o delart . && sudo ./delart scan`")
			}

			version, err := readVersionFile()
			if err != nil {
				return err
			}

			artifacts, err := readArtifacts()
			if err != nil {
				return err
			}

			fmt.Printf("Reading artifact definitions version %s\n\n", version)
			results := make([]scanResult, 0)

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
					matched := false
					if d.IsDir() && artifact.Type == "directory" {
						for _, identifier := range artifact.Identifiers {
							_, statErr := os.Stat(filepath.Join(path, identifier))
							if statErr == nil {
								matched = true
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
								matched = true
								break
							}
						}
					}

					if matched {
						info, infoErr := d.Info()
						if infoErr != nil {
							return fmt.Errorf("read metadata for %s: %w", path, infoErr)
						}
						size, sizeErr := artifactSize(path, artifact.Type, info)
						if sizeErr != nil {
							return sizeErr
						}
						results = append(results, scanResult{
							language:     artifact.Language,
							artifact:     artifact.Name,
							artifactType: artifact.Type,
							createdAt:    info.ModTime(),
							size:         size,
							path:         path,
						})
					}
				}

				return nil
			})

			if err != nil {
				return err
			}
			if len(results) == 0 {
				return fmt.Errorf("no configured artifacts found")
			}

			printScanResults(results)
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

type scanResult struct {
	language     string
	artifact     string
	artifactType string
	createdAt    time.Time
	size         int64
	path         string
}

func artifactSize(path, artifactType string, info fs.FileInfo) (int64, error) {
	if artifactType == "file" {
		return info.Size(), nil
	}

	var size int64
	err := filepath.WalkDir(path, func(currentPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("read size for %s: %w", currentPath, err)
		}
		if entry.IsDir() {
			return nil
		}

		entryInfo, infoErr := entry.Info()
		if infoErr != nil {
			return fmt.Errorf("read metadata for %s: %w", currentPath, infoErr)
		}
		if entryInfo.Mode().IsRegular() {
			size += entryInfo.Size()
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	return size, nil
}

// printScanResults renders detected artifacts as a bordered table.
func printScanResults(results []scanResult) {
	rows := make([][]string, 0, len(results)+1)
	rows = append(rows, []string{"LANGUAGE", "ARTIFACT TYPE", "DATE OF CREATION", "SIZE", "PATH"})
	for _, result := range results {
		rows = append(rows, []string{
			result.language,
			fmt.Sprintf("%s (%s)", result.artifactType, result.artifact),
			result.createdAt.Format(time.RFC3339),
			formatSize(result.size),
			result.path,
		})
	}

	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for column, value := range row {
			if len(value) > widths[column] {
				widths[column] = len(value)
			}
		}

	}

	separator := make([]string, len(widths))
	for column, width := range widths {
		separator[column] = strings.Repeat("-", width+2)
	}
	border := "+" + strings.Join(separator, "+") + "+"

	fmt.Println(border)
	for rowIndex, row := range rows {
		fmt.Print("|")
		for column, value := range row {
			fmt.Printf(" %-*s |", widths[column], value)
		}
		fmt.Println()
		if rowIndex == 0 || rowIndex == len(rows)-1 {
			fmt.Println(border)
		}
	}
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	value := float64(size)
	units := []string{"KB", "MB", "GB", "TB"}
	for _, currentUnit := range units {
		value /= unit
		if value < unit || currentUnit == units[len(units)-1] {
			return fmt.Sprintf("%.1f %s", value, currentUnit)
		}
	}

	return fmt.Sprintf("%d B", size)
}
