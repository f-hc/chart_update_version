// SPDX-License-Identifier: GPL-3.0-only
//
// Copyright (C) 2026 f-hc <207619282+f-hc@users.noreply.github.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, version 3 of the License.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ChartInfo holds the discovered chart information from an ArgoCD Application manifest.
type ChartInfo struct {
	File string // File path relative to the argoapps directory
	Repo string // ArtifactHub repository path (e.g., "cilium/cilium")
}

// discoverCharts scans a directory for ArgoCD Application manifests
// that have an "# artifacthub:" comment and returns the discovered charts.
func discoverCharts(dir string) ([]ChartInfo, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", dir)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve directory path: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read directory: %w", err)
	}

	var charts []ChartInfo

	for _, entry := range entries {
		if repo := getArtifactHubRepoFromEntry(dir, absDir, entry); repo != "" {
			charts = append(charts, ChartInfo{
				File: entry.Name(),
				Repo: repo,
			})
		}
	}

	return charts, nil
}

// getArtifactHubRepoFromEntry checks if a directory entry is an ArgoCD Application
// with an ArtifactHub repo comment and returns the repo path if found.
func getArtifactHubRepoFromEntry(dir, absDir string, entry os.DirEntry) string {
	if entry.IsDir() {
		return ""
	}

	name := entry.Name()
	if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
		return ""
	}

	path := filepath.Join(dir, name)

	// Verify path doesn't escape directory (safety check)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	if !strings.HasPrefix(absPath, absDir+string(os.PathSeparator)) && absPath != absDir {
		return ""
	}

	repo, err := extractArtifactHubRepo(path)
	if err != nil {
		// Skip files that can't be parsed
		return ""
	}

	return repo
}

// extractArtifactHubRepo reads a YAML file and extracts the ArtifactHub repo
// from the first Application document that has the comment.
func extractArtifactHubRepo(path string) (string, error) {
	docs, err := readYAMLDocuments(path)
	if err != nil {
		return "", err
	}

	for _, d := range docs {
		if kind(d) == "Application" {
			if repo := getArtifactHubRepo(d); repo != "" {
				return repo, nil
			}
		}
	}

	return "", nil
}
