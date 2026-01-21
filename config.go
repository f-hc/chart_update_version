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
	"slices"
	"strings"

	"github.com/BooleanCat/go-functional/v2/it"
	"gopkg.in/yaml.v3"
)

const (
	defaultArgoAppsDir = "argoapps"
	argoAppsDirEnvVar  = "UPDATE_VERSION_DIR"
)

// Config holds the application configuration.
type Config struct {
	Dir       string
	DryRun    bool
	CheckOnly bool
}

// ParseConfig parses command line arguments and environment variables to create a Config.
func ParseConfig(args []string, getEnv func(string) string) (Config, error) {
	cfg := defaultConfig()
	cfg = applyEnv(cfg, getEnv)

	cfg, err := parseArgs(cfg, args)
	if err != nil {
		return cfg, err
	}

	return validateConfig(cfg)
}

func defaultConfig() Config {
	return Config{
		Dir: defaultArgoAppsDir,
	}
}

func applyEnv(cfg Config, getEnv func(string) string) Config {
	if v := getEnv(argoAppsDirEnvVar); v != "" {
		cfg.Dir = v
	}
	return cfg
}

func parseArgs(cfg Config, args []string) (Config, error) {
	if len(args) == 0 {
		return cfg, nil
	}

	head, tail := args[0], args[1:]

	switch head {
	case "--dry-run", "-n":
		cfg.DryRun = true
		return parseArgs(cfg, tail)

	case "--check", "-C":
		cfg.CheckOnly = true
		return parseArgs(cfg, tail)

	case "--dir", "-d":
		if len(tail) == 0 {
			return cfg, fmt.Errorf("--dir requires a directory path")
		}
		cfg.Dir = tail[0]
		return parseArgs(cfg, tail[1:])

	case "--help", "-h":
		return cfg, fmt.Errorf("help requested")

	default:
		if strings.HasPrefix(head, "-test.") {
			return parseArgs(cfg, tail)
		}
		if strings.HasPrefix(head, "-") {
			return cfg, fmt.Errorf("unknown flag: %s", head)
		}
		// Ignore positional arguments for now, matching previous behavior
		return parseArgs(cfg, tail)
	}
}

func validateConfig(cfg Config) (Config, error) {
	if cfg.DryRun && cfg.CheckOnly {
		return cfg, fmt.Errorf("--dry-run and --check cannot be used together")
	}

	return cfg, nil
}

// ChartInfo holds the discovered chart information from an ArgoCD Application manifest.
type ChartInfo struct {
	File string // File path relative to the argoapps directory
	Repo string // ArtifactHub repository path (e.g., "cilium/cilium")
}

type DirReader func(name string) ([]os.DirEntry, error)
type FileStater func(name string) (os.FileInfo, error)

// MakeChartDiscoverer creates a function that scans a directory for ArgoCD Application manifests.
func MakeChartDiscoverer(
	stat FileStater,
	readDir DirReader,
	readYaml YAMLReader,
) func(dir string) ([]ChartInfo, error) {
	return func(dir string) ([]ChartInfo, error) {
		info, err := stat(dir)
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

		entries, err := readDir(dir)
		if err != nil {
			return nil, fmt.Errorf("cannot read directory: %w", err)
		}

		// Functional pipeline to discover charts
		// 1. Filter YAML files
		yamlFiles := it.Filter(slices.Values(entries), isYamlFile)

		// 2. Map to full path
		paths := it.Map(yamlFiles, func(e os.DirEntry) string {
			return filepath.Join(dir, e.Name())
		})

		// 3. Filter valid paths (security check)
		validPaths := it.Filter(paths, func(p string) bool {
			return isValidPath(absDir, p)
		})

		// 4. Map to ChartInfo
		chartInfos := it.Map(validPaths, func(p string) ChartInfo {
			return toChartInfo(readYaml, p, dir)
		})

		// 5. Filter valid charts (where Repo is found)
		validCharts := it.Filter(chartInfos, func(c ChartInfo) bool {
			return c.Repo != ""
		})

		return slices.Collect(validCharts), nil
	}
}

// isYamlFile checks if the directory entry is a YAML file.
func isYamlFile(entry os.DirEntry) bool {
	if entry.IsDir() {
		return false
	}
	name := entry.Name()
	return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
}

// isValidPath checks if the path is safe and within the base directory.
func isValidPath(absDir, path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	return strings.HasPrefix(absPath, absDir+string(os.PathSeparator)) || absPath == absDir
}

// toChartInfo extracts chart info from the file.
func toChartInfo(readYaml YAMLReader, path, baseDir string) ChartInfo {
	repo, err := extractArtifactHubRepo(readYaml, path)
	if err != nil {
		return ChartInfo{}
	}

	return ChartInfo{
		File: relativePath(baseDir, path),
		Repo: repo,
	}
}

func relativePath(base, target string) string {
	if rel, err := filepath.Rel(base, target); err == nil {
		return rel
	}
	return target
}

// extractArtifactHubRepo reads a YAML file and extracts the ArtifactHub repo
// from the first Application document that has the comment.
func extractArtifactHubRepo(readYaml YAMLReader, path string) (string, error) {
	docs, err := readYaml(path)
	if err != nil {
		return "", err
	}

	// Filter for Application nodes
	apps := it.Filter(slices.Values(docs), func(n *yaml.Node) bool {
		return kind(n) == "Application"
	})

	// Map to repo strings
	repos := it.Map(apps, getArtifactHubRepo)

	// Find first non-empty
	repo, found := it.Find(repos, func(s string) bool {
		return s != ""
	})

	if found {
		return repo, nil
	}
	return "", nil
}
