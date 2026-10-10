package main

import (
	"encoding/json"
	"fmt"
	"os"
)

const artifactsFile = "artifacts.json"

// Artifact describes one artifact variant and the identifiers used to find it.
type Artifact struct {
	Language    string   `json:"-"`
	Name        string   `json:"-"`
	Description string   `json:"description"`
	Identifiers []string `json:"identifier"`
	Type        string   `json:"type"`
	// Remove lists paths to remove; an empty list means remove the whole match.
	Remove []string `json:"rm"`
}

type artifactManifest struct {
	Version   string                         `json:"version"`
	Artifacts map[string]map[string]Artifact `json:"artifacts"`
}

// readManifest loads and validates the artifact definition manifest.
func readManifest() (artifactManifest, error) {
	var manifest artifactManifest

	data, err := os.ReadFile(artifactsFile)
	if err != nil {
		return manifest, fmt.Errorf("read %s: %w", artifactsFile, err)
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("parse %s: %w", artifactsFile, err)
	}

	if manifest.Version == "" {
		return manifest, fmt.Errorf("%s has no version", artifactsFile)
	}

	return manifest, nil
}

// readVersionFile returns the version of the artifact definition manifest.
func readVersionFile() (string, error) {
	manifest, err := readManifest()
	if err != nil {
		return "", err
	}

	return manifest.Version, nil
}

// readArtifacts returns the manifest's nested artifact definitions as scan rules.
func readArtifacts() ([]Artifact, error) {
	manifest, err := readManifest()
	if err != nil {
		return nil, err
	}

	artifacts := make([]Artifact, 0)
	for name, variants := range manifest.Artifacts {
		for variant, artifact := range variants {
			if artifact.Type != "file" && artifact.Type != "directory" {
				return nil, fmt.Errorf("artifact %q/%q has unsupported type %q", name, variant, artifact.Type)
			}
			if len(artifact.Identifiers) == 0 {
				return nil, fmt.Errorf("artifact %q/%q has no identifiers", name, variant)
			}

			artifact.Language = name
			artifact.Name = variant
			artifacts = append(artifacts, artifact)
		}
	}

	return artifacts, nil
}
