package main

import (
	"encoding/json"
	"fmt"
	"os"
)

const artifactsFile = "artifacts.json"

type Artifact struct {
	Name        string   `json:"-"`
	Description string   `json:"description"`
	Identifiers []string `json:"identifier"`
	Type        string   `json:"type"`
	Remove      string   `json:"rm"`
}

type artifactManifest struct {
	Version   string                         `json:"version"`
	Artifacts map[string]map[string]Artifact `json:"artifacts"`
}

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

func readVersionFile() (string, error) {
	manifest, err := readManifest()
	if err != nil {
		return "", err
	}

	return manifest.Version, nil
}

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

			artifact.Name = fmt.Sprintf("%s (%s)", name, variant)
			artifacts = append(artifacts, artifact)
		}
	}

	return artifacts, nil
}
